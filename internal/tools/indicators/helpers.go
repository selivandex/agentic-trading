package indicators

import (
	"context"
	"fmt"

	"prometheus/internal/domain/market_data"
	"prometheus/internal/tools/shared"
)

func loadCandles(ctx context.Context, deps shared.Deps, args map[string]interface{}, defaultLimit int) ([]market_data.OHLCV, error) {
	if !deps.HasMarketData() {
		return nil, fmt.Errorf("indicator: market data repository not configured")
	}

	exchange, _ := args["exchange"].(string)
	symbol, _ := args["symbol"].(string)
	timeframe, _ := args["timeframe"].(string)
	limit := parseLimit(args["limit"], defaultLimit)
	if exchange == "" || symbol == "" || timeframe == "" {
		return nil, fmt.Errorf("indicator: exchange, symbol, and timeframe are required")
	}

	candles, err := deps.MarketDataRepo.GetLatestOHLCV(ctx, exchange, symbol, timeframe, limit)
	if err != nil {
		return nil, fmt.Errorf("indicator: fetch candles: %w", err)
	}
	if len(candles) == 0 {
		return nil, fmt.Errorf("indicator: no candles available")
	}
	return candles, nil
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

func extractCloses(candles []market_data.OHLCV) []float64 {
	closes := make([]float64, 0, len(candles))
	for i := len(candles) - 1; i >= 0; i-- { // ensure chronological order
		closes = append(closes, candles[i].Close)
	}
	return closes
}

func extractHighLow(candles []market_data.OHLCV) ([]float64, []float64, []float64) {
	high := make([]float64, 0, len(candles))
	low := make([]float64, 0, len(candles))
	close := make([]float64, 0, len(candles))
	for i := len(candles) - 1; i >= 0; i-- {
		high = append(high, candles[i].High)
		low = append(low, candles[i].Low)
		close = append(close, candles[i].Close)
	}
	return high, low, close
}
