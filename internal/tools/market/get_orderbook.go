package market

import (
	"context"
	"fmt"

	"prometheus/internal/tools/shared"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewGetOrderBookTool returns an order book snapshot tool.
func NewGetOrderBookTool(deps shared.Deps) tool.Tool {
	return functiontool.New("get_orderbook", "Get depth snapshot for a trading pair", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		if !deps.HasMarketData() {
			return nil, fmt.Errorf("get_orderbook: market data repository not configured")
		}

		exchange, _ := args["exchange"].(string)
		symbol, _ := args["symbol"].(string)
		if exchange == "" || symbol == "" {
			return nil, fmt.Errorf("get_orderbook: exchange and symbol are required")
		}

		snapshot, err := deps.MarketDataRepo.GetLatestOrderBook(ctx, exchange, symbol)
		if err != nil {
			return nil, fmt.Errorf("get_orderbook: fetch snapshot: %w", err)
		}
		if snapshot == nil {
			return nil, fmt.Errorf("get_orderbook: snapshot not found")
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
