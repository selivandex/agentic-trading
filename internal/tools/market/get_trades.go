package market

import (
	"time"

	"prometheus/internal/tools/shared"

	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewGetTradesTool streams recent trades.
func NewGetTradesTool(deps shared.Deps) tool.Tool {
	t, _ := functiontool.New(
		functiontool.Config{
			Name:        "get_trades",
			Description: "Get recent trades tape",
		},
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			if !deps.HasMarketData() {
				return nil, errors.Wrapf(errors.ErrInternal, "get_trades: market data repository not configured")
			}

			exchange, _ := args["exchange"].(string)
			symbol, _ := args["symbol"].(string)
			limit := parseLimit(args["limit"], 100)
			if exchange == "" || symbol == "" {
				return nil, errors.ErrInvalidInput
			}

			trades, err := deps.MarketDataRepo.GetRecentTrades(ctx, exchange, symbol, limit)
			if err != nil {
				return nil, errors.Wrap(err, "get_trades: fetch trades")
			}

			data := make([]map[string]interface{}, 0, len(trades))
			for _, t := range trades {
				data = append(data, map[string]interface{}{
					"trade_id":  t.TradeID,
					"price":     t.Price,
					"quantity":  t.Quantity,
					"side":      t.Side,
					"is_buyer":  t.IsBuyer,
					"timestamp": t.Timestamp.Format(time.RFC3339),
				})
			}

			return map[string]interface{}{"trades": data}, nil
		},
	)
	return t
}
