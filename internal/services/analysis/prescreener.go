package analysis

import (
	"context"
	"fmt"
	"math"
	"time"

	"prometheus/internal/domain/market_data"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// PreScreenResult contains the result of pre-screening analysis
type PreScreenResult struct {
	ShouldAnalyze bool
	SkipReason    string
	Confidence    float64
	Metrics       PreScreenMetrics
}

// PreScreenMetrics tracks metrics for monitoring
type PreScreenMetrics struct {
	PriceChangePct     float64
	VolumePct          float64
	ATRPct             float64
	TimeSinceLastCheck time.Duration
	OrderBookImbalance float64
}

// PreScreenConfig contains configuration for pre-screening
type PreScreenConfig struct {
	Enabled bool

	// Thresholds
	MinPriceChangePct float64       // Skip if price change < this (default: 0.2%)
	MinVolumePct      float64       // Skip if volume < this % of 24h avg (default: 50%)
	MinATRPct         float64       // Skip if ATR < this % (default: 1%)
	CooldownDuration  time.Duration // Min time between analyses (default: 2h)

	// Feature flags
	CheckPriceMovement   bool
	CheckVolume          bool
	CheckVolatility      bool
	CheckOrderBook       bool
	CheckCooldown        bool
}

// DefaultPreScreenConfig returns default configuration
func DefaultPreScreenConfig() PreScreenConfig {
	return PreScreenConfig{
		Enabled:              true,
		MinPriceChangePct:    0.002, // 0.2%
		MinVolumePct:         0.50,  // 50%
		MinATRPct:            0.01,  // 1%
		CooldownDuration:     2 * time.Hour,
		CheckPriceMovement:   true,
		CheckVolume:          true,
		CheckVolatility:      true,
		CheckOrderBook:       false, // Optional, can be expensive
		CheckCooldown:        true,
	}
}

// PreScreener performs lightweight checks before expensive LLM analysis
type PreScreener struct {
	config         PreScreenConfig
	marketDataRepo market_data.Repository
	lastCheckTimes map[string]time.Time // symbol -> last analysis time
	log            *logger.Logger
}

// NewPreScreener creates a new pre-screener
func NewPreScreener(
	config PreScreenConfig,
	marketDataRepo market_data.Repository,
) *PreScreener {
	return &PreScreener{
		config:         config,
		marketDataRepo: marketDataRepo,
		lastCheckTimes: make(map[string]time.Time),
		log:            logger.Get().With("component", "prescreener"),
	}
}

// ShouldAnalyze determines if a symbol should be analyzed
// Returns true if analysis should proceed, false if it should be skipped
func (ps *PreScreener) ShouldAnalyze(ctx context.Context, exchange, symbol string) (*PreScreenResult, error) {
	if !ps.config.Enabled {
		return &PreScreenResult{
			ShouldAnalyze: true,
			SkipReason:    "",
			Confidence:    1.0,
		}, nil
	}

	result := &PreScreenResult{
		ShouldAnalyze: true,
		Confidence:    1.0,
		Metrics:       PreScreenMetrics{},
	}

	// Check cooldown period
	if ps.config.CheckCooldown {
		if !ps.checkCooldown(symbol, &result.Metrics) {
			result.ShouldAnalyze = false
			result.SkipReason = fmt.Sprintf("cooldown period active (< %v since last analysis)", ps.config.CooldownDuration)
			result.Confidence = 0.0
			return result, nil
		}
	}

	// Get latest ticker for current price and 24h volume
	ticker, err := ps.marketDataRepo.GetLatestTicker(ctx, exchange, symbol)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ticker")
	}
	if ticker == nil {
		// No data available, allow analysis
		return result, nil
	}

	// Check price movement
	if ps.config.CheckPriceMovement {
		priceChangePct := math.Abs(ticker.Change24h)
		result.Metrics.PriceChangePct = priceChangePct

		if priceChangePct < ps.config.MinPriceChangePct {
			result.ShouldAnalyze = false
			result.SkipReason = fmt.Sprintf("insufficient price movement (%.3f%% < %.3f%%)", priceChangePct*100, ps.config.MinPriceChangePct*100)
			result.Confidence = priceChangePct / ps.config.MinPriceChangePct
			return result, nil
		}
	}

	// Check volume
	if ps.config.CheckVolume {
		// Get recent candles to calculate average volume
		candles, err := ps.marketDataRepo.GetLatestOHLCV(ctx, exchange, symbol, "1h", 24)
		if err != nil {
			ps.log.Warn("Failed to get OHLCV for volume check", "error", err)
		} else if len(candles) > 0 {
			avgVolume := ps.calculateAverageVolume(candles)
			currentVolume := ticker.Volume24h

			if avgVolume > 0 {
				volumePct := currentVolume / avgVolume
				result.Metrics.VolumePct = volumePct

				if volumePct < ps.config.MinVolumePct {
					result.ShouldAnalyze = false
					result.SkipReason = fmt.Sprintf("low volume (%.1f%% of average)", volumePct*100)
					result.Confidence = volumePct / ps.config.MinVolumePct
					return result, nil
				}
			}
		}
	}

	// Check volatility (ATR)
	if ps.config.CheckVolatility {
		candles, err := ps.marketDataRepo.GetLatestOHLCV(ctx, exchange, symbol, "1h", 14)
		if err != nil {
			ps.log.Warn("Failed to get OHLCV for ATR check", "error", err)
		} else if len(candles) >= 14 {
			atrPct := ps.calculateATR(candles, ticker.Price)
			result.Metrics.ATRPct = atrPct

			if atrPct < ps.config.MinATRPct {
				result.ShouldAnalyze = false
				result.SkipReason = fmt.Sprintf("extremely low volatility (ATR %.2f%% < %.2f%%)", atrPct*100, ps.config.MinATRPct*100)
				result.Confidence = atrPct / ps.config.MinATRPct
				return result, nil
			}
		}
	}

	// Check order book (optional, can be expensive)
	if ps.config.CheckOrderBook {
		orderbook, err := ps.marketDataRepo.GetLatestOrderBook(ctx, exchange, symbol)
		if err != nil {
			ps.log.Warn("Failed to get orderbook for imbalance check", "error", err)
		} else if orderbook != nil {
			imbalance := ps.calculateOrderBookImbalance(orderbook)
			result.Metrics.OrderBookImbalance = imbalance

			// Store imbalance but don't skip based on it (just for metrics)
		}
	}

	// Update last check time
	ps.lastCheckTimes[symbol] = time.Now()

	return result, nil
}

