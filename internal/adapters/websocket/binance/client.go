package binance

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"

	"prometheus/internal/adapters/websocket"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Client implements websocket.Client for Binance (Spot + Futures)
type Client struct {
	exchange   string
	handler    websocket.EventHandler
	config     websocket.ConnectionConfig
	useTestnet bool

	// Authentication (optional - only for private streams)
	apiKey    string
	secretKey string

	// Connection management
	mu           sync.RWMutex
	connected    atomic.Bool
	stopping     atomic.Bool // Flag to prevent publishing during shutdown
	stopChannels []chan struct{}
	doneChannels []chan struct{}

	// Statistics
	stats            websocket.Stats
	statsMu          sync.RWMutex
	messagesReceived atomic.Int64
	messagesSent     atomic.Int64
	errorCount       atomic.Int64
	reconnectCount   atomic.Int32
	recentErrors     atomic.Int32 // Track recent errors for health check

	logger *logger.Logger
}

// NewClient creates a new Binance WebSocket client
// apiKey and secretKey are optional - only needed for private streams (user data, orders)
func NewClient(exchange string, handler websocket.EventHandler, apiKey, secretKey string, useTestnet bool, log *logger.Logger) *Client {
	return &Client{
		exchange:   exchange,
		handler:    handler,
		apiKey:     apiKey,
		secretKey:  secretKey,
		useTestnet: useTestnet,
		logger:     log,
	}
}

// Connect establishes WebSocket connections based on configuration
func (c *Client) Connect(ctx context.Context, config websocket.ConnectionConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected.Load() {
		return errors.New("client already connected")
	}

	c.config = config

	// Set testnet flag for the library
	futures.UseTestnet = c.useTestnet

	c.logger.Infow("Connecting to Binance WebSocket streams",
		"exchange", c.exchange,
		"stream_count", len(config.Streams),
		"testnet", c.useTestnet,
	)

	// Group streams by market type and stream type
	spotKlineStreams := make(map[string][]string)    // symbol -> []interval
	futuresKlineStreams := make(map[string][]string) // symbol -> []interval
	var spotTickerSymbols []string
	var futuresTickerSymbols []string
	var spotTradeSymbols []string
	var futuresTradeSymbols []string
	var spotDepthSymbols []string
	var futuresDepthSymbols []string
	var markPriceSymbols []string   // Only futures have mark price
	var liquidationSymbols []string // Only futures have liquidations
	var otherStreams []websocket.StreamConfig

	for _, stream := range config.Streams {
		isSpot := stream.MarketType == websocket.MarketTypeSpot
		isFutures := stream.MarketType == websocket.MarketTypeFutures

		switch stream.Type {
		case websocket.StreamTypeKline:
			interval := string(stream.Interval)
			if isSpot {
				if intervals, ok := spotKlineStreams[stream.Symbol]; ok {
					spotKlineStreams[stream.Symbol] = append(intervals, interval)
				} else {
					spotKlineStreams[stream.Symbol] = []string{interval}
				}
			} else if isFutures {
				if intervals, ok := futuresKlineStreams[stream.Symbol]; ok {
					futuresKlineStreams[stream.Symbol] = append(intervals, interval)
				} else {
					futuresKlineStreams[stream.Symbol] = []string{interval}
				}
			}

		case websocket.StreamTypeMarkPrice:
			// Mark price only exists for futures
			if isFutures {
				markPriceSymbols = append(markPriceSymbols, stream.Symbol)
			} else {
				c.logger.Warnw("Mark price stream requested for spot market (not supported)",
					"symbol", stream.Symbol,
				)
			}

		case websocket.StreamTypeTicker:
			if isSpot {
				spotTickerSymbols = append(spotTickerSymbols, stream.Symbol)
			} else if isFutures {
				futuresTickerSymbols = append(futuresTickerSymbols, stream.Symbol)
			}

		case websocket.StreamTypeTrade:
			if isSpot {
				spotTradeSymbols = append(spotTradeSymbols, stream.Symbol)
			} else if isFutures {
				futuresTradeSymbols = append(futuresTradeSymbols, stream.Symbol)
			}

		case websocket.StreamTypeDepth:
			if isSpot {
				spotDepthSymbols = append(spotDepthSymbols, stream.Symbol)
			} else if isFutures {
				futuresDepthSymbols = append(futuresDepthSymbols, stream.Symbol)
			}

		case websocket.StreamTypeLiquidation:
			// Liquidations only exist for futures
			if isFutures {
				liquidationSymbols = append(liquidationSymbols, stream.Symbol)
			} else {
				c.logger.Warnw("Liquidation stream requested for spot market (not supported)",
					"symbol", stream.Symbol,
				)
			}

		default:
			otherStreams = append(otherStreams, stream)
		}
	}

	// Connect SPOT streams
	if len(spotKlineStreams) > 0 {
		if err := c.connectSpotKlineStreams(spotKlineStreams); err != nil {
			return errors.Wrap(err, "failed to connect spot kline streams")
		}
	}

	if len(spotTickerSymbols) > 0 {
		if err := c.connectSpotTickerStreams(spotTickerSymbols); err != nil {
			return errors.Wrap(err, "failed to connect spot ticker streams")
		}
	}

	if len(spotTradeSymbols) > 0 {
		if err := c.connectSpotTradeStreams(spotTradeSymbols); err != nil {
			return errors.Wrap(err, "failed to connect spot trade streams")
		}
	}

	if len(spotDepthSymbols) > 0 {
		if err := c.connectSpotDepthStreams(spotDepthSymbols); err != nil {
			return errors.Wrap(err, "failed to connect spot depth streams")
		}
	}

	// Connect FUTURES streams
	if len(futuresKlineStreams) > 0 {
		if err := c.connectFuturesKlineStreams(futuresKlineStreams); err != nil {
			return errors.Wrap(err, "failed to connect futures kline streams")
		}
	}

	if len(futuresTickerSymbols) > 0 {
		if err := c.connectFuturesTickerStreams(futuresTickerSymbols); err != nil {
			return errors.Wrap(err, "failed to connect futures ticker streams")
		}
	}

	if len(futuresTradeSymbols) > 0 {
		if err := c.connectFuturesTradeStreams(futuresTradeSymbols); err != nil {
			return errors.Wrap(err, "failed to connect futures trade streams")
		}
	}

	if len(futuresDepthSymbols) > 0 {
		if err := c.connectFuturesDepthStreams(futuresDepthSymbols); err != nil {
			return errors.Wrap(err, "failed to connect futures depth streams")
		}
	}

	// Connect mark price streams (futures only)
	if len(markPriceSymbols) > 0 {
		if err := c.connectMarkPriceStreams(markPriceSymbols); err != nil {
			return errors.Wrap(err, "failed to connect mark price streams")
		}
	}

	// Connect liquidation streams (futures only)
	if len(liquidationSymbols) > 0 {
		if err := c.connectLiquidationStreams(liquidationSymbols); err != nil {
			return errors.Wrap(err, "failed to connect liquidation streams")
		}
	}

	// Handle other unimplemented stream types
	if len(otherStreams) > 0 {
		c.logger.Warnw("Some stream types are not yet implemented",
			"count", len(otherStreams),
			"types", func() []string {
				types := make(map[websocket.StreamType]bool)
				for _, s := range otherStreams {
					types[s.Type] = true
				}
				result := make([]string, 0, len(types))
				for t := range types {
					result = append(result, string(t))
				}
				return result
			}(),
		)
	}

	c.connected.Store(true)
	c.recentErrors.Store(0) // Reset error counter on successful connection
	c.statsMu.Lock()
	c.stats.ConnectedSince = time.Now()
	c.stats.ActiveStreams = len(config.Streams)
	c.statsMu.Unlock()

	return nil
}

