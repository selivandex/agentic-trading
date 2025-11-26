package indicators
import (
	"time"
	"github.com/markcheno/go-talib"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
	"google.golang.org/adk/tool"
)
// NewSupertrendTool computes Supertrend indicator
// Supertrend = ATR-based trailing stop indicator
// Formula:
// - Basic Upperband = (HIGH + LOW) / 2 + (Multiplier * ATR)
// - Basic Lowerband = (HIGH + LOW) / 2 - (Multiplier * ATR)
// - If close > prev supertrend, use lowerband, else use upperband
func NewSupertrendTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"supertrend",
		"Supertrend Indicator",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Load candles
			candles, err := loadCandles(ctx, deps, args, 200)
			if err != nil {
				return nil, err
			}
			atrPeriod := parseLimit(args["atr_period"], 10)
			multiplier := parseFloat(args["multiplier"], 3.0)
			if err := ValidateMinLength(candles, atrPeriod+1, "Supertrend"); err != nil {
				return nil, err
			}
			// Prepare data for ta-lib
			data, err := PrepareData(candles)
			if err != nil {
				return nil, err
			}
			// Calculate ATR using ta-lib
			atrValues := talib.Atr(data.High, data.Low, data.Close, atrPeriod)
			// Calculate Supertrend
			supertrend, trend := calculateSupertrend(data, atrValues, multiplier)
			// Get latest values
			supertrendValue, err := GetLastValue(supertrend)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get Supertrend value")
			}
			currentTrend, err := GetLastValue(trend)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get trend")
			}
			currentPrice := candles[0].Close
			atr, _ := GetLastValue(atrValues)
			// Determine signal
			signal := "neutral"
			trendName := "unknown"
			if currentTrend > 0 {
				trendName = "uptrend"
				signal = "bullish"
			} else if currentTrend < 0 {
				trendName = "downtrend"
				signal = "bearish"
			}
			// Check for trend change
			if len(trend) >= 2 {
				prevTrend := trend[len(trend)-2]
				if currentTrend > 0 && prevTrend <= 0 {
					signal = "trend_reversal_bullish"
				} else if currentTrend < 0 && prevTrend >= 0 {
					signal = "trend_reversal_bearish"
				}
			}
			return map[string]interface{}{
				"supertrend":    supertrendValue,
				"current_price": currentPrice,
				"atr":           atr,
				"trend":         trendName,
				"signal":        signal,
				"multiplier":    multiplier,
				"atr_period":    atrPeriod,
			}, nil
		},
		deps,
	).
		WithTimeout(15*time.Second).
		WithRetry(3, 500*time.Millisecond).
		Build()
}
// calculateSupertrend computes supertrend values and trend direction
func calculateSupertrend(data *TalibData, atr []float64, multiplier float64) ([]float64, []float64) {
	n := len(data.Close)
	supertrend := make([]float64, n)
	trend := make([]float64, n)
	// Initialize
	supertrend[0] = data.Close[0]
	trend[0] = 1
	for i := 1; i < n; i++ {
		// Calculate basic bands
		hl2 := (data.High[i] + data.Low[i]) / 2
		upperBand := hl2 + (multiplier * atr[i])
		lowerBand := hl2 - (multiplier * atr[i])
		// Determine supertrend value and trend
		if data.Close[i] > supertrend[i-1] {
			// Uptrend
			supertrend[i] = lowerBand
			if supertrend[i] < supertrend[i-1] {
				supertrend[i] = supertrend[i-1]
			}
			trend[i] = 1
		} else {
			// Downtrend
			supertrend[i] = upperBand
			if supertrend[i] > supertrend[i-1] {
				supertrend[i] = supertrend[i-1]
			}
			trend[i] = -1
		}
	}
	return supertrend, trend
}
