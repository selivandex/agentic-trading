package indicators
import (
	"time"
	"prometheus/internal/tools/shared"
	"google.golang.org/adk/tool"
)
// NewIchimokuTool computes Ichimoku Cloud indicator
// Ichimoku components:
// - Tenkan-sen (Conversion Line): (9-period high + 9-period low) / 2
// - Kijun-sen (Base Line): (26-period high + 26-period low) / 2
// - Senkou Span A (Leading Span A): (Tenkan + Kijun) / 2, plotted 26 periods ahead
// - Senkou Span B (Leading Span B): (52-period high + 52-period low) / 2, plotted 26 periods ahead
// - Chikou Span (Lagging Span): Close price plotted 26 periods back
func NewIchimokuTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"ichimoku",
		"Ichimoku Cloud",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Load candles (need at least 52 + 26 = 78 for full calculation)
			candles, err := loadCandles(ctx, deps, args, 120)
			if err != nil {
				return nil, err
			}
			if err := ValidateMinLength(candles, 78, "Ichimoku"); err != nil {
				return nil, err
			}
			// Reverse to chronological order for easier calculation
			reversed := make([]struct{ high, low, close float64 }, len(candles))
			for i := range candles {
				idx := len(candles) - 1 - i
				reversed[idx].high = candles[i].High
				reversed[idx].low = candles[i].Low
				reversed[idx].close = candles[i].Close
			}
			// Calculate components
			tenkan := calculateMidpoint(reversed, len(reversed)-9, len(reversed), 9)
			kijun := calculateMidpoint(reversed, len(reversed)-26, len(reversed), 26)
			// Senkou Span A (leading span A)
			senkouA := (tenkan + kijun) / 2
			// Senkou Span B (leading span B)
			senkouB := calculateMidpoint(reversed, len(reversed)-52, len(reversed), 52)
			// Chikou Span (lagging span) - current close
			chikouSpan := candles[0].Close
			// Current price
			currentPrice := candles[0].Close
			// Determine cloud color and position
			cloudColor := "green"
			if senkouB > senkouA {
				cloudColor = "red"
			}
			// Determine price position relative to cloud
			pricePosition := "in_cloud"
			if currentPrice > senkouA && currentPrice > senkouB {
				pricePosition = "above_cloud"
			} else if currentPrice < senkouA && currentPrice < senkouB {
				pricePosition = "below_cloud"
			}
			// Generate signal
			signal := "neutral"
			if pricePosition == "above_cloud" && cloudColor == "green" {
				signal = "strong_bullish"
			} else if pricePosition == "above_cloud" && currentPrice > tenkan && currentPrice > kijun {
				signal = "bullish"
			} else if pricePosition == "below_cloud" && cloudColor == "red" {
				signal = "strong_bearish"
			} else if pricePosition == "below_cloud" && currentPrice < tenkan && currentPrice < kijun {
				signal = "bearish"
			}
			return map[string]interface{}{
				"tenkan":         tenkan,
				"kijun":          kijun,
				"senkou_a":       senkouA,
				"senkou_b":       senkouB,
				"chikou":         chikouSpan,
				"current_price":  currentPrice,
				"cloud_color":    cloudColor,
				"price_position": pricePosition,
				"signal":         signal,
			}, nil
		},
		deps,
	).
		WithTimeout(15*time.Second).
		WithRetry(3, 500*time.Millisecond).
		Build()
}
// calculateMidpoint calculates (highest high + lowest low) / 2 for a period
func calculateMidpoint(data []struct{ high, low, close float64 }, start, end, period int) float64 {
	if start < 0 || end > len(data) || end-start < period {
		return 0
	}
	highest := data[start].high
	lowest := data[start].low
	for i := start; i < end && i < start+period; i++ {
		if data[i].high > highest {
			highest = data[i].high
		}
		if data[i].low < lowest {
			lowest = data[i].low
		}
	}
	return (highest + lowest) / 2
}
