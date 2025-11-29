package analysis

import (
	"context"
	"math"
	"time"

	"github.com/markcheno/go-talib"

	"prometheus/internal/domain/market_data"
	"prometheus/internal/domain/regime"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// FeatureExtractor extracts regime features from market data for ML model training/inference
type FeatureExtractor struct {
	*workers.BaseWorker
	mdRepo     market_data.Repository
	regimeRepo regime.Repository
	symbols    []string
}

// NewFeatureExtractor creates a new feature extractor worker
func NewFeatureExtractor(
	mdRepo market_data.Repository,
	regimeRepo regime.Repository,
	symbols []string,
	interval time.Duration,
	enabled bool,
) *FeatureExtractor {
	return &FeatureExtractor{
		BaseWorker: workers.NewBaseWorker("feature_extractor", interval, enabled),
		mdRepo:     mdRepo,
		regimeRepo: regimeRepo,
		symbols:    symbols,
	}
}

// Run executes one iteration of feature extraction
func (fe *FeatureExtractor) Run(ctx context.Context) error {
	fe.Log().Debug("Feature extractor: starting iteration")

	for _, symbol := range fe.symbols {
		if err := fe.extractFeatures(ctx, symbol); err != nil {
			fe.Log().Error("Failed to extract features", "symbol", symbol, "error", err)
			continue
		}
	}

	fe.Log().Debug("Feature extraction complete")
	return nil
}

// extractFeatures extracts and stores features for a single symbol
func (fe *FeatureExtractor) extractFeatures(ctx context.Context, symbol string) error {
	// Fetch OHLCV data (need 250+ candles for EMA200 + lookback)
	candles, err := fe.mdRepo.GetOHLCV(ctx, market_data.OHLCVQuery{
		Symbol:    symbol,
		Timeframe: "1h",
		Limit:     250,
	})
	if err != nil {
		return errors.Wrap(err, "failed to get OHLCV data")
	}

	if len(candles) < 200 {
		fe.Log().Warn("Insufficient data for feature extraction",
			"symbol", symbol,
			"candles", len(candles),
			"required", 200,
		)
		return nil
	}

	// Reverse candles for chronological order (talib expects oldest first)
	for i, j := 0, len(candles)-1; i < j; i, j = i+1, j-1 {
		candles[i], candles[j] = candles[j], candles[i]
	}

	// Convert to talib format
	closes := make([]float64, len(candles))
	highs := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	volumes := make([]float64, len(candles))

	for i, c := range candles {
		closes[i] = c.Close
		highs[i] = c.High
		lows[i] = c.Low
		volumes[i] = c.Volume
	}

	// Initialize feature struct
	features := &regime.Features{
		Symbol:    symbol,
		Timestamp: time.Now(),
	}

	// Extract volatility features
	fe.extractVolatilityFeatures(features, highs, lows, closes)

	// Extract trend features
	fe.extractTrendFeatures(features, highs, lows, closes)

	// Extract volume features
	fe.extractVolumeFeatures(features, closes, volumes)

	// Extract structure features
	fe.extractStructureFeatures(features, closes, highs, lows)

	// Extract cross-asset features (placeholder for Phase 5)
	features.BTCDominance = 0.0
	features.CorrelationTightness = 0.0

	// Extract derivatives features
	fe.extractDerivativesFeatures(ctx, features, symbol)

	// Store features
	if err := fe.regimeRepo.StoreFeatures(ctx, features); err != nil {
		return errors.Wrap(err, "failed to store features")
	}

	fe.Log().Debug("Features extracted and stored",
		"symbol", symbol,
		"atr", features.ATR14,
		"adx", features.ADX,
		"volume_24h", features.Volume24h,
	)

	return nil
}

// extractVolatilityFeatures calculates volatility-related features
func (fe *FeatureExtractor) extractVolatilityFeatures(features *regime.Features, highs, lows, closes []float64) {
	currentPrice := closes[len(closes)-1]

	// ATR (14-period)
	atrValues := talib.Atr(highs, lows, closes, 14)
	if len(atrValues) > 0 {
		features.ATR14 = atrValues[len(atrValues)-1]
		features.ATRPct = (features.ATR14 / currentPrice) * 100
	}

	// Bollinger Bands width
	upper, middle, lower := talib.BBands(closes, 20, 2.0, 2.0, talib.SMA)
	if len(upper) > 0 && len(middle) > 0 && len(lower) > 0 {
		lastUpper := upper[len(upper)-1]
		lastMiddle := middle[len(middle)-1]
		lastLower := lower[len(lower)-1]
		if lastMiddle > 0 {
			features.BBWidth = ((lastUpper - lastLower) / lastMiddle) * 100
		}
	}

	// Historical volatility (standard deviation of returns)
	features.HistoricalVol = fe.calculateHistoricalVolatility(closes)
}

// extractTrendFeatures calculates trend-related features
func (fe *FeatureExtractor) extractTrendFeatures(features *regime.Features, highs, lows, closes []float64) {
	// ADX (14-period)
	adxValues := talib.Adx(highs, lows, closes, 14)
	if len(adxValues) > 0 {
		features.ADX = adxValues[len(adxValues)-1]
	}

	// EMA values
	ema9 := talib.Ema(closes, 9)
	ema21 := talib.Ema(closes, 21)
	ema55 := talib.Ema(closes, 55)
	ema200 := talib.Ema(closes, 200)

	if len(ema9) > 0 {
		features.EMA9 = ema9[len(ema9)-1]
	}
	if len(ema21) > 0 {
		features.EMA21 = ema21[len(ema21)-1]
	}
	if len(ema55) > 0 {
		features.EMA55 = ema55[len(ema55)-1]
	}
	if len(ema200) > 0 {
		features.EMA200 = ema200[len(ema200)-1]
	}

	// EMA alignment
	features.EMAAlignment = fe.determineEMAAlignment(features.EMA9, features.EMA21, features.EMA55, features.EMA200)

	// Higher highs and lower lows count (last 20 periods)
	lookback := 20
	if len(highs) >= lookback && len(lows) >= lookback {
		features.HigherHighsCount, features.LowerLowsCount = fe.countSwings(
			highs[len(highs)-lookback:],
			lows[len(lows)-lookback:],
		)
	}
}

// extractVolumeFeatures calculates volume-related features
func (fe *FeatureExtractor) extractVolumeFeatures(features *regime.Features, closes, volumes []float64) {
	// 24h volume (sum of last 24 hours = last 24 candles for 1h timeframe)
	if len(volumes) >= 24 {
		features.Volume24h = fe.sumSlice(volumes[len(volumes)-24:])
	}

	// Volume change percentage (current vs average of previous 24h)
	if len(volumes) >= 48 {
		recentVolume := fe.sumSlice(volumes[len(volumes)-24:])
		previousVolume := fe.sumSlice(volumes[len(volumes)-48 : len(volumes)-24])
		if previousVolume > 0 {
			features.VolumeChangePct = ((recentVolume - previousVolume) / previousVolume) * 100
		}
	}

	// Volume-price divergence (simplified: correlation of volume and price changes)
	if len(closes) >= 24 && len(volumes) >= 24 {
		features.VolumePriceDivergence = fe.calculateVolumePriceDivergence(
			closes[len(closes)-24:],
			volumes[len(volumes)-24:],
		)
	}
}

// extractStructureFeatures calculates market structure features
func (fe *FeatureExtractor) extractStructureFeatures(features *regime.Features, closes, highs, lows []float64) {
	lookback := 50
	if len(closes) < lookback {
		return
	}

	recentCloses := closes[len(closes)-lookback:]
	recentHighs := highs[len(highs)-lookback:]
	recentLows := lows[len(lows)-lookback:]

	// Support breaks (count of times price broke below local low)
	features.SupportBreaks = fe.countSupportBreaks(recentCloses, recentLows)

	// Resistance breaks (count of times price broke above local high)
	features.ResistanceBreaks = fe.countResistanceBreaks(recentCloses, recentHighs)

	// Consolidation periods (count of sideways movements)
	features.ConsolidationPeriods = fe.detectConsolidation(recentCloses)
}

// extractDerivativesFeatures calculates derivatives-related features
func (fe *FeatureExtractor) extractDerivativesFeatures(ctx context.Context, features *regime.Features, symbol string) {
	// Funding rate (from latest ticker or funding rate data)
	// TODO: Implement GetLatestFundingRate method in market_data repository
	// fundingRate, err := fe.mdRepo.GetLatestFundingRate(ctx, "binance", symbol)
	// if err == nil && fundingRate != nil {
	// 	features.FundingRate = fundingRate.FundingRate
	// 	features.FundingRateMA = fundingRate.FundingRate
	// }

	// 24h liquidations (placeholder - need liquidation data)
	// TODO: Implement liquidation aggregation when liquidation collector is available
	features.Liquidations24h = 0.0
}

// Helper methods

func (fe *FeatureExtractor) calculateHistoricalVolatility(closes []float64) float64 {
	if len(closes) < 2 {
		return 0
	}

	// Calculate returns
	returns := make([]float64, len(closes)-1)
	for i := 1; i < len(closes); i++ {
		if closes[i-1] > 0 {
			returns[i-1] = (closes[i] - closes[i-1]) / closes[i-1]
		}
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

	// Return standard deviation as percentage
	return math.Sqrt(variance) * 100
}

func (fe *FeatureExtractor) determineEMAAlignment(ema9, ema21, ema55, ema200 float64) string {
	if ema9 > ema21 && ema21 > ema55 {
		if ema200 > 0 && ema55 > ema200 {
			return "strong_bullish"
		}
		return "bullish"
	} else if ema9 < ema21 && ema21 < ema55 {
		if ema200 > 0 && ema55 < ema200 {
			return "strong_bearish"
		}
		return "bearish"
	}
	return "neutral"
}

func (fe *FeatureExtractor) countSwings(highs, lows []float64) (uint8, uint8) {
	if len(highs) < 3 || len(lows) < 3 {
		return 0, 0
	}

	var higherHighs, lowerLows uint8

	// Count higher highs
	for i := 1; i < len(highs); i++ {
		if highs[i] > highs[i-1] {
			higherHighs++
		}
	}

	// Count lower lows
	for i := 1; i < len(lows); i++ {
		if lows[i] < lows[i-1] {
			lowerLows++
		}
	}

	return higherHighs, lowerLows
}

func (fe *FeatureExtractor) sumSlice(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum
}

func (fe *FeatureExtractor) calculateVolumePriceDivergence(closes, volumes []float64) float64 {
	if len(closes) != len(volumes) || len(closes) < 2 {
		return 0
	}

	// Calculate price changes
	priceChanges := make([]float64, len(closes)-1)
	for i := 1; i < len(closes); i++ {
		if closes[i-1] > 0 {
			priceChanges[i-1] = (closes[i] - closes[i-1]) / closes[i-1]
		}
	}

	// Calculate volume changes
	volumeChanges := make([]float64, len(volumes)-1)
	for i := 1; i < len(volumes); i++ {
		if volumes[i-1] > 0 {
			volumeChanges[i-1] = (volumes[i] - volumes[i-1]) / volumes[i-1]
		}
	}

	// Calculate simple correlation (Pearson)
	return fe.pearsonCorrelation(priceChanges, volumeChanges)
}

func (fe *FeatureExtractor) pearsonCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) == 0 {
		return 0
	}

	n := float64(len(x))

	// Calculate means
	sumX, sumY := 0.0, 0.0
	for i := range x {
		sumX += x[i]
		sumY += y[i]
	}
	meanX := sumX / n
	meanY := sumY / n

	// Calculate covariance and standard deviations
	cov, stdX, stdY := 0.0, 0.0, 0.0
	for i := range x {
		dx := x[i] - meanX
		dy := y[i] - meanY
		cov += dx * dy
		stdX += dx * dx
		stdY += dy * dy
	}

	if stdX == 0 || stdY == 0 {
		return 0
	}

	return cov / math.Sqrt(stdX*stdY)
}

