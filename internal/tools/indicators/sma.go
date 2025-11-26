package indicators
import (
	"time"
	"github.com/markcheno/go-talib"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
	"google.golang.org/adk/tool"
)
// NewSMATool computes Simple Moving Average using ta-lib
func NewSMATool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"sma",
		"Simple Moving Average",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Load candles
			candles, err := loadCandles(ctx, deps, args, 200)
			if err != nil {
				return nil, err
			}
			period := parseLimit(args["period"], 20)
			if err := ValidateMinLength(candles, period, "SMA"); err != nil {
				return nil, err
			}
			// Prepare data for ta-lib
			closes, err := PrepareCloses(candles)
			if err != nil {
				return nil, err
			}
			// Calculate SMA using ta-lib
			sma := talib.Sma(closes, period)
			// Get latest value
			value, err := GetLastValue(sma)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get SMA value")
			}
			return map[string]interface{}{
				"value":  value,
				"period": period,
			}, nil
		},
		deps,
	).
		WithTimeout(15*time.Second).
		WithRetry(3, 500*time.Millisecond).
		Build()
}
