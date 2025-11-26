package indicators

import (
	"context"
	"fmt"

	"prometheus/internal/tools/shared"

	"google.golang.org/adk/tool/functiontool"
)

// NewMACDTool computes MACD (12,26,9 by default).
func NewMACDTool(deps shared.Deps) *functiontool.Tool {
	return functiontool.New("macd", "Moving Average Convergence Divergence", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		candles, err := loadCandles(ctx, deps, args, 200)
		if err != nil {
			return nil, err
		}
		fast := parseLimit(args["fast"], 12)
		slow := parseLimit(args["slow"], 26)
		signal := parseLimit(args["signal"], 9)
		closes := extractCloses(candles)
		if len(closes) < slow+signal {
			return nil, fmt.Errorf("macd: not enough data")
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
	})
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
