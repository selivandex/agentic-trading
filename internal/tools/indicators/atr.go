package indicators

import (
	"context"
	"fmt"
	"math"

	"prometheus/internal/tools/shared"

	"google.golang.org/adk/tool/functiontool"
)

// NewATRTool computes Average True Range.
func NewATRTool(deps shared.Deps) *functiontool.Tool {
	return functiontool.New("atr", "Average True Range", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		candles, err := loadCandles(ctx, deps, args, 100)
		if err != nil {
			return nil, err
		}
		period := parseLimit(args["period"], 14)
		high, low, close := extractHighLow(candles)
		if len(close) < period+1 {
			return nil, fmt.Errorf("atr: not enough data")
		}

		trs := make([]float64, 0, len(close)-1)
		for i := 1; i < len(close); i++ {
			highLow := high[i] - low[i]
			highClose := math.Abs(high[i] - close[i-1])
			lowClose := math.Abs(low[i] - close[i-1])
			trs = append(trs, math.Max(highLow, math.Max(highClose, lowClose)))
		}

		if len(trs) < period {
			return nil, fmt.Errorf("atr: not enough true range values")
		}

		atr := 0.0
		for i := 0; i < period; i++ {
			atr += trs[i]
		}
		atr = atr / float64(period)

		return map[string]interface{}{"value": atr}, nil
	})
}