// connectFuturesKlineStreams connects to multiple futures kline streams efficiently
func (c *Client) connectFuturesKlineStreams(symbolIntervals map[string][]string) error {
	errHandler := func(err error) {
		c.errorCount.Add(1)
		c.recentErrors.Add(1)
		c.statsMu.Lock()
		c.stats.LastError = err
		c.statsMu.Unlock()

		c.logger.Errorw("WebSocket error",
			"exchange", c.exchange,
			"error", err.Error(),
		)

		if c.handler != nil {
			c.handler.OnError(err)
		}
	}

	klineHandler := func(event *futures.WsKlineEvent) {
		c.messagesReceived.Add(1)

		// Check if we're shutting down - don't process events during shutdown
		if c.stopping.Load() {
			c.logger.Debugw("Ignoring event during shutdown",
				"symbol", event.Symbol,
			)
			return
		}

		if c.handler == nil {
			return
		}

		// Reset recent errors counter on successful message (connection is healthy)
		if c.recentErrors.Load() > 0 {
			c.recentErrors.Store(0)
		}

		// Convert Binance event to our generic format
		genericEvent := c.convertKlineEvent(event)
		if err := c.handler.OnKline(genericEvent); err != nil {
			c.logger.Errorw("Failed to handle kline event",
				"exchange", c.exchange,
				"symbol", event.Symbol,
				"error", err.Error(),
			)
			c.errorCount.Add(1)
		}
	}

	// Use combined multi-interval stream for efficiency
	doneC, stopC, err := futures.WsCombinedKlineServeMultiInterval(
		symbolIntervals,
		klineHandler,
		errHandler,
	)
	if err != nil {
		return errors.Wrap(err, "failed to start kline WebSocket")
	}

	c.stopChannels = append(c.stopChannels, stopC)
	c.doneChannels = append(c.doneChannels, doneC)

	c.logger.Infow("Connected to futures kline streams",
		"exchange", c.exchange,
		"market_type", "futures",
		"symbols", len(symbolIntervals),
	)

	return nil
}

