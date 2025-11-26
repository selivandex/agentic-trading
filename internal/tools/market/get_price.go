package market

import (
	"context"
	"fmt"
	"time"

	"prometheus/internal/tools/shared"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewGetPriceTool returns a tool that fetches the latest ticker snapshot.
func NewGetPriceTool(deps shared.Deps) tool.Tool {
	return functiontool.New("get_price", "Fetch current price with bid/ask spread", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		if !deps.HasMarketData() {
			deps.Log.Error("Tool: get_price called without market data repository")
			return nil, fmt.Errorf("get_price: market data repository not configured")
		}

		exchange, _ := args["exchange"].(string)
		symbol, _ := args["symbol"].(string)
		if exchange == "" || symbol == "" {
			deps.Log.Warn("Tool: get_price called with missing parameters", "exchange", exchange, "symbol", symbol)
			return nil, fmt.Errorf("get_price: exchange and symbol are required")
		}

		deps.Log.Debug("Tool: get_price called", "exchange", exchange, "symbol", symbol)

		ticker, err := deps.MarketDataRepo.GetLatestTicker(ctx, exchange, symbol)
		if err != nil {
			deps.Log.Error("Tool: get_price failed", "exchange", exchange, "symbol", symbol, "error", err)
			return nil, fmt.Errorf("get_price: fetch ticker: %w", err)
		}
		if ticker == nil {
			deps.Log.Warn("Tool: get_price ticker not found", "exchange", exchange, "symbol", symbol)
			return nil, fmt.Errorf("get_price: ticker not found")
		}

		result := map[string]interface{}{
			"exchange":     ticker.Exchange,
			"symbol":       ticker.Symbol,
			"price":        ticker.Price,
			"bid":          ticker.Bid,
			"ask":          ticker.Ask,
			"fundingRate":  ticker.FundingRate,
			"openInterest": ticker.OpenInterest,
			"timestamp":    ticker.Timestamp.Format(time.RFC3339),
		}

		deps.Log.Info("Tool: get_price success", "exchange", exchange, "symbol", symbol, "price", ticker.Price)
		return result, nil
	})
}
