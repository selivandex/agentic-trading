package market

import (
	"encoding/json"
	"math"
	"time"

	"prometheus/internal/domain/market_data"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
	"prometheus/pkg/templates"

	"google.golang.org/adk/tool"
)

// NewMarketAnalysisTool returns comprehensive real-time market analysis in one call.
// Combines: price, order flow (trade imbalance, CVD, whales), orderbook analysis, tick speed.
// Returns human-readable text format optimized for LLM consumption.
func NewMarketAnalysisTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"get_market_analysis",
		"Get comprehensive real-time market analysis. Returns price, spread, order flow, CVD, whale activity, orderbook pressure, and overall signal in readable format.",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			if !deps.HasMarketData() {
				return nil, errors.Wrapf(errors.ErrInternal, "market data repository not configured")
			}

			exchange, _ := args["exchange"].(string)
			symbol, _ := args["symbol"].(string)
			if exchange == "" || symbol == "" {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "exchange and symbol required")
			}

			// Collect all data
			var price priceData
			var orderbook orderbookData
			var tradeFlow tradeFlowData
			var volume volumeData
			var tickSpeed tickSpeedData

			// 1. Get orderbook data
			ob, err := deps.MarketDataRepo.GetLatestOrderBook(ctx, exchange, symbol)
			if err == nil && ob != nil {
				price, orderbook = analyzeOrderbookData(ob.Bids, ob.Asks)
			}

			// 2. Get trades for order flow
			trades, err := deps.MarketDataRepo.GetRecentTrades(ctx, exchange, symbol, 500)
			if err == nil && len(trades) > 0 {
				price.lastTrade = trades[0].Price
				if len(trades) > 1 {
					price.change = trades[0].Price - trades[len(trades)-1].Price
					if trades[len(trades)-1].Price > 0 {
						price.changePct = (price.change / trades[len(trades)-1].Price) * 100
					}
				}
				tradeFlow = analyzeTradeFlow(trades)
				volume = analyzeVolume(trades)
				tickSpeed = analyzeSpeed(trades)
			}

			// Generate overall signal
			signal := computeSignal(tradeFlow, orderbook)

			// Prepare data for template
			templateData := map[string]interface{}{
				"Exchange":  exchange,
				"Symbol":    symbol,
				"Timestamp": time.Now().UTC().Format(time.RFC3339),
				"Price": map[string]interface{}{
					"Current":       price.current,
					"Bid":           price.bid,
					"Ask":           price.ask,
					"BidSize":       price.bidSize,
					"AskSize":       price.askSize,
					"Spread":        price.spread,
					"SpreadPct":     price.spreadPct,
					"SpreadQuality": price.spreadQuality,
					"Change":        price.change,
					"ChangePct":     price.changePct,
				},
				"TradeImbalance": map[string]interface{}{
					"BuyVolume":  tradeFlow.buyVol,
					"SellVolume": tradeFlow.sellVol,
					"DeltaPct":   tradeFlow.deltaPct,
					"Pressure":   tradeFlow.pressure,
					"Signal":     tradeFlow.signal,
				},
				"CVD": map[string]interface{}{
					"CVD":        tradeFlow.cvd,
					"Trend":      tradeFlow.cvdTrend,
					"Divergence": tradeFlow.cvdDivergence,
					"Signal":     tradeFlow.cvdSignal,
				},
				"WhaleActivity": map[string]interface{}{
					"WhaleCount":    tradeFlow.whaleCount,
					"BuyVolumeUSD":  tradeFlow.whaleBuyUSD,
					"SellVolumeUSD": tradeFlow.whaleSellUSD,
					"ImbalancePct":  tradeFlow.whaleImbalancePct,
					"Activity":      tradeFlow.whaleActivity,
					"Signal":        tradeFlow.whaleSignal,
				},
				"Orderbook": map[string]interface{}{
					"BidVolume":    orderbook.bidVol,
					"AskVolume":    orderbook.askVol,
					"ImbalancePct": orderbook.imbalancePct,
					"Pressure":     orderbook.pressure,
					"Signal":       orderbook.signal,
					"BidWall":      formatWall(orderbook.bidWall),
					"AskWall":      formatWall(orderbook.askWall),
				},
				"Volume": map[string]interface{}{
					"TotalUSD":       volume.totalUSD,
					"BuyUSD":         volume.buyUSD,
					"SellUSD":        volume.sellUSD,
					"HourlyEstimate": volume.hourlyEstimate,
					"PeriodMinutes":  volume.periodMinutes,
				},
				"TickSpeed": map[string]interface{}{
					"TradesPerMin": tickSpeed.tradesPerMin,
					"VolumePerMin": tickSpeed.volumePerMin,
					"Activity":     tickSpeed.activity,
					"Signal":       tickSpeed.signal,
				},
				"Signal": map[string]interface{}{
					"Direction":    signal.direction,
					"Strength":     signal.strength,
					"Confidence":   signal.confidence,
					"BullishCount": signal.bullish,
					"BearishCount": signal.bearish,
				},
			}

			// Render using template
			tmpl := templates.Get()
			analysis, err := tmpl.Render("tools/market_analysis", templateData)
			if err != nil {
				return nil, errors.Wrap(err, "failed to render market analysis template")
			}

			return map[string]interface{}{
				"analysis": analysis,
			}, nil
		},
		deps,
	).
		WithTimeout(30*time.Second).
		WithRetry(2, 1*time.Second).
		Build()
}

