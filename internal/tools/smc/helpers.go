package smc

import (
	"context"

	"prometheus/internal/domain/market_data"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
)

// Re-export for use in SMC tools
type OHLCV = market_data.OHLCV

// loadCandles loads OHLCV candles for SMC analysis
func loadCandles(ctx context.Context, deps shared.Deps, args map[string]interface{}, defaultLimit int) ([]market_data.OHLCV, error) {
	if !deps.HasMarketData() {
		return nil, errors.Wrapf(errors.ErrInternal, "smc: market data repository not configured")
	}

	exchange, _ := args["exchange"].(string)
	symbol, _ := args["symbol"].(string)
	timeframe, _ := args["timeframe"].(string)
	limit := parseLimit(args["limit"], defaultLimit)

	if exchange == "" || symbol == "" || timeframe == "" {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "exchange, symbol, and timeframe are required")
	}

	candles, err := deps.MarketDataRepo.GetLatestOHLCV(ctx, exchange, symbol, timeframe, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load candles")
	}

	if len(candles) == 0 {
		return nil, errors.Wrapf(errors.ErrNotFound, "no candles found for %s %s %s", exchange, symbol, timeframe)
	}

	return candles, nil
}

// parseLimit parses limit parameter with default
func parseLimit(val interface{}, defaultVal int) int {
	if val == nil {
		return defaultVal
	}

	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return defaultVal
	}
}

