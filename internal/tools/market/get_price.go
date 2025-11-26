package market

import (
	"context"
	"time"

	"prometheus/internal/tools/shared"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"prometheus/pkg/errors"
)

// NewGetPriceTool returns a tool that fetches the latest ticker snapshot.
func NewGetPriceTool(deps shared.Deps) tool.Tool {
	return functiontool.New("get_price", "Fetch current price with bid/ask spread", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		if !deps.HasMarketData() {
			return nil, errors.Wrapf(errors.ErrInternal, "get_price: market data repository not configured")
		}

		exchange, _ := args["exchange"].(string)
		symbol, _ := args["symbol"].(string)
		if exchange == "" || symbol == "" {
			return nil, errors.ErrInvalidInput
		}

		ticker, err := deps.MarketDataRepo.GetLatestTicker(ctx, exchange, symbol)
		if err != nil {
			return nil, errors.Wrap(err, "get_price: fetch ticker")
		}
		if ticker == nil {
			return nil, errors.Wrapf(errors.ErrInternal, "get_price: ticker not found")
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
