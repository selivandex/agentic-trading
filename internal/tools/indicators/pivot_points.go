package indicators

import (
	"context"

	"prometheus/internal/tools/shared"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewPivotPointsTool computes Pivot Points (Classic, Fibonacci, Woodie, Camarilla)
// Pivot points are support/resistance levels calculated from previous period's high/low/close
func NewPivotPointsTool(deps shared.Deps) tool.Tool {
	return functiontool.New("pivot_points", "Pivot Points Calculator", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		// Load candles (need at least 1 previous day candle)
		candles, err := loadCandles(ctx, deps, args, 10)
		if err != nil {
			return nil, err
		}

		if err := ValidateMinLength(candles, 2, "Pivot Points"); err != nil {
			return nil, err
		}

		// Use previous candle (yesterday's data) for calculation
		prevCandle := candles[1]
		currentPrice := candles[0].Close

		// Get pivot type (classic, fibonacci, woodie, camarilla)
		pivotType := "classic"
		if pt, ok := args["type"].(string); ok {
			pivotType = pt
		}

		var pivot, r1, r2, r3, s1, s2, s3 float64

		switch pivotType {
		case "fibonacci":
			pivot, r1, r2, r3, s1, s2, s3 = calculateFibonacciPivots(prevCandle.High, prevCandle.Low, prevCandle.Close)
		case "woodie":
			pivot, r1, r2, r3, s1, s2, s3 = calculateWoodiePivots(prevCandle.High, prevCandle.Low, prevCandle.Close, candles[0].Open)
		case "camarilla":
			pivot, r1, r2, r3, s1, s2, s3 = calculateCamarillaPivots(prevCandle.High, prevCandle.Low, prevCandle.Close)
		default: // classic
			pivot, r1, r2, r3, s1, s2, s3 = calculateClassicPivots(prevCandle.High, prevCandle.Low, prevCandle.Close)
		}

		// Determine current level
		currentLevel := "between_pivots"
		if currentPrice >= r3 {
			currentLevel = "above_r3"
		} else if currentPrice >= r2 {
			currentLevel = "above_r2"
		} else if currentPrice >= r1 {
			currentLevel = "above_r1"
		} else if currentPrice <= s3 {
			currentLevel = "below_s3"
		} else if currentPrice <= s2 {
			currentLevel = "below_s2"
		} else if currentPrice <= s1 {
			currentLevel = "below_s1"
		} else if currentPrice > pivot {
			currentLevel = "above_pivot"
		} else {
			currentLevel = "below_pivot"
		}

		// Generate signal
		signal := "neutral"
		if currentPrice > pivot && currentPrice < r1 {
			signal = "bullish"
		} else if currentPrice < pivot && currentPrice > s1 {
			signal = "bearish"
		}

		return map[string]interface{}{
			"pivot":         pivot,
			"r1":            r1,
			"r2":            r2,
			"r3":            r3,
			"s1":            s1,
			"s2":            s2,
			"s3":            s3,
			"current_price": currentPrice,
			"current_level": currentLevel,
			"signal":        signal,
			"type":          pivotType,
		}, nil
	})
}

// Classic Pivot Points
func calculateClassicPivots(high, low, close float64) (pivot, r1, r2, r3, s1, s2, s3 float64) {
	pivot = (high + low + close) / 3
	r1 = 2*pivot - low
	s1 = 2*pivot - high
	r2 = pivot + (high - low)
	s2 = pivot - (high - low)
	r3 = high + 2*(pivot-low)
	s3 = low - 2*(high-pivot)
	return
}

// Fibonacci Pivot Points
func calculateFibonacciPivots(high, low, close float64) (pivot, r1, r2, r3, s1, s2, s3 float64) {
	pivot = (high + low + close) / 3
	range_ := high - low
	r1 = pivot + 0.382*range_
	r2 = pivot + 0.618*range_
	r3 = pivot + 1.000*range_
	s1 = pivot - 0.382*range_
	s2 = pivot - 0.618*range_
	s3 = pivot - 1.000*range_
	return
}

// Woodie Pivot Points
func calculateWoodiePivots(high, low, close, open float64) (pivot, r1, r2, r3, s1, s2, s3 float64) {
	pivot = (high + low + 2*close) / 4
	r1 = 2*pivot - low
	s1 = 2*pivot - high
	r2 = pivot + (high - low)
	s2 = pivot - (high - low)
	r3 = high + 2*(pivot-low)
	s3 = low - 2*(high-pivot)
	return
}

// Camarilla Pivot Points
func calculateCamarillaPivots(high, low, close float64) (pivot, r1, r2, r3, s1, s2, s3 float64) {
	pivot = (high + low + close) / 3
	range_ := high - low
	r1 = close + range_*1.1/12
	r2 = close + range_*1.1/6
	r3 = close + range_*1.1/4
	s1 = close - range_*1.1/12
	s2 = close - range_*1.1/6
	s3 = close - range_*1.1/4
	return
}