// connectSpotKlineStreams connects to multiple spot kline streams
func (c *Client) connectSpotKlineStreams(symbolIntervals map[string][]string) error {
	errHandler := func(err error) {
		c.errorCount.Add(1)
		c.recentErrors.Add(1)
		c.statsMu.Lock()
		c.stats.LastError = err
		c.statsMu.Unlock()

		c.logger.Errorw("Spot kline WebSocket error",
			"exchange", c.exchange,
			"error", err.Error(),
		)

		if c.handler != nil {
			c.handler.OnError(err)
		}
	}

	klineHandler := func(event *binance.WsKlineEvent) {
		c.messagesReceived.Add(1)

		if c.stopping.Load() {
			c.logger.Debugw("Ignoring spot kline event during shutdown",
				"symbol", event.Symbol,
			)
			return
		}

		if c.handler == nil {
			return
		}

		// Convert to generic format
		genericEvent := c.convertSpotKlineEvent(event)
		if err := c.handler.OnKline(genericEvent); err != nil {
			c.logger.Errorw("Failed to handle spot kline event",
				"exchange", c.exchange,
				"symbol", event.Symbol,
				"error", err.Error(),
			)
			c.errorCount.Add(1)
		}
	}

	// Use combined multi-interval stream for efficiency
	doneC, stopC, err := binance.WsCombinedKlineServeMultiInterval(
		symbolIntervals,
		klineHandler,
		errHandler,
	)
	if err != nil {
		return errors.Wrap(err, "failed to start spot kline WebSocket")
	}

	c.stopChannels = append(c.stopChannels, stopC)
	c.doneChannels = append(c.doneChannels, doneC)

	c.logger.Infow("Connected to spot kline streams",
		"exchange", c.exchange,
		"market_type", "spot",
		"symbols", len(symbolIntervals),
	)

	return nil
}

// convertKlineEvent converts Binance futures kline event to generic format
func (c *Client) convertKlineEvent(event *futures.WsKlineEvent) *websocket.KlineEvent {
	return &websocket.KlineEvent{
		Exchange:    c.exchange,
		Symbol:      event.Symbol,
		MarketType:  "futures",
		Interval:    event.Kline.Interval,
		OpenTime:    time.UnixMilli(event.Kline.StartTime),
		CloseTime:   time.UnixMilli(event.Kline.EndTime),
		Open:        event.Kline.Open,
		High:        event.Kline.High,
		Low:         event.Kline.Low,
		Close:       event.Kline.Close,
		Volume:      event.Kline.Volume,
		QuoteVolume: event.Kline.QuoteVolume,
		TradeCount:  event.Kline.TradeNum,
		IsFinal:     event.Kline.IsFinal,
		EventTime:   time.UnixMilli(event.Time),
	}
}

// convertSpotKlineEvent converts Binance spot kline event to generic format
func (c *Client) convertSpotKlineEvent(event *binance.WsKlineEvent) *websocket.KlineEvent {
	return &websocket.KlineEvent{
		Exchange:    c.exchange,
		Symbol:      event.Symbol,
		MarketType:  "spot",
		Interval:    event.Kline.Interval,
		OpenTime:    time.UnixMilli(event.Kline.StartTime),
		CloseTime:   time.UnixMilli(event.Kline.EndTime),
		Open:        event.Kline.Open,
		High:        event.Kline.High,
		Low:         event.Kline.Low,
		Close:       event.Kline.Close,
		Volume:      event.Kline.Volume,
		QuoteVolume: event.Kline.QuoteVolume,
		TradeCount:  event.Kline.TradeNum,
		IsFinal:     event.Kline.IsFinal,
		EventTime:   time.UnixMilli(event.Time),
	}
}

// connectMarkPriceStreams connects to mark price streams (funding rate + mark/index price)
func (c *Client) connectMarkPriceStreams(symbols []string) error {
	errHandler := func(err error) {
		c.errorCount.Add(1)
		c.recentErrors.Add(1)
		c.statsMu.Lock()
		c.stats.LastError = err
		c.statsMu.Unlock()

		c.logger.Errorw("Mark price WebSocket error",
			"exchange", c.exchange,
			"error", err.Error(),
		)

		if c.handler != nil {
			c.handler.OnError(err)
		}
	}

	markPriceHandler := func(event *futures.WsMarkPriceEvent) {
		c.messagesReceived.Add(1)

		if c.stopping.Load() {
			return
		}

		if c.handler == nil {
			return
		}

		// Convert to generic format
		genericEvent := c.convertMarkPriceEvent(event)
		if err := c.handler.OnMarkPrice(genericEvent); err != nil {
			c.logger.Errorw("Failed to handle mark price event",
				"exchange", c.exchange,
				"symbol", event.Symbol,
				"error", err.Error(),
			)
			c.errorCount.Add(1)
		}
	}

	// Use combined stream for multiple symbols
	doneC, stopC, err := futures.WsCombinedMarkPriceServe(
		symbols,
		markPriceHandler,
		errHandler,
	)
	if err != nil {
		return errors.Wrap(err, "failed to start mark price WebSocket")
	}

	c.stopChannels = append(c.stopChannels, stopC)
	c.doneChannels = append(c.doneChannels, doneC)

	c.logger.Infow("Connected to mark price streams",
		"exchange", c.exchange,
		"symbols", len(symbols),
	)

	return nil
}

