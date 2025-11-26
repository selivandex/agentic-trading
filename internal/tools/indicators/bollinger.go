package indicators

import (
	"github.com/markcheno/go-talib"

	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewBollingerTool computes Bollinger Bands using ta-lib
func NewBollingerTool(deps shared.Deps) tool.Tool {
	t, _ := functiontool.New(
		functiontool.Config{
			Name:        "bollinger",
			Description: "Bollinger Bands",
		},
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Load candles
			candles, err := loadCandles(ctx, deps, args, 200)
			if err != nil {
				return nil, err
			}

			period := parseLimit(args["period"], 20)
			stdDevUp := parseFloat(args["std_dev_up"], 2.0)
			stdDevDown := parseFloat(args["std_dev_down"], 2.0)

			if err := ValidateMinLength(candles, period, "Bollinger Bands"); err != nil {
				return nil, err
			}

			// Prepare data for ta-lib
			closes, err := PrepareCloses(candles)
			if err != nil {
				return nil, err
			}

			// Calculate Bollinger Bands using ta-lib
			upperBand, middleBand, lowerBand := talib.BBands(closes, period, stdDevUp, stdDevDown, talib.SMA)

			// Get latest values
			upper, err := GetLastValue(upperBand)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get upper band")
			}

			middle, err := GetLastValue(middleBand)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get middle band")
			}

			lower, err := GetLastValue(lowerBand)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get lower band")
			}

			// Calculate bandwidth (useful for volatility assessment)
			bandwidth := ((upper - lower) / middle) * 100

			// Get current price for position analysis
			currentPrice := candles[0].Close

			// Determine position relative to bands
			position := "middle"
			if currentPrice >= upper {
				position = "above_upper"
			} else if currentPrice <= lower {
				position = "below_lower"
			} else if currentPrice > middle {
				position = "upper_half"
			} else if currentPrice < middle {
				position = "lower_half"
			}

			return map[string]interface{}{
				"upper":         upper,
				"middle":        middle,
				"lower":         lower,
				"bandwidth":     bandwidth,
				"current_price": currentPrice,
				"position":      position,
				"period":        period,
			}, nil
		})
	return t
}

// parseFloat parses float64 from interface{} with default
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
