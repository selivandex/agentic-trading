package analysis

import (
	"context"
	"math"
	"time"

	"prometheus/internal/adapters/redis"
	"prometheus/internal/domain/market_data"
	"prometheus/pkg/logger"
)

// VolatilityRegime represents the current market volatility state
type VolatilityRegime string

const (
	RegimeHighVolatility   VolatilityRegime = "high"
	RegimeNormalVolatility VolatilityRegime = "normal"
	RegimeLowVolatility    VolatilityRegime = "low"
)

// IntervalConfig defines intervals for different volatility regimes
type IntervalConfig struct {
	HighVolatility   time.Duration // 15s (active market)
	NormalVolatility time.Duration // 30s (default)
	LowVolatility    time.Duration // 2m (quiet market)
}

// DefaultIntervalConfig returns default intervals
func DefaultIntervalConfig() IntervalConfig {
	return IntervalConfig{
		HighVolatility:   15 * time.Second,
		NormalVolatility: 30 * time.Second,
		LowVolatility:    2 * time.Minute,
	}
}

// RegimeThresholds defines thresholds for volatility classification
type RegimeThresholds struct {
	HighATRPct          float64 // >3% ATR = high volatility
	LowATRPct           float64 // <1% ATR = low volatility
	HighVolumePct       float64 // >150% avg volume = high activity
	LowVolumePct        float64 // <75% avg volume = low activity
	LargeTradeWindow    time.Duration
	LargeTradeThreshold float64 // USD value
}

// DefaultRegimeThresholds returns default thresholds
func DefaultRegimeThresholds() RegimeThresholds {
	return RegimeThresholds{
		HighATRPct:          0.03, // 3%
		LowATRPct:           0.01, // 1%
		HighVolumePct:       1.50, // 150%
		LowVolumePct:        0.75, // 75%
		LargeTradeWindow:    5 * time.Minute,
		LargeTradeThreshold: 100000, // $100K
	}
}

// IntervalAdjuster calculates volatility regime and adjusts intervals
type IntervalAdjuster struct {
	config         IntervalConfig
	thresholds     RegimeThresholds
	marketDataRepo market_data.Repository
	redisClient    *redis.Client
	log            *logger.Logger
}

// NewIntervalAdjuster creates a new interval adjuster
func NewIntervalAdjuster(
	config IntervalConfig,
	thresholds RegimeThresholds,
	marketDataRepo market_data.Repository,
	redisClient *redis.Client,
) *IntervalAdjuster {
	return &IntervalAdjuster{
		config:         config,
		thresholds:     thresholds,
		marketDataRepo: marketDataRepo,
		redisClient:    redisClient,
		log:            logger.Get().With("component", "interval_adjuster"),
	}
}

// GetInterval returns the appropriate interval for a symbol based on volatility regime
func (ia *IntervalAdjuster) GetInterval(ctx context.Context, exchange, symbol string) (time.Duration, error) {
	regime, err := ia.GetRegime(ctx, exchange, symbol)
	if err != nil {
		ia.log.Warn("Failed to get regime, using normal interval",
			"symbol", symbol,
			"error", err,
		)
		return ia.config.NormalVolatility, nil
	}

	switch regime {
	case RegimeHighVolatility:
		return ia.config.HighVolatility, nil
	case RegimeLowVolatility:
		return ia.config.LowVolatility, nil
	default:
		return ia.config.NormalVolatility, nil
	}
}

// GetRegime calculates the current volatility regime for a symbol
func (ia *IntervalAdjuster) GetRegime(ctx context.Context, exchange, symbol string) (VolatilityRegime, error) {
	// Try to get cached regime first
	cached, err := ia.getCachedRegime(ctx, exchange, symbol)
	if err == nil && cached != "" {
		return cached, nil
	}

	// Calculate regime
	regime, err := ia.calculateRegime(ctx, exchange, symbol)
	if err != nil {
		return RegimeNormalVolatility, err
	}

	// Cache the regime for 1 minute
	_ = ia.cacheRegime(ctx, exchange, symbol, regime)

	return regime, nil
}

