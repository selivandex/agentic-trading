package smc

import (
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// SwingPoint represents a swing high or swing low
type SwingPoint struct {
	Type      string  `json:"type"`      // swing_high, swing_low
	Price     float64 `json:"price"`     // Price level
	Index     int     `json:"index"`     // Candle index
	Timestamp int64   `json:"timestamp"` // Unix timestamp
	Strength  int     `json:"strength"`  // Number of candles on each side that confirm it
}

// NewGetSwingPointsTool identifies swing highs and swing lows
// Swing High: local maximum (higher than N candles before and after)
// Swing Low: local minimum (lower than N candles before and after)
func NewGetSwingPointsTool(deps shared.Deps) tool.Tool {
	t, _ := functiontool.New(
		functiontool.Config{
			Name:        "get_swing_points",
			Description: "Get Swing Points (Highs/Lows)",
		},
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Load candles
			candles, err := loadCandles(ctx, deps, args, 100)
			if err != nil {
				return nil, err
			}

			lookback := parseLimit(args["lookback"], 5) // Number of candles on each side

			if len(candles) < lookback*2+1 {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "need at least %d candles for swing detection", lookback*2+1)
			}

			swingHighs := make([]SwingPoint, 0)
			swingLows := make([]SwingPoint, 0)

			// Scan for swing points (skip first and last 'lookback' candles)
			for i := lookback; i < len(candles)-lookback; i++ {
				candle := candles[i]

				// Check for swing high
				isSwingHigh := true
				for j := 1; j <= lookback; j++ {
					// Check left side
					if candles[i+j].High >= candle.High {
						isSwingHigh = false
						break
					}
					// Check right side
					if candles[i-j].High >= candle.High {
						isSwingHigh = false
						break
					}
				}

				if isSwingHigh {
					swingHighs = append(swingHighs, SwingPoint{
						Type:      "swing_high",
						Price:     candle.High,
						Index:     i,
						Timestamp: candle.OpenTime.Unix(),
						Strength:  lookback,
					})
				}

				// Check for swing low
				isSwingLow := true
				for j := 1; j <= lookback; j++ {
					// Check left side
					if candles[i+j].Low <= candle.Low {
						isSwingLow = false
						break
					}
					// Check right side
					if candles[i-j].Low <= candle.Low {
						isSwingLow = false
						break
					}
				}

				if isSwingLow {
					swingLows = append(swingLows, SwingPoint{
						Type:      "swing_low",
						Price:     candle.Low,
						Index:     i,
						Timestamp: candle.OpenTime.Unix(),
						Strength:  lookback,
					})
				}
			}

			currentPrice := candles[0].Close

			// Find nearest swing high and low
			var nearestHigh, nearestLow *SwingPoint

			for i := range swingHighs {
				if swingHighs[i].Price > currentPrice {
					if nearestHigh == nil || swingHighs[i].Price < nearestHigh.Price {
						nearestHigh = &swingHighs[i]
					}
				}
			}

			for i := range swingLows {
				if swingLows[i].Price < currentPrice {
					if nearestLow == nil || swingLows[i].Price > nearestLow.Price {
						nearestLow = &swingLows[i]
					}
				}
			}

			// Calculate range
			range_ := 0.0
			if nearestHigh != nil && nearestLow != nil {
				range_ = nearestHigh.Price - nearestLow.Price
			}

			return map[string]interface{}{
				"swing_highs":   swingHighs,
				"swing_lows":    swingLows,
				"nearest_high":  nearestHigh,
				"nearest_low":   nearestLow,
				"range":         range_,
				"current_price": currentPrice,
				"lookback":      lookback,
			}, nil
		})
	return t
}