// convertMarkPriceEvent converts Binance mark price event to generic format
func (c *Client) convertMarkPriceEvent(event *futures.WsMarkPriceEvent) *websocket.MarkPriceEvent {
	return &websocket.MarkPriceEvent{
		Exchange:             c.exchange,
		Symbol:               event.Symbol,
		MarkPrice:            event.MarkPrice,
		IndexPrice:           event.IndexPrice,
		EstimatedSettlePrice: event.EstimatedSettlePrice,
		FundingRate:          event.FundingRate,
		NextFundingTime:      time.UnixMilli(event.NextFundingTime),
		EventTime:            time.UnixMilli(event.Time),
	}
}

// connectLiquidationStreams connects to liquidation order streams (futures only)
// Uses !forceOrder@arr stream to get all liquidations across all symbols
func (c *Client) connectLiquidationStreams(symbols []string) error {
	errHandler := func(err error) {
		c.errorCount.Add(1)
		c.recentErrors.Add(1)
		c.statsMu.Lock()
		c.stats.LastError = err
		c.statsMu.Unlock()

		c.logger.Errorw("Liquidation WebSocket error",
			"exchange", c.exchange,
			"error", err.Error(),
		)

		if c.handler != nil {
			c.handler.OnError(err)
		}
	}

	liquidationHandler := func(event *futures.WsLiquidationOrderEvent) {
		c.messagesReceived.Add(1)

		if c.stopping.Load() {
			return
		}

		if c.handler == nil {
			return
		}

		// Convert to generic format
		genericEvent := c.convertLiquidationEvent(event)
		if err := c.handler.OnLiquidation(genericEvent); err != nil {
			c.logger.Errorw("Failed to handle liquidation event",
				"exchange", c.exchange,
				"symbol", event.LiquidationOrder.Symbol,
				"error", err.Error(),
			)
			c.errorCount.Add(1)
		}
	}

	// Use all market liquidation stream (!forceOrder@arr)
	// This covers all symbols, so we don't need individual subscriptions
	doneC, stopC, err := futures.WsAllLiquidationOrderServe(
		liquidationHandler,
		errHandler,
	)
	if err != nil {
		return errors.Wrap(err, "failed to start liquidation WebSocket")
	}

	c.stopChannels = append(c.stopChannels, stopC)
	c.doneChannels = append(c.doneChannels, doneC)

	c.logger.Infow("Connected to liquidation stream (all markets)",
		"exchange", c.exchange,
		"symbols", len(symbols),
	)

	return nil
}

// convertLiquidationEvent converts Binance liquidation event to generic format
func (c *Client) convertLiquidationEvent(event *futures.WsLiquidationOrderEvent) *websocket.LiquidationEvent {
	liq := event.LiquidationOrder

	// Calculate value (price * quantity)
	price, _ := strconv.ParseFloat(liq.Price, 64)
	qty, _ := strconv.ParseFloat(liq.OrigQuantity, 64)
	value := price * qty

	// Convert side to generic format
	side := "long"
	if liq.Side == futures.SideTypeSell {
		side = "short"
	}

	return &websocket.LiquidationEvent{
		Exchange:   c.exchange,
		Symbol:     liq.Symbol,
		MarketType: "futures",
		Side:       side,
		OrderType:  string(liq.OrderType),
		Price:      liq.Price,
		Quantity:   liq.OrigQuantity,
		Value:      strconv.FormatFloat(value, 'f', -1, 64),
		EventTime:  time.UnixMilli(event.Time),
	}
}

