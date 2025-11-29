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

	// TODO: Connect other stream types
	if len(otherStreams) > 0 {
		c.logger.Warn("Some stream types are not yet implemented",
			"count", len(otherStreams),
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
