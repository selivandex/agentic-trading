package analysis

import (
	"context"
	"time"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/domain/regime"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// RegimeDetector detects and updates market regime (trending, ranging, volatile)
// This helps agents adapt their strategies based on market conditions
type RegimeDetector struct {
	*workers.BaseWorker
	mdRepo     market_data.Repository
	regimeRepo regime.Repository
	kafka      *kafka.Producer
	symbols    []string
}

// NewRegimeDetector creates a new regime detector worker
func NewRegimeDetector(
	mdRepo market_data.Repository,
	regimeRepo regime.Repository,
	kafka *kafka.Producer,
	symbols []string,
	interval time.Duration,
	enabled bool,
) *RegimeDetector {
	return &RegimeDetector{
		BaseWorker: workers.NewBaseWorker("regime_detector", interval, enabled),
		mdRepo:     mdRepo,
		regimeRepo: regimeRepo,
		kafka:      kafka,
		symbols:    symbols,
	}
}

// Run executes one iteration of regime detection
func (rd *RegimeDetector) Run(ctx context.Context) error {
	rd.Log().Debug("Regime detector: starting iteration")

	if len(rd.symbols) == 0 {
		rd.Log().Warn("No symbols configured for regime detection")
		return nil
	}

	for _, symbol := range rd.symbols {
		if err := rd.detectRegime(ctx, symbol); err != nil {
			rd.Log().Error("Failed to detect regime",
				"symbol", symbol,
				"error", err,
			)
			continue
		}
	}

	rd.Log().Debug("Regime detection complete")

	return nil
}

// detectRegime detects the current market regime for a symbol
func (rd *RegimeDetector) detectRegime(ctx context.Context, symbol string) error {
	// Get OHLCV data for multiple timeframes
	// 1h candles for last 24 hours (24 candles)
	// 4h candles for last 7 days (42 candles)
	// 1d candles for last 30 days (30 candles)

	candles1h, err := rd.mdRepo.GetOHLCV(ctx, market_data.OHLCVQuery{
		Symbol:    symbol,
		Timeframe: "1h",
		Limit:     24,
	})
	if err != nil {
		return errors.Wrap(err, "failed to get 1h OHLCV")
	}

	candles4h, err := rd.mdRepo.GetOHLCV(ctx, market_data.OHLCVQuery{
		Symbol:    symbol,
		Timeframe: "4h",
		Limit:     42,
	})
	if err != nil {
		return errors.Wrap(err, "failed to get 4h OHLCV")
	}

	candles1d, err := rd.mdRepo.GetOHLCV(ctx, market_data.OHLCVQuery{
		Symbol:    symbol,
		Timeframe: "1d",
		Limit:     30,
	})
	if err != nil {
		return errors.Wrap(err, "failed to get 1d OHLCV")
	}

	if len(candles1h) < 10 || len(candles4h) < 10 || len(candles1d) < 10 {
		rd.Log().Warn("Insufficient data for regime detection", "symbol", symbol)
		return nil
	}

	// Calculate regime indicators
	adr := rd.calculateADR(candles1d)        // Average Daily Range
	volatility := rd.calculateVolatility(candles1h) // Price volatility
	trendStrength := rd.calculateTrendStrength(candles4h)

	// Determine regime based on indicators
	regimeType := rd.classifyRegime(adr, volatility, trendStrength)

	rd.Log().Info("Regime detected",
		"symbol", symbol,
		"regime", regimeType,
		"adr", adr,
		"volatility", volatility,
		"trend_strength", trendStrength,
	)

	// Store regime to database
	newRegime := &regime.MarketRegime{
		Symbol:     symbol,
		Timestamp:  time.Now(),
		Regime:     regimeType,
		Confidence: rd.calculateConfidence(adr, volatility, trendStrength),
		Volatility: rd.classifyVolatility(volatility),
		Trend:      rd.classifyTrend(trendStrength),
		ATR14:      adr,
		ADX:        abs(trendStrength),
		BBWidth:    0, // TODO: Calculate Bollinger Band width
		Volume24h:  0, // TODO: Get 24h volume
		VolumeChange: 0, // TODO: Calculate volume change
	}

	if err := rd.regimeRepo.Store(ctx, newRegime); err != nil {
		return errors.Wrap(err, "failed to store regime")
	}

	// Publish regime change event
	rd.publishRegimeEvent(ctx, symbol, regimeType, volatility, trendStrength)

	return nil
}

// calculateADR calculates Average Daily Range as percentage
func (rd *RegimeDetector) calculateADR(candles []market_data.OHLCV) float64 {
	if len(candles) == 0 {
		return 0
	}

	sum := 0.0
	for _, candle := range candles {
		dailyRange := ((candle.High - candle.Low) / candle.Low) * 100
		sum += dailyRange
	}

	return sum / float64(len(candles))
}

// calculateVolatility calculates price volatility (standard deviation)
func (rd *RegimeDetector) calculateVolatility(candles []market_data.OHLCV) float64 {
	if len(candles) < 2 {
		return 0
	}

	// Calculate returns
	returns := make([]float64, len(candles)-1)
	for i := 1; i < len(candles); i++ {
		returns[i-1] = (candles[i].Close - candles[i-1].Close) / candles[i-1].Close
	}

	// Calculate mean
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	// Calculate variance
	variance := 0.0
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns))

	// Standard deviation as percentage
	return sqrt(variance) * 100
}