// connectFuturesTickerStreams connects to futures 24hr ticker streams
func (c *Client) connectFuturesTickerStreams(symbols []string) error {
	errHandler := func(err error) {
		c.errorCount.Add(1)
		c.recentErrors.Add(1)
		c.statsMu.Lock()
		c.stats.LastError = err
		c.statsMu.Unlock()

		c.logger.Errorw("Ticker WebSocket error",
			"exchange", c.exchange,
			"error", err.Error(),
		)

		if c.handler != nil {
			c.handler.OnError(err)
		}
	}

	tickerHandler := func(events futures.WsAllMarketTickerEvent) {
		c.messagesReceived.Add(int64(len(events)))

		if c.stopping.Load() {
			return
		}

		if c.handler == nil {
			return
		}

		// Process each ticker event in the array
		for _, event := range events {
			// Convert to generic format
			genericEvent := c.convertTickerEvent(event)
			if err := c.handler.OnTicker(genericEvent); err != nil {
				c.logger.Errorw("Failed to handle ticker event",
					"exchange", c.exchange,
					"symbol", event.Symbol,
					"error", err.Error(),
				)
				c.errorCount.Add(1)
			}
		}
	}

	// Note: Using WsAllMarketTickerServe for all symbols is more efficient
	// than individual streams, but less flexible for filtering
	// For now, use all tickers approach
	doneC, stopC, err := futures.WsAllMarketTickerServe(
		tickerHandler,
		errHandler,
	)
	if err != nil {
		return errors.Wrap(err, "failed to start ticker WebSocket")
	}

	c.stopChannels = append(c.stopChannels, stopC)
	c.doneChannels = append(c.doneChannels, doneC)

	c.logger.Infow("Connected to futures ticker streams",
		"exchange", c.exchange,
		"market_type", "futures",
		"symbols", len(symbols),
	)

	return nil
}

// connectSpotTickerStreams connects to spot 24hr ticker streams
func (c *Client) connectSpotTickerStreams(symbols []string) error {
	errHandler := func(err error) {
		c.errorCount.Add(1)
		c.recentErrors.Add(1)
		c.statsMu.Lock()
		c.stats.LastError = err
		c.statsMu.Unlock()

		c.logger.Errorw("Spot ticker WebSocket error",
			"exchange", c.exchange,
			"error", err.Error(),
		)

		if c.handler != nil {
			c.handler.OnError(err)
		}
	}

	tickerHandler := binance.WsAllMarketsStatHandler(func(event binance.WsAllMarketsStatEvent) {
		c.messagesReceived.Add(int64(len(event)))

		if c.stopping.Load() {
			return
		}

		if c.handler == nil {
			return
		}

		// Process each ticker event in the array
		for _, tickerEvent := range event {
			genericEvent := c.convertSpotTickerEvent(*tickerEvent)
			if err := c.handler.OnTicker(genericEvent); err != nil {
				c.logger.Errorw("Failed to handle spot ticker event",
					"exchange", c.exchange,
					"symbol", tickerEvent.Symbol,
					"error", err.Error(),
				)
				c.errorCount.Add(1)
			}
		}
	})

	// Use all market tickers stream for efficiency
	doneC, stopC, err := binance.WsAllMarketsStatServe(tickerHandler, errHandler)
	if err != nil {
		return errors.Wrap(err, "failed to start spot ticker WebSocket")
	}

	c.stopChannels = append(c.stopChannels, stopC)
	c.doneChannels = append(c.doneChannels, doneC)

	c.logger.Infow("Connected to spot ticker streams",
		"exchange", c.exchange,
		"market_type", "spot",
		"symbols", len(symbols),
	)

	return nil
}

// convertTickerEvent converts Binance futures ticker event to generic format
func (c *Client) convertTickerEvent(event *futures.WsMarketTickerEvent) *websocket.TickerEvent {
	return &websocket.TickerEvent{
		Exchange:           c.exchange,
		Symbol:             event.Symbol,
		MarketType:         "futures",
		PriceChange:        event.PriceChange,
		PriceChangePercent: event.PriceChangePercent,
		WeightedAvgPrice:   event.WeightedAvgPrice,
		LastPrice:          event.ClosePrice,
		LastQty:            event.CloseQty,
		OpenPrice:          event.OpenPrice,
		HighPrice:          event.HighPrice,
		LowPrice:           event.LowPrice,
		Volume:             event.BaseVolume,
		QuoteVolume:        event.QuoteVolume,
		OpenTime:           time.UnixMilli(event.OpenTime),
		CloseTime:          time.UnixMilli(event.CloseTime),
		FirstTradeID:       event.FirstID,
		LastTradeID:        event.LastID,
		TradeCount:         event.TradeCount,
		EventTime:          time.UnixMilli(event.Time),
	}
}

// convertSpotTickerEvent converts Binance spot ticker event to generic format
func (c *Client) convertSpotTickerEvent(event binance.WsMarketStatEvent) *websocket.TickerEvent {
	return &websocket.TickerEvent{
		Exchange:           c.exchange,
		Symbol:             event.Symbol,
		MarketType:         "spot",
		PriceChange:        event.PriceChange,
		PriceChangePercent: event.PriceChangePercent,
		WeightedAvgPrice:   event.WeightedAvgPrice,
		LastPrice:          event.LastPrice,
		LastQty:            "0", // Field not available in WsMarketStatEvent
		OpenPrice:          event.OpenPrice,
		HighPrice:          event.HighPrice,
		LowPrice:           event.LowPrice,
		Volume:             event.BaseVolume,
		QuoteVolume:        event.QuoteVolume,
		OpenTime:           time.UnixMilli(event.OpenTime),
		CloseTime:          time.UnixMilli(event.CloseTime),
		FirstTradeID:       event.FirstID,
		LastTradeID:        event.LastID,
		TradeCount:         event.Count,
		EventTime:          time.Now(), // EventTime doesn't exist in spot events
	}
}

