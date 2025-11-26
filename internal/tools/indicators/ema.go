package indicators

import (
	"context"

	"prometheus/internal/tools/shared"

	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewEMATool computes the exponential moving average.
func NewEMATool(deps shared.Deps) tool.Tool {
	return functiontool.New("ema", "Exponential Moving Average", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		candles, err := loadCandles(ctx, deps, args, 120)
		if err != nil {
			return nil, err
		}
		period := parseLimit(args["period"], 20)
		closes := extractCloses(candles)
		if len(closes) < period {
			return nil, errors.Wrapf(errors.ErrInternal, "ema: not enough data for period %d", period)
		}

		multiplier := 2.0 / (float64(period) + 1)
		ema := closes[0]
		for i := 1; i < len(closes); i++ {
			ema = (closes[i]-ema)*multiplier + ema
		}

		return map[string]interface{}{"value": ema}, nil
	})
}
