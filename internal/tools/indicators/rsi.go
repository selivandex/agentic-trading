package indicators

import (
	"context"
	"math"

	"prometheus/internal/tools/shared"

	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewRSITool computes Relative Strength Index using closing prices.
func NewRSITool(deps shared.Deps) tool.Tool {
	return functiontool.New("rsi", "Relative Strength Index", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		candles, err := loadCandles(ctx, deps, args, 100)
		if err != nil {
			return nil, err
		}
		period := parseLimit(args["period"], 14)
		closes := extractCloses(candles)
		if len(closes) < period+1 {
			return nil, errors.Wrapf(errors.ErrInternal, "rsi: not enough data for period %d", period)
		}

		gains := 0.0
		losses := 0.0
		for i := 1; i <= period; i++ {
			delta := closes[i] - closes[i-1]
			if delta > 0 {
				gains += delta
			} else {
				losses += -delta
			}
		}
		avgGain := gains / float64(period)
		avgLoss := losses / float64(period)
		rs := math.Inf(1)
		if avgLoss != 0 {
			rs = avgGain / avgLoss
		}
		rsi := 100.0 - (100.0 / (1 + rs))

		return map[string]interface{}{"value": rsi}, nil
	})
}