// connectFuturesTradeStreams connects to futures aggregated trade streams
func (c *Client) connectFuturesTradeStreams(symbols []string) error {
	errHandler := func(err error) {
		c.errorCount.Add(1)
		c.recentErrors.Add(1)
		c.statsMu.Lock()
		c.stats.LastError = err
		c.statsMu.Unlock()

		c.logger.Errorw("AggTrade WebSocket error",
			"exchange", c.exchange,
			"error", err.Error(),
		)

		if c.handler != nil {
			c.handler.OnError(err)
		}
	}

	aggTradeHandler := func(event *futures.WsAggTradeEvent) {
		c.messagesReceived.Add(1)

		if c.stopping.Load() {
			return
		}

		if c.handler == nil {
			return
		}

		// Convert to generic format
		genericEvent := c.convertAggTradeEvent(event)
		if err := c.handler.OnTrade(genericEvent); err != nil {
			c.logger.Errorw("Failed to handle agg trade event",
				"exchange", c.exchange,
				"symbol", event.Symbol,
				"error", err.Error(),
			)
			c.errorCount.Add(1)
		}
	}

	// Use combined stream for multiple symbols
	doneC, stopC, err := futures.WsCombinedAggTradeServe(
		symbols,
		aggTradeHandler,
		errHandler,
	)
	if err != nil {
		return errors.Wrap(err, "failed to start agg trade WebSocket")
	}

	c.stopChannels = append(c.stopChannels, stopC)
	c.doneChannels = append(c.doneChannels, doneC)

	c.logger.Infow("Connected to futures trade streams",
		"exchange", c.exchange,
		"market_type", "futures",
		"symbols", len(symbols),
	)

	return nil
}

// connectSpotTradeStreams connects to spot aggregated trade streams
func (c *Client) connectSpotTradeStreams(symbols []string) error {
	errHandler := func(err error) {
		c.errorCount.Add(1)
		c.recentErrors.Add(1)
		c.statsMu.Lock()
		c.stats.LastError = err
		c.statsMu.Unlock()

		c.logger.Errorw("Spot trade WebSocket error",
			"exchange", c.exchange,
			"error", err.Error(),
		)

		if c.handler != nil {
			c.handler.OnError(err)
		}
	}

	tradeHandler := func(event *binance.WsAggTradeEvent) {
		c.messagesReceived.Add(1)

		if c.stopping.Load() {
			return
		}

		if c.handler == nil {
			return
		}

		// Convert to generic format
		genericEvent := c.convertSpotTradeEvent(event)
		if err := c.handler.OnTrade(genericEvent); err != nil {
			c.logger.Errorw("Failed to handle spot trade event",
				"exchange", c.exchange,
				"symbol", event.Symbol,
				"error", err.Error(),
			)
			c.errorCount.Add(1)
		}
	}

	// Use combined stream for multiple symbols
	doneC, stopC, err := binance.WsCombinedAggTradeServe(
		symbols,
		tradeHandler,
		errHandler,
	)
	if err != nil {
		return errors.Wrap(err, "failed to start spot trade WebSocket")
	}

	c.stopChannels = append(c.stopChannels, stopC)
	c.doneChannels = append(c.doneChannels, doneC)

	c.logger.Infow("Connected to spot trade streams",
		"exchange", c.exchange,
		"market_type", "spot",
		"symbols", len(symbols),
	)

	return nil
}

// convertAggTradeEvent converts Binance futures agg trade event to generic format
func (c *Client) convertAggTradeEvent(event *futures.WsAggTradeEvent) *websocket.TradeEvent {
	return &websocket.TradeEvent{
		Exchange:      c.exchange,
		Symbol:        event.Symbol,
		MarketType:    "futures",
		TradeID:       event.AggregateTradeID,
		Price:         event.Price,
		Quantity:      event.Quantity,
		BuyerOrderID:  0, // Not available in agg trade
		SellerOrderID: 0, // Not available in agg trade
		TradeTime:     time.UnixMilli(event.TradeTime),
		IsBuyerMaker:  event.Maker,
		EventTime:     time.UnixMilli(event.Time),
	}
}

// convertSpotTradeEvent converts Binance spot agg trade event to generic format
func (c *Client) convertSpotTradeEvent(event *binance.WsAggTradeEvent) *websocket.TradeEvent {
	return &websocket.TradeEvent{
		Exchange:      c.exchange,
		Symbol:        event.Symbol,
		MarketType:    "spot",
		TradeID:       event.AggTradeID,
		Price:         event.Price,
		Quantity:      event.Quantity,
		BuyerOrderID:  0, // Not available in spot agg trade events
		SellerOrderID: 0, // Not available in spot agg trade events
		TradeTime:     time.UnixMilli(event.TradeTime),
		IsBuyerMaker:  event.IsBuyerMaker,
		EventTime:     time.Now(), // EventTime doesn't exist in spot events
	}
}

