package indicators
import (
	"time"
	"github.com/markcheno/go-talib"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
	"google.golang.org/adk/tool"
)
// NewROCTool computes Rate of Change using ta-lib
func NewROCTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"roc",
		"Rate of Change",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Load candles
			candles, err := loadCandles(ctx, deps, args, 200)
			if err != nil {
				return nil, err
			}
			period := parseLimit(args["period"], 12)
			if err := ValidateMinLength(candles, period+1, "ROC"); err != nil {
				return nil, err
			}
			// Prepare data for ta-lib
			closes, err := PrepareCloses(candles)
			if err != nil {
				return nil, err
			}
			// Calculate ROC using ta-lib
			// ROC = ((close - close[n periods ago]) / close[n periods ago]) * 100
			roc := talib.Roc(closes, period)
			// Get latest value
			value, err := GetLastValue(roc)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get ROC value")
			}
			// ROC interpretation:
			// Positive = bullish momentum
			// Negative = bearish momentum
			// Magnitude indicates strength
			signal := "neutral"
			momentum := "weak"
			if value > 5 {
				signal = "strong_bullish"
				momentum = "strong"
			} else if value > 1 {
				signal = "bullish"
				momentum = "moderate"
			} else if value < -5 {
				signal = "strong_bearish"
				momentum = "strong"
			} else if value < -1 {
				signal = "bearish"
				momentum = "moderate"
			}
			return map[string]interface{}{
				"value":    value,
				"signal":   signal,
				"momentum": momentum,
				"period":   period,
			}, nil
		},
		deps,
	).
		WithTimeout(15*time.Second).
		WithRetry(3, 500*time.Millisecond).
		Build()
}
