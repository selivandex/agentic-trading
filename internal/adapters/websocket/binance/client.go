package binance

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/adshao/go-binance/v2/futures"

	"prometheus/internal/adapters/websocket"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Client implements websocket.Client for Binance Futures
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

	c.logger.Info("Connecting to Binance WebSocket streams",
		"exchange", c.exchange,
		"stream_count", len(config.Streams),
		"testnet", c.useTestnet,
	)

	// Group streams by type for efficient combined stream connections
	klineStreams := make(map[string][]string) // symbol -> []interval
	var markPriceSymbols []string
	var tickerSymbols []string
	var aggTradeSymbols []string
	var otherStreams []websocket.StreamConfig

	for _, stream := range config.Streams {
		switch stream.Type {
		case websocket.StreamTypeKline:
			interval := string(stream.Interval)
			if intervals, ok := klineStreams[stream.Symbol]; ok {
				klineStreams[stream.Symbol] = append(intervals, interval)
			} else {
				klineStreams[stream.Symbol] = []string{interval}
			}
		case websocket.StreamTypeMarkPrice:
			markPriceSymbols = append(markPriceSymbols, stream.Symbol)
		case websocket.StreamTypeTicker:
			tickerSymbols = append(tickerSymbols, stream.Symbol)
		case websocket.StreamTypeTrade:
			aggTradeSymbols = append(aggTradeSymbols, stream.Symbol)
		default:
			otherStreams = append(otherStreams, stream)
		}
	}

	// Connect kline streams using combined endpoint
	if len(klineStreams) > 0 {
		if err := c.connectKlineStreams(klineStreams); err != nil {
			return errors.Wrap(err, "failed to connect kline streams")
		}
	}

	// Connect mark price streams (funding rate + mark/index price)
	if len(markPriceSymbols) > 0 {
		if err := c.connectMarkPriceStreams(markPriceSymbols); err != nil {
			return errors.Wrap(err, "failed to connect mark price streams")
		}
	}

	// Connect ticker streams (24hr statistics)
	if len(tickerSymbols) > 0 {
		if err := c.connectTickerStreams(tickerSymbols); err != nil {
			return errors.Wrap(err, "failed to connect ticker streams")
		}
	}

	// Connect aggregated trade streams
	if len(aggTradeSymbols) > 0 {
		if err := c.connectAggTradeStreams(aggTradeSymbols); err != nil {
			return errors.Wrap(err, "failed to connect agg trade streams")
		}
	}

	// TODO: Connect other stream types (depth, liquidations)
	if len(otherStreams) > 0 {
		c.logger.Warn("Some stream types are not yet implemented",
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
	c.statsMu.Lock()
	c.stats.ConnectedSince = time.Now()
	c.stats.ActiveStreams = len(config.Streams)
	c.statsMu.Unlock()

	return nil
}

// connectKlineStreams connects to multiple kline streams efficiently
func (c *Client) connectKlineStreams(symbolIntervals map[string][]string) error {
	errHandler := func(err error) {
		c.errorCount.Add(1)
		c.statsMu.Lock()
		c.stats.LastError = err
		c.statsMu.Unlock()

		c.logger.Error("WebSocket error",
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
			c.logger.Debug("Ignoring event during shutdown",
				"symbol", event.Symbol,
			)
			return
		}

		if c.handler == nil {
			return
		}

		// Convert Binance event to our generic format
		genericEvent := c.convertKlineEvent(event)
		if err := c.handler.OnKline(genericEvent); err != nil {
			c.logger.Error("Failed to handle kline event",
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

	c.logger.Info("Connected to kline streams",
		"exchange", c.exchange,
		"symbols", len(symbolIntervals),
	)

	return nil
}

// convertKlineEvent converts Binance kline event to generic format
func (c *Client) convertKlineEvent(event *futures.WsKlineEvent) *websocket.KlineEvent {
	return &websocket.KlineEvent{
		Exchange:    c.exchange,
		Symbol:      event.Symbol,
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
		c.statsMu.Lock()
		c.stats.LastError = err
		c.statsMu.Unlock()

		c.logger.Error("Mark price WebSocket error",
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
			c.logger.Error("Failed to handle mark price event",
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

	c.logger.Info("Connected to mark price streams",
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

// connectTickerStreams connects to 24hr ticker streams
func (c *Client) connectTickerStreams(symbols []string) error {
	errHandler := func(err error) {
		c.errorCount.Add(1)
		c.statsMu.Lock()
		c.stats.LastError = err
		c.statsMu.Unlock()

		c.logger.Error("Ticker WebSocket error",
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
				c.logger.Error("Failed to handle ticker event",
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

	c.logger.Info("Connected to ticker streams",
		"exchange", c.exchange,
		"symbols", len(symbols),
	)

	return nil
}

// convertTickerEvent converts Binance ticker event to generic format
func (c *Client) convertTickerEvent(event *futures.WsMarketTickerEvent) *websocket.TickerEvent {
	return &websocket.TickerEvent{
		Exchange:           c.exchange,
		Symbol:             event.Symbol,
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

// connectAggTradeStreams connects to aggregated trade streams
func (c *Client) connectAggTradeStreams(symbols []string) error {
	errHandler := func(err error) {
		c.errorCount.Add(1)
		c.statsMu.Lock()
		c.stats.LastError = err
		c.statsMu.Unlock()

		c.logger.Error("AggTrade WebSocket error",
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
			c.logger.Error("Failed to handle agg trade event",
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

	c.logger.Info("Connected to agg trade streams",
		"exchange", c.exchange,
		"symbols", len(symbols),
	)

	return nil
}

// convertAggTradeEvent converts Binance agg trade event to generic format
func (c *Client) convertAggTradeEvent(event *futures.WsAggTradeEvent) *websocket.TradeEvent {
	return &websocket.TradeEvent{
		Exchange:      c.exchange,
		Symbol:        event.Symbol,
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

// Start begins receiving events (already started in Connect)
func (c *Client) Start(ctx context.Context) error {
	if !c.connected.Load() {
		return errors.New("client not connected")
	}

	c.logger.Info("WebSocket client started",
		"exchange", c.exchange,
	)

	// Monitor context cancellation
	go func() {
		<-ctx.Done()
		if err := c.Stop(context.Background()); err != nil {
			c.logger.Error("Error stopping WebSocket client",
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

	c.logger.Info("Stopping WebSocket client",
		"exchange", c.exchange,
		"active_streams", len(c.stopChannels),
	)

	// Signal all streams to stop (send signal, don't close channel)
	for i, stopC := range c.stopChannels {
		select {
		case stopC <- struct{}{}:
			c.logger.Debug("Sent stop signal to stream", "stream", i)
		default:
			c.logger.Debug("Stop channel full, forcing close", "stream", i)
			close(stopC)
		}
	}

	// Wait for all streams to finish with timeout
	done := make(chan struct{})
	go func() {
		for i, doneC := range c.doneChannels {
			c.logger.Debug("Waiting for stream to finish", "stream", i)
			<-doneC
			c.logger.Debug("Stream finished", "stream", i)
		}
		close(done)
	}()

	timeout := 10 * time.Second
	select {
	case <-done:
		c.logger.Info("All WebSocket streams stopped gracefully",
			"exchange", c.exchange,
		)
	case <-time.After(timeout):
		c.logger.Error("WebSocket stop timeout, some streams may still be running",
			"exchange", c.exchange,
			"timeout", timeout,
		)
		return errors.New("websocket stop timeout")
	}

	c.connected.Store(false)
	c.stopChannels = nil
	c.doneChannels = nil

	c.logger.Info("✓ WebSocket client fully stopped",
		"exchange", c.exchange,
	)

	return nil
}

// IsConnected returns connection status
func (c *Client) IsConnected() bool {
	return c.connected.Load()
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
