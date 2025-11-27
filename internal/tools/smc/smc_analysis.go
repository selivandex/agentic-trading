package smc

import (
	"math"
	"time"

	"prometheus/internal/domain/market_data"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
)

// NewSMCAnalysisTool returns comprehensive Smart Money Concepts analysis in one call.
// This tool calculates ALL SMC patterns at once:
// - Market structure (trend, BOS, CHoCH)
// - Fair Value Gaps (FVG)
// - Order Blocks
// - Liquidity zones
// - Swing points
// - Imbalances
func NewSMCAnalysisTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"get_smc_analysis",
		"Get comprehensive Smart Money Concepts analysis in one call. Returns market structure, FVGs, order blocks, liquidity zones, and swing points. USE THIS INSTEAD OF CALLING INDIVIDUAL SMC TOOLS.",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Load candles ONCE
			candles, err := loadCandles(ctx, deps, args, 100)
			if err != nil {
				return nil, err
			}

			if len(candles) < 30 {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "need at least 30 candles, got %d", len(candles))
			}

			currentPrice := candles[0].Close
			lookback := parseLimit(args["lookback"], 5)

			result := map[string]interface{}{
				"symbol":        args["symbol"],
				"exchange":      args["exchange"],
				"timeframe":     args["timeframe"],
				"timestamp":     time.Now().UTC().Format(time.RFC3339),
				"current_price": currentPrice,

				// Market structure analysis
				"structure": analyzeStructure(candles, lookback),

				// Fair Value Gaps
				"fvg": analyzeFVGs(candles, currentPrice),

				// Order Blocks
				"order_blocks": analyzeOrderBlocks(candles, currentPrice),

				// Liquidity zones
				"liquidity": analyzeLiquidity(candles, currentPrice),

				// Swing points
				"swings": analyzeSwings(candles, lookback),
			}

			// Generate overall SMC signal
			result["signal"] = generateSMCSignal(result, currentPrice)

			return result, nil
		},
		deps,
	).
		WithTimeout(30*time.Second).
		WithRetry(2, 1*time.Second).
		Build()
}

// analyzeStructure returns market structure analysis
func analyzeStructure(candles []market_data.OHLCV, lookback int) map[string]interface{} {
	// Find swing points
	swingHighs := findSwingHighsWithIndex(candles, lookback)
	swingLows := findSwingLowsWithIndex(candles, lookback)

	// Determine trend
	trend, hh, hl, lh, ll := analyzeSwingProgression(swingHighs, swingLows)

	// Determine structure signal
	structureSignal := "neutral"
	if trend == "uptrend" {
		structureSignal = "bullish_structure"
	} else if trend == "downtrend" {
		structureSignal = "bearish_structure"
	}

	return map[string]interface{}{
		"trend":             trend,
		"higher_highs":      len(hh),
		"higher_lows":       len(hl),
		"lower_highs":       len(lh),
		"lower_lows":        len(ll),
		"structure_signal":  structureSignal,
		"recent_swing_high": getRecentSwingHigh(swingHighs),
		"recent_swing_low":  getRecentSwingLow(swingLows),
	}
}

// analyzeFVGs returns Fair Value Gap analysis
func analyzeFVGs(candles []market_data.OHLCV, currentPrice float64) map[string]interface{} {
	minGapPct := 0.1
	var bullishFVGs, bearishFVGs []map[string]interface{}

	for i := 0; i < len(candles)-2; i++ {
		candle0 := candles[i]
		candle2 := candles[i+2]

		// Bullish FVG
		if candle2.Low > candle0.High {
			gapStart := candle0.High
			gapEnd := candle2.Low
			gapPct := ((gapEnd - gapStart) / gapStart) * 100

			if gapPct >= minGapPct {
				filled := currentPrice <= gapStart
				bullishFVGs = append(bullishFVGs, map[string]interface{}{
					"start":  math.Round(gapStart*100) / 100,
					"end":    math.Round(gapEnd*100) / 100,
					"size":   math.Round(gapPct*100) / 100,
					"filled": filled,
					"index":  i,
				})
			}
		}

		// Bearish FVG
		if candle2.High < candle0.Low {
			gapStart := candle2.High
			gapEnd := candle0.Low
			gapPct := ((gapEnd - gapStart) / gapStart) * 100

			if gapPct >= minGapPct {
				filled := currentPrice >= gapEnd
				bearishFVGs = append(bearishFVGs, map[string]interface{}{
					"start":  math.Round(gapStart*100) / 100,
					"end":    math.Round(gapEnd*100) / 100,
					"size":   math.Round(gapPct*100) / 100,
					"filled": filled,
					"index":  i,
				})
			}
		}
	}

	// Find nearest unfilled FVG
	nearestBullish := findNearestUnfilledFVG(bullishFVGs, currentPrice, "bullish")
	nearestBearish := findNearestUnfilledFVG(bearishFVGs, currentPrice, "bearish")

	fvgSignal := "no_fvg"
	if nearestBullish != nil {
		fvgSignal = "bullish_fvg_support"
	}
	if nearestBearish != nil {
		if fvgSignal == "no_fvg" {
			fvgSignal = "bearish_fvg_resistance"
		} else {
			fvgSignal = "fvg_confluence"
		}
	}

	return map[string]interface{}{
		"bullish_count":   len(bullishFVGs),
		"bearish_count":   len(bearishFVGs),
		"nearest_bullish": nearestBullish,
		"nearest_bearish": nearestBearish,
		"signal":          fvgSignal,
	}
}

