package indicators

import (
	"context"

	"github.com/markcheno/go-talib"

	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewKeltnerTool computes Keltner Channels
// Keltner Channels = EMA +/- (ATR * multiplier)
// Similar to Bollinger Bands but uses ATR instead of standard deviation
func NewKeltnerTool(deps shared.Deps) tool.Tool {
	return functiontool.New("keltner", "Keltner Channels", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		// Load candles
		candles, err := loadCandles(ctx, deps, args, 200)
		if err != nil {
			return nil, err
		}

		emaPeriod := parseLimit(args["ema_period"], 20)
		atrPeriod := parseLimit(args["atr_period"], 14)
		multiplier := parseFloat(args["multiplier"], 2.0)

		minLength := max(emaPeriod, atrPeriod)
		if err := ValidateMinLength(candles, minLength, "Keltner Channels"); err != nil {
			return nil, err
		}

		// Prepare data for ta-lib
		data, err := PrepareData(candles)
		if err != nil {
			return nil, err
		}

		// Calculate EMA (middle line)
		emaValues := talib.Ema(data.Close, emaPeriod)
		middleLine, err := GetLastValue(emaValues)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get EMA value")
		}

		// Calculate ATR
		atrValues := talib.Atr(data.High, data.Low, data.Close, atrPeriod)
		atr, err := GetLastValue(atrValues)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get ATR value")
		}

		// Calculate upper and lower channels
		upperChannel := middleLine + (atr * multiplier)
		lowerChannel := middleLine - (atr * multiplier)

		currentPrice := candles[0].Close

		// Calculate channel width (similar to Bollinger bandwidth)
		channelWidth := ((upperChannel - lowerChannel) / middleLine) * 100

		// Determine position
		position := "middle"
		if currentPrice >= upperChannel {
			position = "above_upper"
		} else if currentPrice <= lowerChannel {
			position = "below_lower"
		} else if currentPrice > middleLine {
			position = "upper_half"
		} else {
			position = "lower_half"
		}

		// Generate signal
		signal := "neutral"
		if position == "above_upper" {
			signal = "overbought"
		} else if position == "below_lower" {
			signal = "oversold"
		} else if currentPrice > middleLine {
			signal = "bullish"
		} else {
			signal = "bearish"
		}

		return map[string]interface{}{
			"upper":         upperChannel,
			"middle":        middleLine,
			"lower":         lowerChannel,
			"atr":           atr,
			"channel_width": channelWidth,
			"current_price": currentPrice,
			"position":      position,
			"signal":        signal,
		}, nil
	})
}

// max returns maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