func (fe *FeatureExtractor) countSupportBreaks(closes, lows []float64) uint8 {
	if len(closes) < 10 || len(lows) < 10 {
		return 0
	}

	var breaks uint8
	windowSize := 5

	for i := windowSize; i < len(closes); i++ {
		// Find local low in previous window
		localLow := lows[i-windowSize]
		for j := i - windowSize + 1; j < i; j++ {
			if lows[j] < localLow {
				localLow = lows[j]
			}
		}

		// Check if current close broke below local low
		if closes[i] < localLow*0.99 { // 1% threshold
			breaks++
		}
	}

	return breaks
}

func (fe *FeatureExtractor) countResistanceBreaks(closes, highs []float64) uint8 {
	if len(closes) < 10 || len(highs) < 10 {
		return 0
	}

	var breaks uint8
	windowSize := 5

	for i := windowSize; i < len(closes); i++ {
		// Find local high in previous window
		localHigh := highs[i-windowSize]
		for j := i - windowSize + 1; j < i; j++ {
			if highs[j] > localHigh {
				localHigh = highs[j]
			}
		}

		// Check if current close broke above local high
		if closes[i] > localHigh*1.01 { // 1% threshold
			breaks++
		}
	}

	return breaks
}

func (fe *FeatureExtractor) detectConsolidation(closes []float64) uint8 {
	if len(closes) < 10 {
		return 0
	}

	var consolidationPeriods uint8
	windowSize := 10
	threshold := 0.02 // 2% range for consolidation

	for i := windowSize; i < len(closes); i++ {
		window := closes[i-windowSize : i]

		// Find range
		min, max := window[0], window[0]
		for _, price := range window {
			if price < min {
				min = price
			}
			if price > max {
				max = price
			}
		}

		// Check if range is tight (consolidation)
		if max > 0 && (max-min)/max < threshold {
			consolidationPeriods++
		}
	}

	return consolidationPeriods
}
