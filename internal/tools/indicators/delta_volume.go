package indicators

import (
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewDeltaVolumeTool computes Buy vs Sell Volume Delta
// Requires trades data (tape) to determine actual buy/sell volume
// Falls back to heuristic if trades data unavailable
func NewDeltaVolumeTool(deps shared.Deps) tool.Tool {
	t, _ := functiontool.New(
		functiontool.Config{
			Name:        "delta_volume",
			Description: "Buy/Sell Volume Delta",
		},
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
		// Get symbol and exchange
		symbol, ok := args["symbol"].(string)
		if !ok || symbol == "" {
			return nil, errors.Wrapf(errors.ErrInvalidInput, "symbol required")
		}

		exchange := "binance" // Default
		if ex, ok := args["exchange"].(string); ok {
			exchange = ex
		}

		limit := parseLimit(args["limit"], 100)

		// Try to get actual trades data
		trades, err := deps.MarketDataRepo.GetRecentTrades(ctx, exchange, symbol, limit)
		if err != nil || len(trades) == 0 {
			// Fallback to candle-based heuristic
			return calculateDeltaFromCandles(ctx, deps, args)
		}

		// Calculate actual buy/sell volume from trades
		buyVolume := 0.0
		sellVolume := 0.0

		for _, trade := range trades {
			if trade.Side == "buy" {
				buyVolume += trade.Quantity
			} else {
				sellVolume += trade.Quantity
			}
		}

		delta := buyVolume - sellVolume
		totalVolume := buyVolume + sellVolume

		// Calculate delta percentage
		deltaPct := 0.0
		if totalVolume > 0 {
			deltaPct = (delta / totalVolume) * 100
		}

		// Determine pressure
		pressure := "balanced"
		if deltaPct > 20 {
			pressure = "strong_buy"
		} else if deltaPct > 5 {
			pressure = "buy"
		} else if deltaPct < -20 {
			pressure = "strong_sell"
		} else if deltaPct < -5 {
			pressure = "sell"
		}

		// Signal
		signal := "neutral"
		if pressure == "strong_buy" {
			signal = "bullish"
		} else if pressure == "strong_sell" {
			signal = "bearish"
		}

		return map[string]interface{}{
			"buy_volume":   buyVolume,
			"sell_volume":  sellVolume,
			"delta":        delta,
			"delta_pct":    deltaPct,
			"total_volume": totalVolume,
			"pressure":     pressure,
			"signal":       signal,
			"data_source":  "trades",
		}, nil
	})
	return t
}

// calculateDeltaFromCandles estimates delta from candle data (less accurate)
func calculateDeltaFromCandles(ctx tool.Context, deps shared.Deps, args map[string]interface{}) (map[string]interface{}, error) {
	candles, err := loadCandles(ctx, deps, args, 50)
	if err != nil {
		return nil, err
	}

	if len(candles) == 0 {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "no candles for delta volume")
	}

	// Heuristic: if close > open, assume more buying; if close < open, more selling
	buyVolume := 0.0
	sellVolume := 0.0

	for _, candle := range candles {
		if candle.Close > candle.Open {
			// Bullish candle - assume buying pressure
			ratio := (candle.Close - candle.Open) / (candle.High - candle.Low)
			buyVolume += candle.Volume * ratio
			sellVolume += candle.Volume * (1 - ratio)
		} else if candle.Close < candle.Open {
			// Bearish candle - assume selling pressure
			ratio := (candle.Open - candle.Close) / (candle.High - candle.Low)
			sellVolume += candle.Volume * ratio
			buyVolume += candle.Volume * (1 - ratio)
		} else {
			// Doji - split 50/50
			buyVolume += candle.Volume * 0.5
			sellVolume += candle.Volume * 0.5
		}
	}

	delta := buyVolume - sellVolume
	totalVolume := buyVolume + sellVolume
	deltaPct := (delta / totalVolume) * 100

	pressure := "balanced"
	if deltaPct > 15 {
		pressure = "buy"
	} else if deltaPct < -15 {
		pressure = "sell"
	}

	return map[string]interface{}{
		"buy_volume":   buyVolume,
		"sell_volume":  sellVolume,
		"delta":        delta,
		"delta_pct":    deltaPct,
		"total_volume": totalVolume,
		"pressure":     pressure,
		"signal":       "neutral",
		"data_source":  "candles_heuristic",
		"note":         "Using candle-based estimation. For accurate data, use trades stream.",
	}, nil
}
