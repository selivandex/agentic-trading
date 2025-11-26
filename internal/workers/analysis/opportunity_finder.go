package analysis

import (
	"context"
	"time"

	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/domain/trading_pair"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// OpportunityFinder scans markets for trading setups and opportunities
// This worker runs more frequently than market_scanner to catch quick opportunities
type OpportunityFinder struct {
	*workers.BaseWorker
	mdRepo          market_data.Repository
	tradingPairRepo trading_pair.Repository
	exchFactory     exchanges.Factory
	kafka           *kafka.Producer
	symbols         []string // List of symbols to monitor
}

// NewOpportunityFinder creates a new opportunity finder worker
func NewOpportunityFinder(
	mdRepo market_data.Repository,
	tradingPairRepo trading_pair.Repository,
	exchFactory exchanges.Factory,
	kafka *kafka.Producer,
	symbols []string,
	interval time.Duration,
	enabled bool,
) *OpportunityFinder {
	return &OpportunityFinder{
		BaseWorker:      workers.NewBaseWorker("opportunity_finder", interval, enabled),
		mdRepo:          mdRepo,
		tradingPairRepo: tradingPairRepo,
		exchFactory:     exchFactory,
		kafka:           kafka,
		symbols:         symbols,
	}
}

// Run executes one iteration of opportunity finding
// Scans for:
// - Strong breakouts with volume confirmation
// - Significant price movements (>2% in 1m)
// - Large liquidations (potential reversals)
// - Funding rate extremes
// - Unusual volume spikes
func (of *OpportunityFinder) Run(ctx context.Context) error {
	of.Log().Debug("Opportunity finder: starting iteration")

	if len(of.symbols) == 0 {
		of.Log().Warn("No symbols configured for opportunity finding")
		return nil
	}

	opportunities := 0

	for _, symbol := range of.symbols {
		// Check for opportunities on this symbol
		found, err := of.findOpportunities(ctx, symbol)
		if err != nil {
			of.Log().Error("Failed to find opportunities",
				"symbol", symbol,
				"error", err,
			)
			continue
		}

		if found {
			opportunities++
		}
	}

	of.Log().Debug("Opportunity finding complete", "opportunities_found", opportunities)

	return nil
}

// findOpportunities checks a single symbol for trading opportunities
func (of *OpportunityFinder) findOpportunities(ctx context.Context, symbol string) (bool, error) {
	// Get recent OHLCV data (last 60 candles of 1m)
	candles, err := of.mdRepo.GetOHLCV(ctx, market_data.OHLCVQuery{
		Symbol:    symbol,
		Timeframe: "1m",
		Limit:     60,
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to get OHLCV data")
	}

	if len(candles) < 2 {
		return false, nil // Not enough data
	}

	// Get latest two candles
	current := candles[len(candles)-1]
	previous := candles[len(candles)-2]

	// Check for strong price movement (>2% in 1m)
	priceChange := ((current.Close - previous.Close) / previous.Close) * 100

	if abs(priceChange) >= 2.0 {
		// Strong movement detected
		direction := "up"
		if priceChange < 0 {
			direction = "down"
		}

		of.Log().Info("Strong price movement detected",
			"symbol", symbol,
			"change_pct", priceChange,
			"direction", direction,
			"current_price", current.Close,
		)

		// Check if volume confirms the move
		avgVolume := of.calculateAvgVolume(candles[:len(candles)-1])
		volumeRatio := current.Volume / avgVolume

		if volumeRatio >= 2.0 {
			// Volume confirmation - this is a strong signal
			of.Log().Info("Volume confirmation",
				"symbol", symbol,
				"volume_ratio", volumeRatio,
				"avg_volume", avgVolume,
				"current_volume", current.Volume,
			)

			// Publish opportunity event
			of.publishOpportunityEvent(ctx, symbol, "breakout", priceChange, volumeRatio)
			return true, nil
		}
	}

	// TODO: Add more opportunity detection patterns:
	// - RSI divergence
	// - Support/resistance breakouts
	// - Liquidity zones
	// - Fair value gaps
	// - Order flow imbalances

	return false, nil
}

// calculateAvgVolume calculates average volume from candles
func (of *OpportunityFinder) calculateAvgVolume(candles []market_data.OHLCV) float64 {
	if len(candles) == 0 {
		return 0
	}

	sum := 0.0
	for _, candle := range candles {
		sum += candle.Volume
	}

	return sum / float64(len(candles))
}

// Event structures

type OpportunityEvent struct {
	Symbol       string    `json:"symbol"`
	Type         string    `json:"type"`          // breakout, reversal, divergence, etc
	Direction    string    `json:"direction"`     // up, down
	PriceChange  float64   `json:"price_change"`  // percentage
	VolumeRatio  float64   `json:"volume_ratio"`  // vs average
	CurrentPrice float64   `json:"current_price"` // current price
	Timestamp    time.Time `json:"timestamp"`
}

func (of *OpportunityFinder) publishOpportunityEvent(
	ctx context.Context,
	symbol, opportunityType string,
	priceChange, volumeRatio float64,
) {
	direction := "up"
	if priceChange < 0 {
		direction = "down"
	}

	event := OpportunityEvent{
		Symbol:       symbol,
		Type:         opportunityType,
		Direction:    direction,
		PriceChange:  priceChange,
		VolumeRatio:  volumeRatio,
		CurrentPrice: 0, // TODO: Get from latest candle
		Timestamp:    time.Now(),
	}

	if err := of.kafka.Publish(ctx, "market.opportunity_found", symbol, event); err != nil {
		of.Log().Error("Failed to publish opportunity event", "error", err)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
