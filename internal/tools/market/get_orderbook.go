package market

import (
	"context"

	"prometheus/internal/tools/shared"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"prometheus/pkg/errors"
)

// NewGetOrderBookTool returns an order book snapshot tool.
func NewGetOrderBookTool(deps shared.Deps) tool.Tool {
	return functiontool.New("get_orderbook", "Get depth snapshot for a trading pair", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		if !deps.HasMarketData() {
			return nil, errors.Wrapf(errors.ErrInternal, "get_orderbook: market data repository not configured")
		}

		exchange, _ := args["exchange"].(string)
		symbol, _ := args["symbol"].(string)
		if exchange == "" || symbol == "" {
			return nil, errors.ErrInvalidInput
		}

		snapshot, err := deps.MarketDataRepo.GetLatestOrderBook(ctx, exchange, symbol)
		if err != nil {
			return nil, errors.Wrap(err, "get_orderbook: fetch snapshot")
		}
		if snapshot == nil {
			return nil, errors.Wrapf(errors.ErrInternal, "get_orderbook: snapshot not found")
		}

		return map[string]interface{}{
			"exchange":  snapshot.Exchange,
			"symbol":    snapshot.Symbol,
			"timestamp": snapshot.Timestamp,
			"bids":      snapshot.Bids,
			"asks":      snapshot.Asks,
			"bid_depth": snapshot.BidDepth,
			"ask_depth": snapshot.AskDepth,
		}, nil
	})
}
