package smc

import (
	"prometheus/internal/domain/market_data"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
)

// Re-export for use in SMC tools
type OHLCV = market_data.OHLCV

// loadCandles loads OHLCV candles for SMC analysis
func loadCandles(ctx tool.Context, deps shared.Deps, args map[string]interface{}, defaultLimit int) ([]market_data.OHLCV, error) {
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

// swingPoint represents a swing high or low with its price and index
type swingPoint struct {
	price float64
	index int
}

// findSwingHighsWithIndex finds swing highs in candle data
func findSwingHighsWithIndex(candles []market_data.OHLCV, lookback int) []swingPoint {
	points := make([]swingPoint, 0)
	for i := lookback; i < len(candles)-lookback; i++ {
		candle := candles[i]
		isSwing := true
		for j := 1; j <= lookback; j++ {
			if candles[i+j].High >= candle.High || candles[i-j].High >= candle.High {
				isSwing = false
				break
			}
		}
		if isSwing {
			points = append(points, swingPoint{price: candle.High, index: i})
		}
	}
	return points
}

// findSwingLowsWithIndex finds swing lows in candle data
func findSwingLowsWithIndex(candles []market_data.OHLCV, lookback int) []swingPoint {
	points := make([]swingPoint, 0)
	for i := lookback; i < len(candles)-lookback; i++ {
		candle := candles[i]
		isSwing := true
		for j := 1; j <= lookback; j++ {
			if candles[i+j].Low <= candle.Low || candles[i-j].Low <= candle.Low {
				isSwing = false
				break
			}
		}
		if isSwing {
			points = append(points, swingPoint{price: candle.Low, index: i})
		}
	}
	return points
}

// analyzeSwingProgression analyzes swing points to determine trend
func analyzeSwingProgression(highs, lows []swingPoint) (trend string, hh, hl, lh, ll []float64) {
	hh = make([]float64, 0)
	hl = make([]float64, 0)
	lh = make([]float64, 0)
	ll = make([]float64, 0)

	// Analyze highs
	for i := 1; i < len(highs); i++ {
		if highs[i].price > highs[i-1].price {
			hh = append(hh, highs[i].price) // Higher high
		} else {
			lh = append(lh, highs[i].price) // Lower high
		}
	}

	// Analyze lows
	for i := 1; i < len(lows); i++ {
		if lows[i].price > lows[i-1].price {
			hl = append(hl, lows[i].price) // Higher low
		} else {
			ll = append(ll, lows[i].price) // Lower low
		}
	}

	// Determine trend
	if len(hh) >= 2 && len(hl) >= 2 {
		trend = "uptrend" // Making HH and HL
	} else if len(lh) >= 2 && len(ll) >= 2 {
		trend = "downtrend" // Making LH and LL
	} else {
		trend = "ranging"
	}

	return trend, hh, hl, lh, ll
}
