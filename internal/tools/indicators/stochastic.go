package indicators

import (
	"time"

	"github.com/markcheno/go-talib"

	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
)

// NewStochasticTool computes Stochastic Oscillator using ta-lib
func NewStochasticTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"stochastic",
		"Stochastic Oscillator",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Load candles
			candles, err := loadCandles(ctx, deps, args, 200)
			if err != nil {
				return nil, err
			}

			// Stochastic parameters
			kPeriod := parseLimit(args["k_period"], 14) // %K period
			dPeriod := parseLimit(args["d_period"], 3)  // %D period (signal)
			slowK := parseLimit(args["slow_k"], 3)      // Slow %K smoothing

			minLength := kPeriod + slowK + dPeriod
			if err := ValidateMinLength(candles, minLength, "Stochastic"); err != nil {
				return nil, err
			}

			// Prepare data for ta-lib
			data, err := PrepareData(candles)
			if err != nil {
				return nil, err
			}

			// Calculate Stochastic using ta-lib
			// Returns slowK and slowD
			slowKLine, slowDLine := talib.Stoch(
				data.High,
				data.Low,
				data.Close,
				kPeriod,
				slowK,
				talib.SMA,
				dPeriod,
				talib.SMA,
			)

			// Get latest values
			k, err := GetLastValue(slowKLine)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get %K value")
			}

			d, err := GetLastValue(slowDLine)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get %D value")
			}

			// Determine signal
			signal := "neutral"
			if k < 20 && d < 20 {
				signal = "oversold" // Potential buy signal
			} else if k > 80 && d > 80 {
				signal = "overbought" // Potential sell signal
			} else if k > d && k < 50 {
				signal = "bullish_cross" // %K crossed above %D in lower zone
			} else if k < d && k > 50 {
				signal = "bearish_cross" // %K crossed below %D in upper zone
			}

			return map[string]interface{}{
				"k":        k,
				"d":        d,
				"signal":   signal,
				"k_period": kPeriod,
				"d_period": dPeriod,
			}, nil
		},
		deps,
	).
		WithTimeout(15*time.Second).
		WithRetry(3, 500*time.Millisecond).
		WithStats().
		Build()
}
