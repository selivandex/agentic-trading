package market

import (
	"context"
	"fmt"
	"time"

	"prometheus/internal/tools/shared"

	"google.golang.org/adk/tool/functiontool"
)

// NewGetPriceTool returns a tool that fetches the latest ticker snapshot.
func NewGetPriceTool(deps shared.Deps) *functiontool.Tool {
	return functiontool.New("get_price", "Fetch current price with bid/ask spread", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		if !deps.HasMarketData() {
			return nil, fmt.Errorf("get_price: market data repository not configured")
		}

		exchange, _ := args["exchange"].(string)
		symbol, _ := args["symbol"].(string)
		if exchange == "" || symbol == "" {
			return nil, fmt.Errorf("get_price: exchange and symbol are required")
		}

		ticker, err := deps.MarketDataRepo.GetLatestTicker(ctx, exchange, symbol)
		if err != nil {
			return nil, fmt.Errorf("get_price: fetch ticker: %w", err)
		}
		if ticker == nil {
			return nil, fmt.Errorf("get_price: ticker not found")
		}

		return map[string]interface{}{
			"exchange":     ticker.Exchange,
			"symbol":       ticker.Symbol,
			"price":        ticker.Price,
			"bid":          ticker.Bid,
			"ask":          ticker.Ask,
			"fundingRate":  ticker.FundingRate,
			"openInterest": ticker.OpenInterest,
			"timestamp":    ticker.Timestamp.Format(time.RFC3339),
		}, nil
	})
}
