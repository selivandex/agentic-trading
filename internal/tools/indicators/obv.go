package indicators

import (
	"context"

	"github.com/markcheno/go-talib"

	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewOBVTool computes On-Balance Volume using ta-lib
func NewOBVTool(deps shared.Deps) tool.Tool {
	return functiontool.New("obv", "On-Balance Volume", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		// Load candles
		candles, err := loadCandles(ctx, deps, args, 100)
		if err != nil {
			return nil, err
		}

		if err := ValidateMinLength(candles, 2, "OBV"); err != nil {
			return nil, err
		}

		// Prepare data for ta-lib
		data, err := PrepareData(candles)
		if err != nil {
			return nil, err
		}

		// Calculate OBV using ta-lib
		obv := talib.Obv(data.Close, data.Volume)

		// Get latest value
		value, err := GetLastValue(obv)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get OBV value")
		}

		// Get previous values for trend analysis
		lastNValues, err := GetLastNValues(obv, 10)
		if err != nil {
			return nil, err
		}

		// Analyze OBV trend
		trend := analyzeTrend(lastNValues)

		// OBV divergence analysis
		// Rising OBV + rising price = confirmation (bullish)
		// Falling OBV + rising price = bearish divergence (warning)
		// Rising OBV + falling price = bullish divergence (reversal signal)
		priceChange := candles[0].Close - candles[9].Close
		obvChange := lastNValues[len(lastNValues)-1] - lastNValues[0]

		divergence := "none"
		if priceChange > 0 && obvChange < 0 {
			divergence = "bearish" // Price up, volume down
		} else if priceChange < 0 && obvChange > 0 {
			divergence = "bullish" // Price down, volume up
		}

		signal := "neutral"
		if trend == "uptrend" && divergence == "none" {
			signal = "bullish_confirmation"
		} else if trend == "downtrend" && divergence == "none" {
			signal = "bearish_confirmation"
		} else if divergence == "bullish" {
			signal = "bullish_reversal"
		} else if divergence == "bearish" {
			signal = "bearish_reversal"
		}

		return map[string]interface{}{
			"value":      value,
			"trend":      trend,
			"divergence": divergence,
			"signal":     signal,
		}, nil
	})
}

// analyzeTrend determines trend from series of values
func analyzeTrend(values []float64) string {
	if len(values) < 2 {
		return "unknown"
	}

	// Simple linear regression slope
	n := len(values)
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0

	for i, y := range values {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (float64(n)*sumXY - sumX*sumY) / (float64(n)*sumX2 - sumX*sumX)

	if slope > 0.5 {
		return "uptrend"
	} else if slope < -0.5 {
		return "downtrend"
	}

	return "sideways"
}
