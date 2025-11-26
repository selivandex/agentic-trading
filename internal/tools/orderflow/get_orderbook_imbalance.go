package orderflow
import (
	"encoding/json"
	"time"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
	"google.golang.org/adk/tool"
)
// OrderBookLevel represents a single price level in order book
type OrderBookLevel struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
}
// NewGetOrderbookImbalanceTool analyzes bid/ask imbalance in order book
func NewGetOrderbookImbalanceTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"get_orderbook_imbalance",
		"Get OrderBook Imbalance (Bid/Ask Delta)",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			if !deps.HasMarketData() {
				return nil, errors.Wrapf(errors.ErrInternal, "market data repository not configured")
			}
			exchange, _ := args["exchange"].(string)
			symbol, _ := args["symbol"].(string)
			depth := parseLimit(args["depth"], 20)
			if exchange == "" || symbol == "" {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "exchange and symbol required")
			}
			// Get latest order book
			ob, err := deps.MarketDataRepo.GetLatestOrderBook(ctx, exchange, symbol)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get order book")
			}
			// Parse bids and asks
			var bids, asks []OrderBookLevel
			json.Unmarshal([]byte(ob.Bids), &bids)
			json.Unmarshal([]byte(ob.Asks), &asks)
			// Calculate total volumes
			bidVolume := 0.0
			askVolume := 0.0
			// Sum top N levels
			for i := 0; i < depth && i < len(bids); i++ {
				bidVolume += bids[i].Amount
			}
			for i := 0; i < depth && i < len(asks); i++ {
				askVolume += asks[i].Amount
			}
			totalVolume := bidVolume + askVolume
			imbalance := bidVolume - askVolume
			imbalancePct := (imbalance / totalVolume) * 100
			bidAskRatio := bidVolume / askVolume
			// Analyze imbalance
			pressure := "balanced"
			if imbalancePct > 30 {
				pressure = "strong_bid"
			} else if imbalancePct > 10 {
				pressure = "bid"
			} else if imbalancePct < -30 {
				pressure = "strong_ask"
			} else if imbalancePct < -10 {
				pressure = "ask"
			}
			// Signal
			signal := "neutral"
			if pressure == "strong_bid" {
				signal = "bullish" // Strong buying interest
			} else if pressure == "strong_ask" {
				signal = "bearish" // Strong selling interest
			}
			// Find largest bid/ask walls
			largestBid := OrderBookLevel{}
			largestAsk := OrderBookLevel{}
			for _, bid := range bids {
				if bid.Amount > largestBid.Amount {
					largestBid = bid
				}
			}
			for _, ask := range asks {
				if ask.Amount > largestAsk.Amount {
					largestAsk = ask
				}
			}
			return map[string]interface{}{
				"bid_volume":     bidVolume,
				"ask_volume":     askVolume,
				"total_volume":   totalVolume,
				"imbalance":      imbalance,
				"imbalance_pct":  imbalancePct,
				"bid_ask_ratio":  bidAskRatio,
				"pressure":       pressure,
				"signal":         signal,
				"largest_bid":    largestBid,
				"largest_ask":    largestAsk,
				"depth_analyzed": depth,
			}, nil
		},
		deps,
	).
		WithTimeout(10*time.Second).
		WithRetry(3, 500*time.Millisecond).
		Build()
}
