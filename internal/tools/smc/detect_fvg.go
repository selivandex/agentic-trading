package smc

import (
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// FairValueGap represents a detected Fair Value Gap
type FairValueGap struct {
	Type       string  `json:"type"`        // bullish, bearish
	StartPrice float64 `json:"start_price"` // Bottom of gap
	EndPrice   float64 `json:"end_price"`   // Top of gap
	GapSize    float64 `json:"gap_size"`    // Size in price
	GapSizePct float64 `json:"gap_size_pct"` // Size as percentage
	FormTime   int64   `json:"form_time"`   // Unix timestamp
	Index      int     `json:"index"`       // Candle index (0 = most recent)
	Filled     bool    `json:"filled"`      // Has price returned to fill gap?
}

// NewDetectFVGTool detects Fair Value Gaps (FVG) - price inefficiencies
// Bullish FVG: candle[i-2].low > candle[i].high (gap between them, candle i-1 created it)
// Bearish FVG: candle[i-2].high < candle[i].low
func NewDetectFVGTool(deps shared.Deps) tool.Tool {
	t, _ := functiontool.New(
		functiontool.Config{
			Name:        "detect_fvg",
			Description: "Detect Fair Value Gaps",
		},
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
		// Load candles
		candles, err := loadCandles(ctx, deps, args, 100)
		if err != nil {
			return nil, err
		}

		if len(candles) < 3 {
			return nil, errors.Wrapf(errors.ErrInvalidInput, "need at least 3 candles for FVG detection")
		}

		minGapSizePct := parseFloat(args["min_gap_pct"], 0.1) // Minimum 0.1% gap

		fvgs := make([]FairValueGap, 0)
		currentPrice := candles[0].Close

		// Scan for FVGs (need 3-candle pattern)
		// candles are in DESC order (newest first)
		for i := 0; i < len(candles)-2; i++ {
			candle0 := candles[i]     // Most recent of the 3
			candle1 := candles[i+1]   // Middle candle
			candle2 := candles[i+2]   // Oldest of the 3

			// Bullish FVG: gap between candle2.low and candle0.high
			// Means candle1 moved price up so fast that it left a gap
			if candle2.Low > candle0.High {
				gapStart := candle0.High
				gapEnd := candle2.Low
				gapSize := gapEnd - gapStart
				gapSizePct := (gapSize / gapStart) * 100

				if gapSizePct >= minGapSizePct {
					// Check if gap has been filled
					filled := currentPrice <= gapStart

					fvgs = append(fvgs, FairValueGap{
						Type:       "bullish",
						StartPrice: gapStart,
						EndPrice:   gapEnd,
						GapSize:    gapSize,
						GapSizePct: gapSizePct,
						FormTime:   candle1.OpenTime.Unix(),
						Index:      i,
						Filled:     filled,
					})
				}
			}

			// Bearish FVG: gap between candle2.high and candle0.low
			// Means candle1 moved price down so fast that it left a gap
			if candle2.High < candle0.Low {
				gapStart := candle2.High
				gapEnd := candle0.Low
				gapSize := gapEnd - gapStart
				gapSizePct := (gapSize / gapStart) * 100

				if gapSizePct >= minGapSizePct {
					// Check if gap has been filled
					filled := currentPrice >= gapEnd

					fvgs = append(fvgs, FairValueGap{
						Type:       "bearish",
						StartPrice: gapStart,
						EndPrice:   gapEnd,
						GapSize:    gapSize,
						GapSizePct: gapSizePct,
						FormTime:   candle1.OpenTime.Unix(),
						Index:      i,
						Filled:     filled,
					})
				}
			}
		}

		// Find nearest unfilled FVG
		var nearestFVG *FairValueGap
		minDistance := 999999.0

		for i := range fvgs {
			if !fvgs[i].Filled {
				// Distance from current price to gap
				distance := 0.0
				if fvgs[i].Type == "bullish" {
					distance = fvgs[i].StartPrice - currentPrice
				} else {
					distance = currentPrice - fvgs[i].EndPrice
				}

				if distance > 0 && distance < minDistance {
					minDistance = distance
					nearestFVG = &fvgs[i]
				}
			}
		}

		// Generate signal
		signal := "no_fvg"
		if nearestFVG != nil {
			if nearestFVG.Type == "bullish" {
				signal = "bullish_fvg_below" // Support zone
			} else {
				signal = "bearish_fvg_above" // Resistance zone
			}
		}

		return map[string]interface{}{
			"fvgs_found":     len(fvgs),
			"unfilled_fvgs":  countUnfilled(fvgs),
			"nearest_fvg":    nearestFVG,
			"all_fvgs":       fvgs,
			"signal":         signal,
			"current_price":  currentPrice,
		}, nil
	})
	return t
}

// Helper functions
func countUnfilled(fvgs []FairValueGap) int {
	count := 0
	for _, fvg := range fvgs {
		if !fvg.Filled {
			count++
		}
	}
	return count
}

func parseFloat(val interface{}, defaultVal float64) float64 {
	if val == nil {
		return defaultVal
	}

	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return defaultVal
	}
}

