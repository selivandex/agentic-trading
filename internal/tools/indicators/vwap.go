package indicators

import (
	"time"

	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
)

// NewVWAPTool computes Volume Weighted Average Price
// VWAP = cumsum(price * volume) / cumsum(volume)
// Typically calculated from start of trading day
func NewVWAPTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"vwap",
		"Volume Weighted Average Price",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Load candles
			candles, err := loadCandles(ctx, deps, args, 100)
			if err != nil {
				return nil, err
			}

			if len(candles) == 0 {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "no candles for VWAP")
			}

			// Calculate VWAP
			// Typically uses (high + low + close) / 3 as typical price
			cumulativePV := 0.0 // Price * Volume
			cumulativeV := 0.0  // Volume

			// Iterate from oldest to newest (reverse our DESC order)
			for i := len(candles) - 1; i >= 0; i-- {
				candle := candles[i]

				// Typical price (high + low + close) / 3
				typicalPrice := (candle.High + candle.Low + candle.Close) / 3

				// Accumulate price * volume
				cumulativePV += typicalPrice * candle.Volume
				cumulativeV += candle.Volume
			}

			if cumulativeV == 0 {
				return nil, errors.Wrapf(errors.ErrInternal, "zero volume for VWAP calculation")
			}

			vwap := cumulativePV / cumulativeV
			currentPrice := candles[0].Close

			// Calculate deviation from VWAP
			deviationPct := ((currentPrice - vwap) / vwap) * 100

			// Determine position relative to VWAP
			position := "at_vwap"
			if deviationPct > 2 {
				position = "above_vwap"
			} else if deviationPct < -2 {
				position = "below_vwap"
			}

			// VWAP is used as dynamic support/resistance
			// Price below VWAP = bearish, above = bullish
			signal := "neutral"
			if currentPrice > vwap {
				signal = "bullish"
			} else if currentPrice < vwap {
				signal = "bearish"
			}

			return map[string]interface{}{
				"vwap":          vwap,
				"current_price": currentPrice,
				"deviation_pct": deviationPct,
				"position":      position,
				"signal":        signal,
			}, nil
		},
		deps,
	).
		WithTimeout(15*time.Second).
		WithRetry(3, 500*time.Millisecond).
		WithStats().
		Build()
}
