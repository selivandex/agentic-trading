package orderflow

import (
	"context"

	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// WhaleTrade represents a large trade
type WhaleTrade struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Quantity  float64 `json:"quantity"`
	ValueUSD  float64 `json:"value_usd"`
	Side      string  `json:"side"`
	Timestamp int64   `json:"timestamp"`
}

// NewGetWhaleTradesToolcollects large transactions (whale activity)
func NewGetWhaleTradesTool(deps shared.Deps) tool.Tool {
	return functiontool.New("get_whale_trades", "Get Whale Trades (Large Transactions)", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		if !deps.HasMarketData() {
			return nil, errors.Wrapf(errors.ErrInternal, "market data repository not configured")
		}

		exchange, _ := args["exchange"].(string)
		symbol, _ := args["symbol"].(string)
		limit := parseLimit(args["limit"], 200)
		minUSDValue := parseFloat(args["min_usd"], 100000.0) // Default $100k

		if exchange == "" || symbol == "" {
			return nil, errors.Wrapf(errors.ErrInvalidInput, "exchange and symbol required")
		}

		// Get recent trades
		trades, err := deps.MarketDataRepo.GetRecentTrades(ctx, exchange, symbol, limit)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get recent trades")
		}

		// Filter whale trades
		whaleTrades := make([]WhaleTrade, 0)
		totalWhaleVolumeUSD := 0.0

		for _, trade := range trades {
			valueUSD := trade.Price * trade.Quantity

			if valueUSD >= minUSDValue {
				whaleTrades = append(whaleTrades, WhaleTrade{
					Symbol:    symbol,
					Price:     trade.Price,
					Quantity:  trade.Quantity,
					ValueUSD:  valueUSD,
					Side:      trade.Side,
					Timestamp: trade.Timestamp.Unix(),
				})

				totalWhaleVolumeUSD += valueUSD
			}
		}

		// Analyze whale activity
		whaleBuyVolume := 0.0
		whaleSellVolume := 0.0

		for _, wt := range whaleTrades {
			if wt.Side == "buy" {
				whaleBuyVolume += wt.ValueUSD
			} else {
				whaleSellVolume += wt.ValueUSD
			}
		}

		whaleImbalance := whaleBuyVolume - whaleSellVolume
		whaleImbalancePct := 0.0
		if totalWhaleVolumeUSD > 0 {
			whaleImbalancePct = (whaleImbalance / totalWhaleVolumeUSD) * 100
		}

		// Signal
		signal := "neutral"
		whaleActivity := "none"

		if len(whaleTrades) > 0 {
			whaleActivity = "present"

			if whaleImbalancePct > 30 {
				signal = "whale_buying"
			} else if whaleImbalancePct < -30 {
				signal = "whale_selling"
			}
		}

		return map[string]interface{}{
			"whale_trades":        whaleTrades[:min(len(whaleTrades), 10)], // Return top 10
			"whale_count":         len(whaleTrades),
			"whale_buy_volume":    whaleBuyVolume,
			"whale_sell_volume":   whaleSellVolume,
			"whale_imbalance":     whaleImbalance,
			"whale_imbalance_pct": whaleImbalancePct,
			"total_whale_volume":  totalWhaleVolumeUSD,
			"whale_activity":      whaleActivity,
			"signal":              signal,
			"min_threshold_usd":   minUSDValue,
		}, nil
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func parseFloat(val interface{}, defaultVal float64) float64 {
	if val == nil {
		return defaultVal
	}

	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return defaultVal
	}
}