// connectFuturesDepthStreams connects to futures depth (order book) streams
func (c *Client) connectFuturesDepthStreams(symbols []string) error {
	errHandler := func(err error) {
		c.errorCount.Add(1)
		c.recentErrors.Add(1)
		c.statsMu.Lock()
		c.stats.LastError = err
		c.statsMu.Unlock()

		c.logger.Errorw("Depth WebSocket error",
			"exchange", c.exchange,
			"error", err.Error(),
		)

		if c.handler != nil {
			c.handler.OnError(err)
		}
	}

	depthHandler := func(event *futures.WsDepthEvent) {
		c.messagesReceived.Add(1)

		if c.stopping.Load() {
			return
		}

		if c.handler == nil {
			return
		}

		// Convert to generic format
		genericEvent := c.convertFuturesDepthEvent(event)
		if err := c.handler.OnDepth(genericEvent); err != nil {
			c.logger.Errorw("Failed to handle depth event",
				"exchange", c.exchange,
				"symbol", event.Symbol,
				"error", err.Error(),
			)
			c.errorCount.Add(1)
		}
	}

	// Build symbol->level map for combined stream
	symbolLevels := make(map[string]string, len(symbols))
	for _, symbol := range symbols {
		symbolLevels[symbol] = "10" // Depth levels: 5, 10, or 20
	}

	// Use combined stream for multiple symbols
	doneC, stopC, err := futures.WsCombinedDepthServe(
		symbolLevels,
		depthHandler,
		errHandler,
	)
	if err != nil {
		return errors.Wrap(err, "failed to start futures depth WebSocket")
	}

	c.stopChannels = append(c.stopChannels, stopC)
	c.doneChannels = append(c.doneChannels, doneC)

	c.logger.Infow("Connected to futures depth streams",
		"exchange", c.exchange,
		"market_type", "futures",
		"symbols", len(symbols),
		"update_speed", "100ms",
		"depth_levels", 10,
	)

	return nil
}

// connectSpotDepthStreams connects to spot depth (order book) streams
func (c *Client) connectSpotDepthStreams(symbols []string) error {
	errHandler := func(err error) {
		c.errorCount.Add(1)
		c.recentErrors.Add(1)
		c.statsMu.Lock()
		c.stats.LastError = err
		c.statsMu.Unlock()

		c.logger.Errorw("Spot depth WebSocket error",
			"exchange", c.exchange,
			"error", err.Error(),
		)

		if c.handler != nil {
			c.handler.OnError(err)
		}
	}

	depthHandler := func(event *binance.WsPartialDepthEvent) {
		c.messagesReceived.Add(1)

		if c.stopping.Load() {
			return
		}

		if c.handler == nil {
			return
		}

		// Convert to generic format
		genericEvent := c.convertSpotDepthEvent(event)
		if err := c.handler.OnDepth(genericEvent); err != nil {
			c.logger.Errorw("Failed to handle spot depth event",
				"exchange", c.exchange,
				"symbol", event.Symbol,
				"error", err.Error(),
			)
			c.errorCount.Add(1)
		}
	}

	// Build symbol->level map for combined stream
	symbolLevels := make(map[string]string, len(symbols))
	for _, symbol := range symbols {
		symbolLevels[symbol] = "10" // Depth levels: 5, 10, or 20
	}

	// Use combined stream for multiple symbols
	doneC, stopC, err := binance.WsCombinedPartialDepthServe(
		symbolLevels,
		depthHandler,
		errHandler,
	)
	if err != nil {
		return errors.Wrap(err, "failed to start spot depth WebSocket")
	}

	c.stopChannels = append(c.stopChannels, stopC)
	c.doneChannels = append(c.doneChannels, doneC)

	c.logger.Infow("Connected to spot depth streams",
		"exchange", c.exchange,
		"market_type", "spot",
		"symbols", len(symbols),
		"depth_levels", 10,
	)

	return nil
}

// convertFuturesDepthEvent converts Binance futures depth event to generic format
func (c *Client) convertFuturesDepthEvent(event *futures.WsDepthEvent) *websocket.DepthEvent {
	bids := make([]websocket.PriceLevel, len(event.Bids))
	for i, bid := range event.Bids {
		bids[i] = websocket.PriceLevel{
			Price:    bid.Price,
			Quantity: bid.Quantity,
		}
	}

	asks := make([]websocket.PriceLevel, len(event.Asks))
	for i, ask := range event.Asks {
		asks[i] = websocket.PriceLevel{
			Price:    ask.Price,
			Quantity: ask.Quantity,
		}
	}

	return &websocket.DepthEvent{
		Exchange:     c.exchange,
		Symbol:       event.Symbol,
		MarketType:   "futures",
		Bids:         bids,
		Asks:         asks,
		LastUpdateID: event.LastUpdateID,
		EventTime:    time.UnixMilli(event.Time),
	}
}

