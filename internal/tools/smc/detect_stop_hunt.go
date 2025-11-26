package smc

import (
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
	"time"

	"google.golang.org/adk/tool"
)

// StopHunt represents a detected stop hunt event
type StopHunt struct {
	Type        string  `json:"type"`       // bullish_hunt, bearish_hunt
	HuntPrice   float64 `json:"hunt_price"` // Price where stops were hunted
	WickSize    float64 `json:"wick_size"`  // Size of the hunting wick
	WickSizePct float64 `json:"wick_size_pct"`
	Reversal    float64 `json:"reversal"` // How much price reversed after hunt
	FormTime    int64   `json:"form_time"`
	Index       int     `json:"index"`
}

// NewDetectStopHuntTool detects stop hunt patterns
// Stop Hunt = Price briefly exceeds swing high/low then reverses sharply
// Indicates smart money triggering retail stops before reversing
func NewDetectStopHuntTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"detect_stop_hunt",
		"Detect Stop Hunts",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			candles, err := loadCandles(ctx, deps, args, 100)
			if err != nil {
				return nil, err
			}

			if len(candles) < 10 {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "need at least 10 candles")
			}

			lookback := parseLimit(args["lookback"], 10)
			minWickPct := parseFloat(args["min_wick_pct"], 1.0)         // Min 1% wick
			minReversalPct := parseFloat(args["min_reversal_pct"], 0.5) // Min 0.5% reversal

			stopHunts := make([]StopHunt, 0)

			for i := 0; i < len(candles)-lookback; i++ {
				candle := candles[i]
				candleRange := candle.High - candle.Low

				if candleRange == 0 {
					continue
				}

				// Find recent swing high/low
				swingHigh := candle.High
				swingLow := candle.Low

				for j := i + 1; j < i+lookback && j < len(candles); j++ {
					if candles[j].High > swingHigh {
						swingHigh = candles[j].High
					}
					if candles[j].Low < swingLow {
						swingLow = candles[j].Low
					}
				}

				// Check for bullish stop hunt (lower wick exceeds swing low, then reverses)
				if candle.Low < swingLow {
					upperWickPct := ((candle.High - max(candle.Open, candle.Close)) / candleRange) * 100
					lowerWick := min(candle.Open, candle.Close) - candle.Low
					lowerWickPct := (lowerWick / candleRange) * 100

					// Long wick below + close above
					if lowerWickPct >= minWickPct && candle.Close > candle.Open {
						reversalPct := ((candle.Close - candle.Low) / candle.Low) * 100

						if reversalPct >= minReversalPct {
							stopHunts = append(stopHunts, StopHunt{
								Type:        "bullish_hunt",
								HuntPrice:   candle.Low,
								WickSize:    lowerWick,
								WickSizePct: lowerWickPct,
								Reversal:    reversalPct,
								FormTime:    candle.OpenTime.Unix(),
								Index:       i,
							})
						}
					}

					_ = upperWickPct // May use in future for additional validation
				}

				// Check for bearish stop hunt (upper wick exceeds swing high, then reverses)
				if candle.High > swingHigh {
					upperWick := candle.High - max(candle.Open, candle.Close)
					upperWickPct := (upperWick / candleRange) * 100

					// Long wick above + close below
					if upperWickPct >= minWickPct && candle.Close < candle.Open {
						reversalPct := ((candle.High - candle.Close) / candle.High) * 100

						if reversalPct >= minReversalPct {
							stopHunts = append(stopHunts, StopHunt{
								Type:        "bearish_hunt",
								HuntPrice:   candle.High,
								WickSize:    upperWick,
								WickSizePct: upperWickPct,
								Reversal:    reversalPct,
								FormTime:    candle.OpenTime.Unix(),
								Index:       i,
							})
						}
					}
				}
			}

			// Find most recent stop hunt
			var recentHunt *StopHunt
			if len(stopHunts) > 0 {
				recentHunt = &stopHunts[0]
			}

			signal := "no_stop_hunt"
			if recentHunt != nil && recentHunt.Index < 5 {
				// Recent stop hunt
				if recentHunt.Type == "bullish_hunt" {
					signal = "bullish_after_hunt" // Longs stopped, now reversal up
				} else {
					signal = "bearish_after_hunt" // Shorts stopped, now reversal down
				}
			}

			return map[string]interface{}{
				"stop_hunts":    stopHunts,
				"recent_hunt":   recentHunt,
				"signal":        signal,
				"current_price": candles[0].Close,
			}, nil
		},
		deps,
	).
		WithTimeout(10*time.Second).
		WithRetry(3, 500*time.Millisecond).
		WithStats().
		Build()
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