// calculateTrendStrength calculates trend strength using linear regression slope
func (rd *RegimeDetector) calculateTrendStrength(candles []market_data.OHLCV) float64 {
	if len(candles) < 2 {
		return 0
	}

	// Simple linear regression: y = mx + b
	n := float64(len(candles))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0

	for i, candle := range candles {
		x := float64(i)
		y := candle.Close

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Calculate slope (m)
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// Normalize slope relative to price
	avgPrice := sumY / n
	normalizedSlope := (slope / avgPrice) * 100

	return normalizedSlope
}

// classifyRegime determines regime type based on indicators
func (rd *RegimeDetector) classifyRegime(adr, volatility, trendStrength float64) regime.RegimeType {
	// High volatility (ADR > 5% or volatility > 3%)
	if adr > 5.0 || volatility > 3.0 {
		return regime.RegimeVolatile
	}

	// Strong trend (trend strength > 2% or < -2%)
	if abs(trendStrength) > 2.0 {
		if trendStrength > 0 {
			return regime.RegimeTrendUp
		}
		return regime.RegimeTrendDown
	}

	// Weak trend or ranging
	if abs(trendStrength) < 0.5 {
		return regime.RegimeRange
	}

	// Moderate trend
	if trendStrength > 0 {
		return regime.RegimeTrendUp
	}
	return regime.RegimeTrendDown
}

// calculateConfidence calculates confidence score (0-1) for regime detection
func (rd *RegimeDetector) calculateConfidence(adr, volatility, trendStrength float64) float64 {
	// Simple heuristic: higher values = more confident
	// This is simplified - in production you'd use more sophisticated methods

	confidence := 0.5 // Base confidence

	// Strong trend increases confidence
	if abs(trendStrength) > 3.0 {
		confidence += 0.3
	} else if abs(trendStrength) > 1.5 {
		confidence += 0.2
	}

	// Clear volatility signals increase confidence
	if volatility > 4.0 || volatility < 1.0 {
		confidence += 0.2
	}

	// Ensure confidence is between 0 and 1
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// Event structures

type RegimeChangeEvent struct {
	Symbol        string    `json:"symbol"`
	RegimeType    string    `json:"regime_type"`
	Confidence    float64   `json:"confidence"`
	Volatility    float64   `json:"volatility"`
	TrendStrength float64   `json:"trend_strength"`
	Timestamp     time.Time `json:"timestamp"`
}

func (rd *RegimeDetector) publishRegimeEvent(
	ctx context.Context,
	symbol string,
	regimeType regime.RegimeType,
	volatility, trendStrength float64,
) {
	event := RegimeChangeEvent{
		Symbol:        symbol,
		RegimeType:    regimeType.String(),
		Confidence:    rd.calculateConfidence(0, volatility, trendStrength),
		Volatility:    volatility,
		TrendStrength: trendStrength,
		Timestamp:     time.Now(),
	}

	if err := rd.kafka.Publish(ctx, "market.regime_change", symbol, event); err != nil {
		rd.Log().Error("Failed to publish regime change event", "error", err)
	}
}

// classifyVolatility classifies volatility level
func (rd *RegimeDetector) classifyVolatility(volatility float64) regime.VolLevel {
	if volatility > 5.0 {
		return regime.VolExtreme
	} else if volatility > 3.0 {
		return regime.VolHigh
	} else if volatility > 1.5 {
		return regime.VolMedium
	}
	return regime.VolLow
}

// classifyTrend classifies trend direction
func (rd *RegimeDetector) classifyTrend(trendStrength float64) regime.TrendType {
	if trendStrength > 1.0 {
		return regime.TrendBullish
	} else if trendStrength < -1.0 {
		return regime.TrendBearish
	}
	return regime.TrendNeutral
}

// sqrt returns square root (simplified implementation)
func sqrt(x float64) float64 {
	if x == 0 {
		return 0
	}
	// Newton's method approximation
	z := x
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}

