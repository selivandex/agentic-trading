package trading

import (
	"context"
	"time"

	"google.golang.org/adk/tool"

	"prometheus/internal/domain/position"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
)

// GetPortfolioSummaryOutput defines the output schema for portfolio summary
type GetPortfolioSummaryOutput struct {
	TotalEquityUSD      float64            `json:"total_equity_usd" description:"Total portfolio equity in USD"`
	TotalPnLUSD         float64            `json:"total_pnl_usd" description:"Total unrealized PnL in USD"`
	TotalPnLPercent     float64            `json:"total_pnl_percent" description:"Total PnL as percentage"`
	OpenPositionsCount  int                `json:"open_positions_count" description:"Number of open positions"`
	ExposureBySymbol    map[string]float64 `json:"exposure_by_symbol" description:"USD exposure per symbol"`
	ExposureByExchange  map[string]float64 `json:"exposure_by_exchange" description:"USD exposure per exchange"`
	LongExposure        float64            `json:"long_exposure" description:"Total long exposure USD"`
	ShortExposure       float64            `json:"short_exposure" description:"Total short exposure USD"`
	NetExposure         float64            `json:"net_exposure" description:"Net exposure (long - short)"`
	DailyPnL            float64            `json:"daily_pnl" description:"PnL today"`
}

// NewGetPortfolioSummaryTool creates a tool for retrieving aggregated portfolio summary
func NewGetPortfolioSummaryTool(deps shared.Deps) tool.Tool {
	handler := &getPortfolioSummaryHandler{
		positionRepo: deps.PositionRepo,
	}

	return shared.NewToolBuilder(
		"get_portfolio_summary",
		"Get aggregated portfolio summary with exposure breakdown and PnL statistics. "+
			"Use this to understand current portfolio state before making trading decisions.",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Get user ID from context
			userID, err := parseUUIDArg(args["user_id"], "user_id")
			if err != nil {
				if meta, ok := shared.MetadataFromContext(ctx); ok {
					userID = meta.UserID
				} else {
					return nil, err
				}
			}

			output, err := handler.execute(ctx, userID)
			if err != nil {
				return nil, err
			}

			// Convert to map for ADK tool response
			return map[string]interface{}{
				"total_equity_usd":      output.TotalEquityUSD,
				"total_pnl_usd":         output.TotalPnLUSD,
				"total_pnl_percent":     output.TotalPnLPercent,
				"open_positions_count":  output.OpenPositionsCount,
				"exposure_by_symbol":    output.ExposureBySymbol,
				"exposure_by_exchange":  output.ExposureByExchange,
				"long_exposure":         output.LongExposure,
				"short_exposure":        output.ShortExposure,
				"net_exposure":          output.NetExposure,
				"daily_pnl":             output.DailyPnL,
			}, nil
		},
		deps,
	).
		WithTimeout(10 * time.Second).
		WithRetry(3, 500*time.Millisecond).
		Build()
}

type getPortfolioSummaryHandler struct {
	positionRepo position.Repository
}

func (h *getPortfolioSummaryHandler) execute(ctx context.Context, userID interface{}) (*GetPortfolioSummaryOutput, error) {
	// Convert userID to UUID
	uid, err := toUUID(userID)
	if err != nil {
		return nil, errors.Wrap(err, "invalid user_id")
	}

	// Get all open positions for user
	positions, err := h.positionRepo.GetOpenByUser(ctx, uid)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get open positions")
	}

	// Initialize aggregated metrics
	var totalEquity, totalPnL, longExp, shortExp, dailyPnL float64
	exposureBySymbol := make(map[string]float64)
	exposureByExchange := make(map[string]float64)

	// Calculate aggregated metrics
	for _, pos := range positions {
		// Convert decimals to float64
		currentPrice, _ := pos.CurrentPrice.Float64()
		size, _ := pos.Size.Float64()
		pnl, _ := pos.UnrealizedPnL.Float64()

		// Position value in USD
		positionValue := currentPrice * size
		totalEquity += positionValue
		totalPnL += pnl

		// Track exposure by symbol
		exposureBySymbol[pos.Symbol] += positionValue

		// Track exposure by exchange (use ExchangeAccountID as proxy for now)
		// TODO: Join with exchange_accounts table to get actual exchange name
		exchangeKey := pos.ExchangeAccountID.String()
		exposureByExchange[exchangeKey] += positionValue

		// Track long/short exposure
		if pos.Side.String() == "long" {
			longExp += positionValue
		} else {
			shortExp += positionValue
		}

		// Daily PnL (simplified - checks if position was opened today)
		if pos.OpenedAt.After(time.Now().Add(-24 * time.Hour)) {
			dailyPnL += pnl
		}
	}

	// Calculate total PnL percentage
	totalPnLPercent := 0.0
	if totalEquity > 0 {
		totalPnLPercent = (totalPnL / totalEquity) * 100
	}

	return &GetPortfolioSummaryOutput{
		TotalEquityUSD:     totalEquity,
		TotalPnLUSD:        totalPnL,
		TotalPnLPercent:    totalPnLPercent,
		OpenPositionsCount: len(positions),
		ExposureBySymbol:   exposureBySymbol,
		ExposureByExchange: exposureByExchange,
		LongExposure:       longExp,
		ShortExposure:      shortExp,
		NetExposure:        longExp - shortExp,
		DailyPnL:           dailyPnL,
	}, nil
}

