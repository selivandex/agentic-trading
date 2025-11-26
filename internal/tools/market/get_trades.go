package market

import (
	"context"
	"time"

	"prometheus/internal/tools/shared"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"prometheus/pkg/errors"
)

// NewGetTradesTool returns a tool fetching recent trades from storage.
func NewGetTradesTool(deps shared.Deps) tool.Tool {
	return functiontool.New("get_trades", "Fetch recent trades", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		if !deps.HasMarketData() {
			return nil, errors.Wrapf(errors.ErrInternal, "get_trades: market data repository not configured")
		}

		exchange, _ := args["exchange"].(string)
		symbol, _ := args["symbol"].(string)
		limit := parseLimit(args["limit"], 50)
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
	})
}
