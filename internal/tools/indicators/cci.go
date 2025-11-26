package indicators
import (
	"time"
	"github.com/markcheno/go-talib"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
	"google.golang.org/adk/tool"
)
// NewCCITool computes Commodity Channel Index using ta-lib
func NewCCITool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"cci",
		"Commodity Channel Index",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Load candles
			candles, err := loadCandles(ctx, deps, args, 200)
			if err != nil {
				return nil, err
			}
			period := parseLimit(args["period"], 20)
			if err := ValidateMinLength(candles, period, "CCI"); err != nil {
				return nil, err
			}
			// Prepare data for ta-lib
			data, err := PrepareData(candles)
			if err != nil {
				return nil, err
			}
			// Calculate CCI using ta-lib
			cci := talib.Cci(data.High, data.Low, data.Close, period)
			// Get latest value
			value, err := GetLastValue(cci)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get CCI value")
			}
			// CCI interpretation:
			// > +100 = overbought
			// < -100 = oversold
			// Crossing zero line = trend change
			signal := "neutral"
			if value > 100 {
				signal = "overbought"
			} else if value < -100 {
				signal = "oversold"
			} else if value > 0 {
				signal = "bullish"
			} else if value < 0 {
				signal = "bearish"
			}
			return map[string]interface{}{
				"value":  value,
				"signal": signal,
				"period": period,
			}, nil
		},
		deps,
	).
		WithTimeout(15*time.Second).
		WithRetry(3, 500*time.Millisecond).
		Build()
}
