package market

import (
	"context"
	"time"

	"prometheus/internal/tools/shared"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"prometheus/pkg/errors"
)

// NewGetOHLCVTool loads historical candles for a symbol/timeframe.
func NewGetOHLCVTool(deps shared.Deps) tool.Tool {
	return functiontool.New("get_ohlcv", "Retrieve historical OHLCV candles", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		if !deps.HasMarketData() {
			return nil, errors.Wrapf(errors.ErrInternal, "get_ohlcv: market data repository not configured")
		}

		exchange, _ := args["exchange"].(string)
		symbol, _ := args["symbol"].(string)
		timeframe, _ := args["timeframe"].(string)
		limit := parseLimit(args["limit"], 100)
		if exchange == "" || symbol == "" || timeframe == "" {
			return nil, errors.ErrInvalidInput
		}

		candles, err := deps.MarketDataRepo.GetLatestOHLCV(ctx, exchange, symbol, timeframe, limit)
		if err != nil {
			return nil, errors.Wrap(err, "get_ohlcv: fetch candles")
		}

		data := make([]map[string]interface{}, 0, len(candles))
		for _, c := range candles {
			data = append(data, map[string]interface{}{
				"open_time":    c.OpenTime.Format(time.RFC3339),
				"open":         c.Open,
				"high":         c.High,
				"low":          c.Low,
				"close":        c.Close,
				"volume":       c.Volume,
				"quote_volume": c.QuoteVolume,
				"trades":       c.Trades,
			})
		}

		return map[string]interface{}{"candles": data}, nil
	})
}

func parseLimit(raw interface{}, fallback int) int {
	switch v := raw.(type) {
	case int:
		if v > 0 {
			return v
		}
	case float64:
		if int(v) > 0 {
			return int(v)
		}
	}
	return fallback
}
