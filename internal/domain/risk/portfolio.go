package risk

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/position"
	"prometheus/pkg/logger"
)

// PortfolioChecker validates portfolio-level risk limits
type PortfolioChecker struct {
	maxTotalExposure decimal.Decimal // e.g., 0.5 = 50% of portfolio
	maxPerAssetPct   decimal.Decimal // e.g., 0.15 = 15% per asset
	maxCorrelatedExp decimal.Decimal // e.g., 0.3 = 30% correlated
	highCorrelation  decimal.Decimal // e.g., 0.7 threshold
	posRepo          position.Repository
	corrService      CorrelationService
	log              *logger.Logger
}

// PortfolioCheckInput contains data for portfolio validation
type PortfolioCheckInput struct {
	UserID     uuid.UUID
	NewSymbol  string
	NewSizeUSD decimal.Decimal
}

// PortfolioCheckResult contains portfolio validation result
type PortfolioCheckResult struct {
	Approved              bool
	Errors                []string
	Warnings              []string
	TotalExposurePct      decimal.Decimal
	AssetConcentrationPct decimal.Decimal
	CorrelatedExposurePct decimal.Decimal
	NumPositions          int
}

// NewPortfolioChecker creates a new portfolio checker
func NewPortfolioChecker(
	maxTotalExposure float64,
	maxPerAsset float64,
	maxCorrelated float64,
	posRepo position.Repository,
	corrService CorrelationService,
	log *logger.Logger,
) *PortfolioChecker {
	return &PortfolioChecker{
		maxTotalExposure: decimal.NewFromFloat(maxTotalExposure),
		maxPerAssetPct:   decimal.NewFromFloat(maxPerAsset),
		maxCorrelatedExp: decimal.NewFromFloat(maxCorrelated),
		highCorrelation:  decimal.NewFromFloat(0.7),
		posRepo:          posRepo,
		corrService:      corrService,
		log:              log,
	}
}

// Check validates portfolio risk limits for a new trade
func (c *PortfolioChecker) Check(ctx context.Context, input *PortfolioCheckInput) (*PortfolioCheckResult, error) {
	result := &PortfolioCheckResult{
		Approved: true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	// Get all open positions
	positions, err := c.posRepo.GetOpenByUser(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get open positions: %w", err)
	}

	result.NumPositions = len(positions)

	// Calculate total portfolio value and exposure
	totalValue := decimal.Zero
	totalExposure := decimal.Zero
	assetExposure := make(map[string]decimal.Decimal)

	for _, pos := range positions {
		posValue := pos.Size.Mul(pos.EntryPrice)
		totalValue = totalValue.Add(posValue)
		totalExposure = totalExposure.Add(posValue)

		if existing, ok := assetExposure[pos.Symbol]; ok {
			assetExposure[pos.Symbol] = existing.Add(posValue)
		} else {
			assetExposure[pos.Symbol] = posValue
		}
	}

	// Add new position to projections
	newTotalExposure := totalExposure.Add(input.NewSizeUSD)
	newTotalValue := totalValue.Add(input.NewSizeUSD)

	// Update asset exposure with new position
	if existing, ok := assetExposure[input.NewSymbol]; ok {
		assetExposure[input.NewSymbol] = existing.Add(input.NewSizeUSD)
	} else {
		assetExposure[input.NewSymbol] = input.NewSizeUSD
	}

	// 1. Total exposure check
	if !newTotalValue.IsZero() {
		exposurePct := newTotalExposure.Div(newTotalValue)
		result.TotalExposurePct = exposurePct

		if exposurePct.GreaterThan(c.maxTotalExposure) {
			result.Approved = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("max_exposure_exceeded: %.1f%% > %.1f%%",
					exposurePct.Mul(decimal.NewFromInt(100)).InexactFloat64(),
					c.maxTotalExposure.Mul(decimal.NewFromInt(100)).InexactFloat64(),
				))
		} else if exposurePct.GreaterThan(c.maxTotalExposure.Mul(decimal.NewFromFloat(0.9))) {
			// Warn if approaching limit (90%)
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("approaching_max_exposure: %.1f%%",
					exposurePct.Mul(decimal.NewFromInt(100)).InexactFloat64(),
				))
		}
	}

	// 2. Per-asset concentration check
	assetPct := input.NewSizeUSD
	if !newTotalValue.IsZero() {
		assetPct = assetExposure[input.NewSymbol].Div(newTotalValue)
	}
	result.AssetConcentrationPct = assetPct

	if assetPct.GreaterThan(c.maxPerAssetPct) {
		result.Approved = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("asset_concentration_exceeded: %s at %.1f%% > %.1f%%",
				input.NewSymbol,
				assetPct.Mul(decimal.NewFromInt(100)).InexactFloat64(),
				c.maxPerAssetPct.Mul(decimal.NewFromInt(100)).InexactFloat64(),
			))
	}

	// 3. Correlation check
	correlations, err := c.corrService.GetCorrelations(ctx, input.NewSymbol)
	if err == nil && len(correlations) > 0 {
		correlatedExposure := decimal.Zero
		for symbol, posExposure := range assetExposure {
			if corr, ok := correlations[symbol]; ok && corr.GreaterThan(c.highCorrelation) {
				// Highly correlated asset, add to correlated exposure
				correlatedExposure = correlatedExposure.Add(posExposure.Mul(corr))
			}
		}

		if !newTotalValue.IsZero() {
			correlatedPct := correlatedExposure.Div(newTotalValue)
			result.CorrelatedExposurePct = correlatedPct

			if correlatedPct.GreaterThan(c.maxCorrelatedExp) {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("high_correlation_exposure: %.1f%%",
						correlatedPct.Mul(decimal.NewFromInt(100)).InexactFloat64(),
					))
			}
		}
	} else if err != nil {
		c.log.Warnf("Failed to get correlations for %s: %v", input.NewSymbol, err)
	}

	// 4. Diversification check (warning if too concentrated)
	if len(assetExposure) == 1 && result.NumPositions > 0 {
		result.Warnings = append(result.Warnings, "portfolio_not_diversified: single asset")
	}

	return result, nil
}
