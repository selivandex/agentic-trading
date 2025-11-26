package orderflow

import (
	"context"

	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewGetTradeImbalanceTool calculates buy vs sell pressure from recent trades
func NewGetTradeImbalanceTool(deps shared.Deps) tool.Tool {
	return functiontool.New("get_trade_imbalance", "Get Trade Imbalance (Buy vs Sell Pressure)", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		if !deps.HasMarketData() {
			return nil, errors.Wrapf(errors.ErrInternal, "market data repository not configured")
		}

		exchange, _ := args["exchange"].(string)
		symbol, _ := args["symbol"].(string)
		limit := parseLimit(args["limit"], 100)

		if exchange == "" || symbol == "" {
			return nil, errors.Wrapf(errors.ErrInvalidInput, "exchange and symbol required")
		}

		// Get recent trades
		trades, err := deps.MarketDataRepo.GetRecentTrades(ctx, exchange, symbol, limit)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get recent trades")
		}

		if len(trades) == 0 {
			return nil, errors.Wrapf(errors.ErrNotFound, "no trades found")
		}

		// Calculate buy/sell volumes
		buyVolume := 0.0
		sellVolume := 0.0
		buyCount := 0
		sellCount := 0

		for _, trade := range trades {
			if trade.Side == "buy" {
				buyVolume += trade.Quantity
				buyCount++
			} else {
				sellVolume += trade.Quantity
				sellCount++
			}
		}

		totalVolume := buyVolume + sellVolume
		delta := buyVolume - sellVolume
		deltaPct := (delta / totalVolume) * 100

		// Determine pressure
		pressure := "balanced"
		if deltaPct > 30 {
			pressure = "strong_buy"
		} else if deltaPct > 10 {
			pressure = "buy"
		} else if deltaPct < -30 {
			pressure = "strong_sell"
		} else if deltaPct < -10 {
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
			"buy_volume":      buyVolume,
			"sell_volume":     sellVolume,
			"buy_count":       buyCount,
			"sell_count":      sellCount,
			"total_volume":    totalVolume,
			"delta":           delta,
			"delta_pct":       deltaPct,
			"pressure":        pressure,
			"signal":          signal,
			"trades_analyzed": len(trades),
		}, nil
	})
}

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