// checkCooldown checks if enough time has passed since last analysis
func (ps *PreScreener) checkCooldown(symbol string, metrics *PreScreenMetrics) bool {
	lastCheck, exists := ps.lastCheckTimes[symbol]
	if !exists {
		return true // First check, allow
	}

	timeSince := time.Since(lastCheck)
	metrics.TimeSinceLastCheck = timeSince

	return timeSince >= ps.config.CooldownDuration
}

// calculateAverageVolume calculates average volume from candles
func (ps *PreScreener) calculateAverageVolume(candles []market_data.OHLCV) float64 {
	if len(candles) == 0 {
		return 0
	}

	sum := 0.0
	for _, candle := range candles {
		sum += candle.Volume
	}

	return sum / float64(len(candles))
}

// calculateATR calculates Average True Range as percentage of price
func (ps *PreScreener) calculateATR(candles []market_data.OHLCV, currentPrice float64) float64 {
	if len(candles) < 2 || currentPrice == 0 {
		return 0
	}

	// Calculate True Range for each candle
	trSum := 0.0
	for i := 1; i < len(candles); i++ {
		high := candles[i].High
		low := candles[i].Low
		prevClose := candles[i-1].Close

		tr := math.Max(
			high-low,
			math.Max(
				math.Abs(high-prevClose),
				math.Abs(low-prevClose),
			),
		)
		trSum += tr
	}

	atr := trSum / float64(len(candles)-1)
	return atr / currentPrice // Return as percentage
}

// calculateOrderBookImbalance calculates bid/ask imbalance
func (ps *PreScreener) calculateOrderBookImbalance(orderbook *market_data.OrderBookSnapshot) float64 {
	if orderbook.BidDepth+orderbook.AskDepth == 0 {
		return 0
	}

	// Imbalance = (Bid - Ask) / (Bid + Ask)
	// Range: -1 (all asks) to +1 (all bids)
	return (orderbook.BidDepth - orderbook.AskDepth) / (orderbook.BidDepth + orderbook.AskDepth)
}

// RecordAnalysis records that an analysis was performed for a symbol
func (ps *PreScreener) RecordAnalysis(symbol string) {
	ps.lastCheckTimes[symbol] = time.Now()
}

// GetMetrics returns current metrics for monitoring
func (ps *PreScreener) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"enabled":         ps.config.Enabled,
		"symbols_tracked": len(ps.lastCheckTimes),
		"config": map[string]interface{}{
			"min_price_change_pct": ps.config.MinPriceChangePct,
			"min_volume_pct":       ps.config.MinVolumePct,
			"min_atr_pct":          ps.config.MinATRPct,
			"cooldown_duration":    ps.config.CooldownDuration.String(),
		},
	}
}

