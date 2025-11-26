package market

import (
	"prometheus/internal/tools/shared"

	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewGetOrderBookTool returns an order book snapshot tool.
func NewGetOrderBookTool(deps shared.Deps) tool.Tool {
	t, _ := functiontool.New(
		functiontool.Config{
			Name:        "get_orderbook",
			Description: "Get depth snapshot for a trading pair",
		},
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
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
		},
	)
	return t
}