// Data structures for analysis
type priceData struct {
	current, bid, ask, bidSize, askSize float64
	spread, spreadPct                   float64
	spreadQuality                       string
	lastTrade, change, changePct        float64
}

type orderbookData struct {
	bidVol, askVol, imbalancePct float64
	pressure, signal             string
	bidWall, askWall             *wallData
}

type wallData struct {
	price, size, ratio float64
}

type tradeFlowData struct {
	buyVol, sellVol, deltaPct float64
	pressure, signal          string
	tradeCount                int
	// CVD
	cvd                     float64
	cvdTrend, cvdDivergence string
	cvdSignal               string
	// Whales
	whaleCount                 int
	whaleBuyUSD, whaleSellUSD  float64
	whaleImbalancePct          float64
	whaleActivity, whaleSignal string
}

type volumeData struct {
	totalUSD, buyUSD, sellUSD float64
	hourlyEstimate            float64
	periodMinutes             float64
}

type tickSpeedData struct {
	tradesPerMin, volumePerMin float64
	activity, signal           string
}

type signalData struct {
	direction, strength string
	confidence          float64
	bullish, bearish    int
}

// OrderBookLevel represents a single price level in order book
type OrderBookLevel struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
}

func analyzeOrderbookData(bidsJSON, asksJSON string) (priceData, orderbookData) {
	var bids, asks []OrderBookLevel
	json.Unmarshal([]byte(bidsJSON), &bids)
	json.Unmarshal([]byte(asksJSON), &asks)

	var price priceData
	var ob orderbookData

	if len(bids) > 0 {
		price.bid = bids[0].Price
		price.bidSize = bids[0].Amount
	}
	if len(asks) > 0 {
		price.ask = asks[0].Price
		price.askSize = asks[0].Amount
	}
	if len(bids) > 0 && len(asks) > 0 {
		mid := (bids[0].Price + asks[0].Price) / 2
		spread := asks[0].Price - bids[0].Price
		spreadPct := (spread / mid) * 100

		price.current = math.Round(mid*100) / 100
		price.spread = math.Round(spread*10000) / 10000
		price.spreadPct = math.Round(spreadPct*10000) / 10000

		price.spreadQuality = "normal"
		if spreadPct < 0.02 {
			price.spreadQuality = "tight"
		} else if spreadPct > 0.1 {
			price.spreadQuality = "wide"
		}
	}

	// Orderbook analysis
	depth := 20
	bidVol, askVol := 0.0, 0.0
	for i := 0; i < depth && i < len(bids); i++ {
		bidVol += bids[i].Amount
	}
	for i := 0; i < depth && i < len(asks); i++ {
		askVol += asks[i].Amount
	}

	totalVol := bidVol + askVol
	if totalVol > 0 {
		ob.bidVol = bidVol
		ob.askVol = askVol
		ob.imbalancePct = ((bidVol - askVol) / totalVol) * 100
	}

	ob.pressure = "balanced"
	if ob.imbalancePct > 30 {
		ob.pressure = "strong_bid"
	} else if ob.imbalancePct > 10 {
		ob.pressure = "bid"
	} else if ob.imbalancePct < -30 {
		ob.pressure = "strong_ask"
	} else if ob.imbalancePct < -10 {
		ob.pressure = "ask"
	}

	ob.signal = "neutral"
	if ob.pressure == "strong_bid" {
		ob.signal = "bullish"
	} else if ob.pressure == "strong_ask" {
		ob.signal = "bearish"
	}

	// Detect walls
	ob.bidWall, ob.askWall = detectOrderWalls(bids, asks, depth)

	return price, ob
}