// convertSpotDepthEvent converts Binance spot depth event to generic format
func (c *Client) convertSpotDepthEvent(event *binance.WsPartialDepthEvent) *websocket.DepthEvent {
	bids := make([]websocket.PriceLevel, len(event.Bids))
	for i, bid := range event.Bids {
		bids[i] = websocket.PriceLevel{
			Price:    bid.Price,
			Quantity: bid.Quantity,
		}
	}

	asks := make([]websocket.PriceLevel, len(event.Asks))
	for i, ask := range event.Asks {
		asks[i] = websocket.PriceLevel{
			Price:    ask.Price,
			Quantity: ask.Quantity,
		}
	}

	return &websocket.DepthEvent{
		Exchange:     c.exchange,
		Symbol:       event.Symbol,
		MarketType:   "spot",
		Bids:         bids,
		Asks:         asks,
		LastUpdateID: event.LastUpdateID,
		EventTime:    time.Now(), // Spot partial depth events don't have timestamp
	}
}

// Start begins receiving events (already started in Connect)
func (c *Client) Start(ctx context.Context) error {
	if !c.connected.Load() {
		return errors.New("client not connected")
	}

	c.logger.Infow("WebSocket client started",
		"exchange", c.exchange,
	)

	// Monitor context cancellation
	go func() {
		<-ctx.Done()
		if err := c.Stop(context.Background()); err != nil {
			c.logger.Errorw("Error stopping WebSocket client",
				"exchange", c.exchange,
				"error", err.Error(),
			)
		}
	}()

	return nil
}

// Stop gracefully closes all WebSocket connections
func (c *Client) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected.Load() {
		return nil
	}

	// Set stopping flag FIRST to prevent new events from being published
	c.stopping.Store(true)

	c.logger.Infow("Stopping WebSocket client",
		"exchange", c.exchange,
		"active_streams", len(c.stopChannels),
	)

	// Signal all streams to stop (send signal, don't close channel)
	for i, stopC := range c.stopChannels {
		select {
		case stopC <- struct{}{}:
			c.logger.Debugw("Sent stop signal to stream", "stream", i)
		default:
			c.logger.Debugw("Stop channel full, forcing close", "stream", i)
			close(stopC)
		}
	}

	// Wait for all streams to finish with timeout
	done := make(chan struct{})
	go func() {
		for i, doneC := range c.doneChannels {
			c.logger.Debugw("Waiting for stream to finish", "stream", i)
			<-doneC
			c.logger.Debugw("Stream finished", "stream", i)
		}
		close(done)
	}()

	// Reduced timeout: WebSocketPublisher.Shutdown() prevents further events from being published
	// so it's safe to force-stop streams that don't respond quickly
	timeout := 5 * time.Second
	select {
	case <-done:
		c.logger.Infow("All WebSocket streams stopped gracefully",
			"exchange", c.exchange,
		)
	case <-time.After(timeout):
		c.logger.Warnw("WebSocket stop timeout, forcing shutdown",
			"exchange", c.exchange,
			"timeout", timeout,
			"note", "WebSocketPublisher already stopped, so no new events will be published",
		)
		// Force close all stop channels to unblock any remaining goroutines
		for i, stopC := range c.stopChannels {
			select {
			case <-stopC:
				// Already closed
			default:
				c.logger.Debugw("Force closing stop channel", "stream", i)
				close(stopC)
			}
		}
	}

	// Always mark as disconnected, even on timeout
	c.connected.Store(false)
	c.stopChannels = nil
	c.doneChannels = nil

	c.logger.Infow("✓ WebSocket client stopped",
		"exchange", c.exchange,
	)

	return nil
}

// IsConnected returns connection status
// Returns false if too many recent errors occurred
func (c *Client) IsConnected() bool {
	if !c.connected.Load() {
		return false
	}

	// If we have more than 5 recent errors, consider connection unhealthy
	recentErrors := c.recentErrors.Load()
	if recentErrors > 5 {
		c.logger.Warnw("Connection considered unhealthy due to recent errors",
			"recent_errors", recentErrors,
		)
		return false
	}

	return true
}

// GetStats returns current statistics
func (c *Client) GetStats() websocket.Stats {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()

	stats := c.stats
	stats.MessagesReceived = c.messagesReceived.Load()
	stats.MessagesSent = c.messagesSent.Load()
	stats.ErrorCount = c.errorCount.Load()
	stats.ReconnectCount = int(c.reconnectCount.Load())

	return stats
}
