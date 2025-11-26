package indicators
import (
	"time"
	"github.com/markcheno/go-talib"
	"prometheus/internal/tools/shared"
	"google.golang.org/adk/tool"
)
// NewEMARibbonTool computes multiple EMAs (EMA Ribbon)
// Useful for trend strength and support/resistance levels
// Default periods: 9, 21, 55, 200 (common Fibonacci numbers)
func NewEMARibbonTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"ema_ribbon",
		"EMA Ribbon (Multiple EMAs)",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Load candles (need enough for 200 EMA)
			candles, err := loadCandles(ctx, deps, args, 250)
			if err != nil {
				return nil, err
			}
			if err := ValidateMinLength(candles, 200, "EMA Ribbon"); err != nil {
				return nil, err
			}
			// Prepare data for ta-lib
			closes, err := PrepareCloses(candles)
			if err != nil {
				return nil, err
			}
			// Calculate multiple EMAs
			ema9 := talib.Ema(closes, 9)
			ema21 := talib.Ema(closes, 21)
			ema55 := talib.Ema(closes, 55)
			ema200 := talib.Ema(closes, 200)
			// Get latest values
			e9, _ := GetLastValue(ema9)
			e21, _ := GetLastValue(ema21)
			e55, _ := GetLastValue(ema55)
			e200, _ := GetLastValue(ema200)
			currentPrice := candles[0].Close
			// Analyze EMA alignment (bullish when 9 > 21 > 55 > 200)
			alignment := "neutral"
			if e9 > e21 && e21 > e55 && e55 > e200 {
				alignment = "bullish" // Perfect ascending order
			} else if e9 < e21 && e21 < e55 && e55 < e200 {
				alignment = "bearish" // Perfect descending order
			} else if e9 > e21 && e21 > e55 {
				alignment = "short_term_bullish"
			} else if e9 < e21 && e21 < e55 {
				alignment = "short_term_bearish"
			}
			// Price position relative to ribbon
			pricePosition := "below_all"
			if currentPrice > e9 {
				pricePosition = "above_fast"
			}
			if currentPrice > e21 {
				pricePosition = "above_medium"
			}
			if currentPrice > e55 {
				pricePosition = "above_slow"
			}
			if currentPrice > e200 {
				pricePosition = "above_all"
			}
			// Trend strength based on EMA spacing
			spacing := ((e9 - e200) / e200) * 100
			trendStrength := "weak"
			if spacing > 5 || spacing < -5 {
				trendStrength = "strong"
			} else if spacing > 2 || spacing < -2 {
				trendStrength = "moderate"
			}
			// Generate signal
			signal := "neutral"
			if alignment == "bullish" && pricePosition == "above_all" {
				signal = "strong_bullish"
			} else if alignment == "bearish" && pricePosition == "below_all" {
				signal = "strong_bearish"
			} else if currentPrice > e21 && e21 > e55 {
				signal = "bullish"
			} else if currentPrice < e21 && e21 < e55 {
				signal = "bearish"
			}
			// Find support/resistance levels
			support := min4(e9, e21, e55, e200)
			resistance := max4(e9, e21, e55, e200)
			return map[string]interface{}{
				"ema_9":          e9,
				"ema_21":         e21,
				"ema_55":         e55,
				"ema_200":        e200,
				"current_price":  currentPrice,
				"alignment":      alignment,
				"price_position": pricePosition,
				"trend_strength": trendStrength,
				"signal":         signal,
				"support":        support,
				"resistance":     resistance,
			}, nil
		},
		deps,
	).
		WithTimeout(20*time.Second).
		WithRetry(3, 500*time.Millisecond).
		Build()
}
// Helper functions
func min4(a, b, c, d float64) float64 {
	min := a
	if b < min {
		min = b
	}
	if c < min {
		min = c
	}
	if d < min {
		min = d
	}
	return min
}
func max4(a, b, c, d float64) float64 {
	max := a
	if b > max {
		max = b
	}
	if c > max {
		max = c
	}
	if d > max {
		max = d
	}
	return max
}