// calculateRegime calculates volatility regime based on multiple factors
func (ia *IntervalAdjuster) calculateRegime(ctx context.Context, exchange, symbol string) (VolatilityRegime, error) {
	scores := 0 // Positive = high volatility, Negative = low volatility

	// Factor 1: ATR (Average True Range)
	ticker, err := ia.marketDataRepo.GetLatestTicker(ctx, exchange, symbol)
	if err != nil {
		return RegimeNormalVolatility, err
	}
	if ticker == nil {
		return RegimeNormalVolatility, nil
	}

	candles, err := ia.marketDataRepo.GetLatestOHLCV(ctx, exchange, symbol, "1h", 14)
	if err == nil && len(candles) >= 14 {
		atrPct := ia.calculateATR(candles, ticker.LastPrice)

		if atrPct > ia.thresholds.HighATRPct {
			scores += 2 // Strong indicator
		} else if atrPct < ia.thresholds.LowATRPct {
			scores -= 2
		}
	}

	// Factor 2: Volume
	candles24h, err := ia.marketDataRepo.GetLatestOHLCV(ctx, exchange, symbol, "1h", 24)
	if err == nil && len(candles24h) > 0 {
		avgVolume := ia.calculateAverageVolume(candles24h)
		if avgVolume > 0 && ticker.Volume > 0 {
			volumeRatio := ticker.Volume / avgVolume

			if volumeRatio > ia.thresholds.HighVolumePct {
				scores += 1
			} else if volumeRatio < ia.thresholds.LowVolumePct {
				scores -= 1
			}
		}
	}

	// Factor 3: Recent large trades
	trades, err := ia.marketDataRepo.GetRecentTrades(ctx, exchange, symbol, 100)
	if err == nil {
		hasLargeTrades := false
		cutoff := time.Now().Add(-ia.thresholds.LargeTradeWindow)

		for _, trade := range trades {
			if trade.Timestamp.After(cutoff) {
				tradeValue := trade.Price * trade.Quantity
				if tradeValue > ia.thresholds.LargeTradeThreshold {
					hasLargeTrades = true
					break
				}
			}
		}

		if hasLargeTrades {
			scores += 1
		}
	}

	// Determine regime based on total score
	if scores >= 2 {
		return RegimeHighVolatility, nil
	} else if scores <= -2 {
		return RegimeLowVolatility, nil
	}
	return RegimeNormalVolatility, nil
}

// calculateATR calculates Average True Range as percentage
func (ia *IntervalAdjuster) calculateATR(candles []market_data.OHLCV, currentPrice float64) float64 {
	if len(candles) < 2 || currentPrice == 0 {
		return 0
	}

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
	return atr / currentPrice
}

// calculateAverageVolume calculates average volume
func (ia *IntervalAdjuster) calculateAverageVolume(candles []market_data.OHLCV) float64 {
	if len(candles) == 0 {
		return 0
	}

	sum := 0.0
	for _, candle := range candles {
		sum += candle.Volume
	}
	return sum / float64(len(candles))
}

// getCachedRegime retrieves cached regime from Redis
func (ia *IntervalAdjuster) getCachedRegime(ctx context.Context, exchange, symbol string) (VolatilityRegime, error) {
	key := ia.regimeKey(exchange, symbol)
	val, err := ia.redisClient.GetString(ctx, key)
	if err != nil {
		return "", err
	}
	return VolatilityRegime(val), nil
}

// cacheRegime stores regime in Redis with 1-minute TTL
func (ia *IntervalAdjuster) cacheRegime(ctx context.Context, exchange, symbol string, regime VolatilityRegime) error {
	key := ia.regimeKey(exchange, symbol)
	return ia.redisClient.SetString(ctx, key, string(regime), time.Minute)
}

// regimeKey generates Redis key for regime storage
func (ia *IntervalAdjuster) regimeKey(exchange, symbol string) string {
	return "regime:" + exchange + ":" + symbol
}

// GetMetrics returns metrics for monitoring
func (ia *IntervalAdjuster) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"intervals": map[string]string{
			"high":   ia.config.HighVolatility.String(),
			"normal": ia.config.NormalVolatility.String(),
			"low":    ia.config.LowVolatility.String(),
		},
		"thresholds": map[string]interface{}{
			"high_atr_pct":          ia.thresholds.HighATRPct * 100,
			"low_atr_pct":           ia.thresholds.LowATRPct * 100,
			"high_vol_pct":          ia.thresholds.HighVolumePct * 100,
			"low_vol_pct":           ia.thresholds.LowVolumePct * 100,
			"large_trade_threshold": ia.thresholds.LargeTradeThreshold,
		},
	}
}