// analyzeOrderBlocks returns order block analysis
func analyzeOrderBlocks(candles []market_data.OHLCV, currentPrice float64) map[string]interface{} {
	var bullishOBs, bearishOBs []map[string]interface{}

	for i := 1; i < len(candles)-1; i++ {
		prev := candles[i+1]
		curr := candles[i]
		next := candles[i-1]

		// Bullish Order Block: down candle followed by strong up move
		if prev.Close < prev.Open && next.Close > prev.High {
			body := math.Abs(prev.Close - prev.Open)
			if body > 0 {
				bullishOBs = append(bullishOBs, map[string]interface{}{
					"high":   math.Round(prev.High*100) / 100,
					"low":    math.Round(prev.Low*100) / 100,
					"tested": currentPrice >= prev.Low && currentPrice <= prev.High,
					"index":  i,
				})
			}
		}

		// Bearish Order Block: up candle followed by strong down move
		if prev.Close > prev.Open && next.Close < prev.Low {
			body := math.Abs(prev.Close - prev.Open)
			if body > 0 {
				bearishOBs = append(bearishOBs, map[string]interface{}{
					"high":   math.Round(prev.High*100) / 100,
					"low":    math.Round(prev.Low*100) / 100,
					"tested": currentPrice >= prev.Low && currentPrice <= prev.High,
					"index":  i,
				})
			}
		}

		_ = curr // Suppress unused warning
	}

	// Find nearest order blocks
	var nearestBullish, nearestBearish map[string]interface{}
	for _, ob := range bullishOBs {
		if ob["low"].(float64) < currentPrice {
			if nearestBullish == nil || ob["high"].(float64) > nearestBullish["high"].(float64) {
				nearestBullish = ob
			}
		}
	}
	for _, ob := range bearishOBs {
		if ob["high"].(float64) > currentPrice {
			if nearestBearish == nil || ob["low"].(float64) < nearestBearish["low"].(float64) {
				nearestBearish = ob
			}
		}
	}

	obSignal := "no_order_blocks"
	if nearestBullish != nil && nearestBearish != nil {
		obSignal = "order_block_range"
	} else if nearestBullish != nil {
		obSignal = "bullish_ob_support"
	} else if nearestBearish != nil {
		obSignal = "bearish_ob_resistance"
	}

	return map[string]interface{}{
		"bullish_count":   len(bullishOBs),
		"bearish_count":   len(bearishOBs),
		"nearest_bullish": nearestBullish,
		"nearest_bearish": nearestBearish,
		"signal":          obSignal,
	}
}

// analyzeLiquidity returns liquidity zone analysis
func analyzeLiquidity(candles []market_data.OHLCV, currentPrice float64) map[string]interface{} {
	// Find equal highs (liquidity above)
	equalHighs := findEqualLevels(candles, true, 0.001)

	// Find equal lows (liquidity below)
	equalLows := findEqualLevels(candles, false, 0.001)

	// Recent high as liquidity target
	recentHigh := candles[0].High
	recentLow := candles[0].Low
	for i := 0; i < minInt(20, len(candles)); i++ {
		if candles[i].High > recentHigh {
			recentHigh = candles[i].High
		}
		if candles[i].Low < recentLow {
			recentLow = candles[i].Low
		}
	}

	// Distance to liquidity
	distanceToHighLiq := ((recentHigh - currentPrice) / currentPrice) * 100
	distanceToLowLiq := ((currentPrice - recentLow) / currentPrice) * 100

	liquiditySignal := "neutral"
	if distanceToHighLiq < 1.0 {
		liquiditySignal = "near_liquidity_high"
	} else if distanceToLowLiq < 1.0 {
		liquiditySignal = "near_liquidity_low"
	}

	return map[string]interface{}{
		"equal_highs_count":      len(equalHighs),
		"equal_lows_count":       len(equalLows),
		"nearest_high_liquidity": math.Round(recentHigh*100) / 100,
		"nearest_low_liquidity":  math.Round(recentLow*100) / 100,
		"distance_to_high_pct":   math.Round(distanceToHighLiq*100) / 100,
		"distance_to_low_pct":    math.Round(distanceToLowLiq*100) / 100,
		"signal":                 liquiditySignal,
	}
}

