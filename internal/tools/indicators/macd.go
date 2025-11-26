package indicators

import (
	"time"

	"prometheus/internal/tools"
	"prometheus/internal/tools/shared"

	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
)

// NewMACDTool computes MACD (12,26,9 by default).
func NewMACDTool(deps shared.Deps) tool.Tool {
	return tools.NewFactory(
		"macd",
		"Moving Average Convergence Divergence",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			candles, err := loadCandles(ctx, deps, args, 200)
			if err != nil {
				return nil, err
			}
			fast := parseLimit(args["fast"], 12)
			slow := parseLimit(args["slow"], 26)
			signal := parseLimit(args["signal"], 9)
			closes := extractCloses(candles)
			if len(closes) < slow+signal {
				return nil, errors.Wrapf(errors.ErrInternal, "macd: not enough data")
			}

			emaFast := computeEMA(closes, fast)
			emaSlow := computeEMA(closes, slow)
			macdLine := emaFast - emaSlow
			signalLine := computeEMA([]float64{macdLine}, signal)
			histogram := macdLine - signalLine

			return map[string]interface{}{
				"macd":      macdLine,
				"signal":    signalLine,
				"histogram": histogram,
			}, nil
		},
		deps,
	).
		WithTimeout(15 * time.Second).
		WithRetry(3, 500*time.Millisecond).
		WithStats().
		Build()
}

func computeEMA(series []float64, period int) float64 {
	if len(series) == 0 || period <= 1 {
		return 0
	}
	multiplier := 2.0 / (float64(period) + 1)
	ema := series[0]
	for i := 1; i < len(series); i++ {
		ema = (series[i]-ema)*multiplier + ema
	}
	return ema
}