func detectOrderWalls(bids, asks []OrderBookLevel, depth int) (*wallData, *wallData) {
	var bidWall, askWall *wallData

	if len(bids) > 0 {
		avgSize, maxSize, maxPrice := 0.0, 0.0, 0.0
		for i := 0; i < depth && i < len(bids); i++ {
			avgSize += bids[i].Amount
			if bids[i].Amount > maxSize {
				maxSize = bids[i].Amount
				maxPrice = bids[i].Price
			}
		}
		avgSize /= float64(min(depth, len(bids)))
		if maxSize > avgSize*3 {
			bidWall = &wallData{
				price: maxPrice,
				size:  maxSize,
				ratio: maxSize / avgSize,
			}
		}
	}

	if len(asks) > 0 {
		avgSize, maxSize, maxPrice := 0.0, 0.0, 0.0
		for i := 0; i < depth && i < len(asks); i++ {
			avgSize += asks[i].Amount
			if asks[i].Amount > maxSize {
				maxSize = asks[i].Amount
				maxPrice = asks[i].Price
			}
		}
		avgSize /= float64(min(depth, len(asks)))
		if maxSize > avgSize*3 {
			askWall = &wallData{
				price: maxPrice,
				size:  maxSize,
				ratio: maxSize / avgSize,
			}
		}
	}

	return bidWall, askWall
}

func analyzeTradeFlow(trades []market_data.Trade) tradeFlowData {
	var flow tradeFlowData
	if len(trades) == 0 {
		return flow
	}

	flow.tradeCount = len(trades)

	// Trade imbalance
	for _, t := range trades {
		if t.Side == "buy" {
			flow.buyVol += t.Quantity
		} else {
			flow.sellVol += t.Quantity
		}
	}

	total := flow.buyVol + flow.sellVol
	if total > 0 {
		flow.deltaPct = ((flow.buyVol - flow.sellVol) / total) * 100
	}

	flow.pressure = "balanced"
	if flow.deltaPct > 30 {
		flow.pressure = "strong_buy"
	} else if flow.deltaPct > 10 {
		flow.pressure = "buy"
	} else if flow.deltaPct < -30 {
		flow.pressure = "strong_sell"
	} else if flow.deltaPct < -10 {
		flow.pressure = "sell"
	}

	flow.signal = "neutral"
	if flow.pressure == "strong_buy" {
		flow.signal = "bullish"
	} else if flow.pressure == "strong_sell" {
		flow.signal = "bearish"
	}

	// CVD
	cvd := 0.0
	cvdStart := 0.0
	for i := len(trades) - 1; i >= 0; i-- {
		if trades[i].Side == "buy" {
			cvd += trades[i].Quantity
		} else {
			cvd -= trades[i].Quantity
		}
		if i == len(trades)-1 {
			cvdStart = cvd
		}
	}

	flow.cvd = cvd
	cvdChange := cvd - cvdStart

	flow.cvdTrend = "neutral"
	if cvdChange > 0 {
		flow.cvdTrend = "accumulation"
	} else if cvdChange < 0 {
		flow.cvdTrend = "distribution"
	}

	priceChange := trades[0].Price - trades[len(trades)-1].Price
	flow.cvdDivergence = "none"
	if priceChange > 0 && cvdChange < 0 {
		flow.cvdDivergence = "bearish"
	} else if priceChange < 0 && cvdChange > 0 {
		flow.cvdDivergence = "bullish"
	}

	flow.cvdSignal = "neutral"
	if flow.cvdTrend == "accumulation" && flow.cvdDivergence == "none" {
		flow.cvdSignal = "bullish"
	} else if flow.cvdTrend == "distribution" && flow.cvdDivergence == "none" {
		flow.cvdSignal = "bearish"
	} else if flow.cvdDivergence == "bullish" {
		flow.cvdSignal = "bullish_reversal"
	} else if flow.cvdDivergence == "bearish" {
		flow.cvdSignal = "bearish_reversal"
	}

	// Whales
	for _, t := range trades {
		valueUSD := t.Price * t.Quantity
		if valueUSD >= 100000.0 {
			flow.whaleCount++
			if t.Side == "buy" {
				flow.whaleBuyUSD += valueUSD
			} else {
				flow.whaleSellUSD += valueUSD
			}
		}
	}

	totalWhale := flow.whaleBuyUSD + flow.whaleSellUSD
	if totalWhale > 0 {
		flow.whaleImbalancePct = ((flow.whaleBuyUSD - flow.whaleSellUSD) / totalWhale) * 100
	}

	flow.whaleActivity = "none"
	flow.whaleSignal = "neutral"
	if flow.whaleCount > 0 {
		flow.whaleActivity = "present"
		if flow.whaleImbalancePct > 30 {
			flow.whaleSignal = "whale_buying"
		} else if flow.whaleImbalancePct < -30 {
			flow.whaleSignal = "whale_selling"
		}
	}

	return flow
}