// analyzeSwings returns swing point analysis
func analyzeSwings(candles []market_data.OHLCV, lookback int) map[string]interface{} {
	swingHighs := findSwingHighsWithIndex(candles, lookback)
	swingLows := findSwingLowsWithIndex(candles, lookback)

	var recentHighs, recentLows []map[string]interface{}

	// Get last 5 swing highs
	for i := 0; i < minInt(5, len(swingHighs)); i++ {
		recentHighs = append(recentHighs, map[string]interface{}{
			"price": math.Round(swingHighs[i].price*100) / 100,
			"index": swingHighs[i].index,
		})
	}

	// Get last 5 swing lows
	for i := 0; i < minInt(5, len(swingLows)); i++ {
		recentLows = append(recentLows, map[string]interface{}{
			"price": math.Round(swingLows[i].price*100) / 100,
			"index": swingLows[i].index,
		})
	}

	return map[string]interface{}{
		"swing_highs":      recentHighs,
		"swing_lows":       recentLows,
		"total_high_count": len(swingHighs),
		"total_low_count":  len(swingLows),
	}
}

// generateSMCSignal produces overall SMC signal
func generateSMCSignal(result map[string]interface{}, currentPrice float64) map[string]interface{} {
	bullishSignals := 0
	bearishSignals := 0

	// Count structure signal
	if structure, ok := result["structure"].(map[string]interface{}); ok {
		if signal, ok := structure["structure_signal"].(string); ok {
			if signal == "bullish_structure" {
				bullishSignals++
			} else if signal == "bearish_structure" {
				bearishSignals++
			}
		}
	}

	// Count FVG signal
	if fvg, ok := result["fvg"].(map[string]interface{}); ok {
		if signal, ok := fvg["signal"].(string); ok {
			if signal == "bullish_fvg_support" {
				bullishSignals++
			} else if signal == "bearish_fvg_resistance" {
				bearishSignals++
			}
		}
	}

	// Count order block signal
	if ob, ok := result["order_blocks"].(map[string]interface{}); ok {
		if signal, ok := ob["signal"].(string); ok {
			if signal == "bullish_ob_support" {
				bullishSignals++
			} else if signal == "bearish_ob_resistance" {
				bearishSignals++
			}
		}
	}

	direction := "neutral"
	confidence := 0.5

	total := bullishSignals + bearishSignals
	if total > 0 {
		if bullishSignals > bearishSignals {
			direction = "bullish"
			confidence = float64(bullishSignals) / float64(total+1)
		} else if bearishSignals > bullishSignals {
			direction = "bearish"
			confidence = float64(bearishSignals) / float64(total+1)
		}
	}

	return map[string]interface{}{
		"direction":     direction,
		"confidence":    math.Round(confidence*100) / 100,
		"bullish_count": bullishSignals,
		"bearish_count": bearishSignals,
		"bias":          getBias(direction, bullishSignals, bearishSignals),
	}
}

// Helper functions

func getRecentSwingHigh(swings []swingPoint) float64 {
	if len(swings) == 0 {
		return 0
	}
	return math.Round(swings[0].price*100) / 100
}

func getRecentSwingLow(swings []swingPoint) float64 {
	if len(swings) == 0 {
		return 0
	}
	return math.Round(swings[0].price*100) / 100
}

func findNearestUnfilledFVG(fvgs []map[string]interface{}, currentPrice float64, fvgType string) map[string]interface{} {
	var nearest map[string]interface{}
	minDistance := math.MaxFloat64

	for _, fvg := range fvgs {
		if filled, ok := fvg["filled"].(bool); ok && filled {
			continue
		}

		var distance float64
		if fvgType == "bullish" {
			distance = currentPrice - fvg["start"].(float64)
		} else {
			distance = fvg["end"].(float64) - currentPrice
		}

		if distance > 0 && distance < minDistance {
			minDistance = distance
			nearest = fvg
		}
	}

	return nearest
}

func findEqualLevels(candles []market_data.OHLCV, isHigh bool, tolerance float64) []float64 {
	var levels []float64
	levelCount := make(map[float64]int)

	for _, c := range candles {
		var level float64
		if isHigh {
			level = math.Round(c.High/tolerance) * tolerance
		} else {
			level = math.Round(c.Low/tolerance) * tolerance
		}
		levelCount[level]++
	}

	for level, count := range levelCount {
		if count >= 2 {
			levels = append(levels, level)
		}
	}

	return levels
}

func getBias(direction string, bullish, bearish int) string {
	diff := bullish - bearish
	if diff >= 2 {
		return "strong_bullish"
	} else if diff == 1 {
		return "mild_bullish"
	} else if diff <= -2 {
		return "strong_bearish"
	} else if diff == -1 {
		return "mild_bearish"
	}
	return "neutral"
}

// minInt returns smaller of two ints
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