func analyzeVolume(trades []market_data.Trade) volumeData {
	var vol volumeData
	if len(trades) == 0 {
		return vol
	}

	for _, t := range trades {
		usd := t.Price * t.Quantity
		vol.totalUSD += usd
		if t.Side == "buy" {
			vol.buyUSD += usd
		} else {
			vol.sellUSD += usd
		}
	}

	if len(trades) > 1 {
		timeSpan := trades[0].Timestamp.Sub(trades[len(trades)-1].Timestamp)
		vol.periodMinutes = timeSpan.Minutes()
		if vol.periodMinutes > 0 {
			vol.hourlyEstimate = (vol.totalUSD / vol.periodMinutes) * 60
		}
	}

	return vol
}

func analyzeSpeed(trades []market_data.Trade) tickSpeedData {
	var tick tickSpeedData
	if len(trades) < 2 {
		return tick
	}

	timeSpan := trades[0].Timestamp.Sub(trades[len(trades)-1].Timestamp)
	if timeSpan.Seconds() == 0 {
		return tick
	}

	tick.tradesPerMin = float64(len(trades)) / timeSpan.Minutes()

	totalUSD := 0.0
	for _, t := range trades {
		totalUSD += t.Price * t.Quantity
	}
	tick.volumePerMin = totalUSD / timeSpan.Minutes()

	tick.activity = "low"
	if tick.tradesPerMin > 100 {
		tick.activity = "extreme"
	} else if tick.tradesPerMin > 50 {
		tick.activity = "high"
	} else if tick.tradesPerMin > 20 {
		tick.activity = "moderate"
	}

	tick.signal = "neutral"
	if tick.activity == "extreme" {
		tick.signal = "breakout_likely"
	}

	return tick
}

func computeSignal(flow tradeFlowData, ob orderbookData) signalData {
	var sig signalData

	signals := []string{flow.signal, flow.cvdSignal, flow.whaleSignal, ob.signal}
	for _, s := range signals {
		switch s {
		case "bullish", "bullish_reversal", "whale_buying", "breakout_likely":
			sig.bullish++
		case "bearish", "bearish_reversal", "whale_selling":
			sig.bearish++
		}
	}

	total := sig.bullish + sig.bearish
	sig.direction = "neutral"
	sig.confidence = 0.5

	if total > 0 {
		if sig.bullish > sig.bearish {
			sig.direction = "bullish"
			sig.confidence = float64(sig.bullish) / float64(total)
		} else if sig.bearish > sig.bullish {
			sig.direction = "bearish"
			sig.confidence = float64(sig.bearish) / float64(total)
		}
	}

	sig.strength = "weak"
	if sig.confidence > 0.7 {
		sig.strength = "strong"
	} else if sig.confidence > 0.5 {
		sig.strength = "moderate"
	}

	return sig
}

func formatWall(w *wallData) interface{} {
	if w == nil {
		return nil
	}
	return map[string]interface{}{
		"Price": w.price,
		"Size":  w.size,
		"Ratio": w.ratio,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
