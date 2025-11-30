package bybit

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	bybitws "github.com/hirokisan/bybit/v2"

	"prometheus/internal/adapters/websocket"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Client implements websocket.Client for Bybit using hirokisan/bybit library
type Client struct {
	exchange string
	handler  websocket.EventHandler
	config   websocket.ConnectionConfig
	testnet  bool

	// Authentication (optional - only for private streams)
	apiKey    string
	secretKey string

	// Bybit WebSocket service
	wsClient *bybitws.WebSocketClient
	wsPublic bybitws.V5WebsocketPublicServiceI

	// Connection management
	mu        sync.RWMutex
	connected atomic.Bool
	stopping  atomic.Bool
	stopChan  chan struct{}
	doneChan  chan struct{}

	// Statistics
	stats            websocket.Stats
	statsMu          sync.RWMutex
	messagesReceived atomic.Int64
	messagesSent     atomic.Int64
	errorCount       atomic.Int64
	reconnectCount   atomic.Int32

	logger *logger.Logger
}

// NewClient creates a new Bybit WebSocket client using hirokisan/bybit
func NewClient(exchange string, handler websocket.EventHandler, apiKey, secretKey string, testnet bool, log *logger.Logger) *Client {
	return &Client{
		exchange:  exchange,
		handler:   handler,
		apiKey:    apiKey,
		secretKey: secretKey,
		testnet:   testnet,
		logger:    log,
		stopChan:  make(chan struct{}),
		doneChan:  make(chan struct{}),
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

	c.logger.Infow("Connecting to Bybit WebSocket streams",
		"exchange", c.exchange,
		"stream_count", len(config.Streams),
		"testnet", c.testnet,
	)

	// Create Bybit WebSocket client
	wsClient := bybitws.NewWebsocketClient()

	// Set baseURL for testnet if needed
	if c.testnet {
		wsClient = wsClient.WithBaseURL("wss://stream-testnet.bybit.com")
	}

	c.wsClient = wsClient

	// Group streams by type
	var markPriceSymbols []bybitws.SymbolV5

	for _, stream := range config.Streams {
		// Only support futures market for Bybit
		if stream.MarketType != websocket.MarketTypeFutures {
			c.logger.Warnw("Bybit WebSocket only supports futures market",
				"requested_market", stream.MarketType,
			)
			continue
		}

		switch stream.Type {
		case websocket.StreamTypeMarkPrice:
			markPriceSymbols = append(markPriceSymbols, bybitws.SymbolV5(stream.Symbol))
		default:
			c.logger.Warnw("Stream type not yet implemented for Bybit",
				"type", stream.Type,
			)
		}
	}

	// Subscribe to tickers (includes funding rate and mark price)
	if len(markPriceSymbols) > 0 {
		if err := c.subscribeToTickers(markPriceSymbols); err != nil {
			return errors.Wrap(err, "failed to subscribe to tickers")
		}
	}

	c.connected.Store(true)
	c.statsMu.Lock()
	c.stats.ConnectedSince = time.Now()
	c.stats.ActiveStreams = len(config.Streams)
	c.statsMu.Unlock()

	return nil
}

// subscribeToTickers subscribes to ticker streams (includes mark price and funding rate)
func (c *Client) subscribeToTickers(symbols []bybitws.SymbolV5) error {
	c.logger.Infow("Subscribing to Bybit ticker streams",
		"symbols", symbols,
	)

	// Create V5 Public service for Linear (futures) category
	svc, err := c.wsClient.V5().Public(bybitws.CategoryV5Linear)
	if err != nil {
		return errors.Wrap(err, "failed to create V5 public service")
	}

	c.wsPublic = svc

	// Subscribe to tickers for each symbol
	for _, symbol := range symbols {
		_, err := svc.SubscribeTicker(
			bybitws.V5WebsocketPublicTickerParamKey{
				Symbol: symbol,
			},
			c.handleTickerMessage,
		)
		if err != nil {
			return errors.Wrapf(err, "failed to subscribe to ticker for %s", symbol)
		}

		c.logger.Debugw("Subscribed to ticker",
			"symbol", symbol,
		)
	}

	return nil
}

// handleTickerMessage processes ticker messages from Bybit
func (c *Client) handleTickerMessage(response bybitws.V5WebsocketPublicTickerResponse) error {
	c.messagesReceived.Add(1)

	// Check if we're shutting down
	if c.stopping.Load() {
		c.logger.Debugw("Ignoring ticker during shutdown")
		return nil
	}

	if c.handler == nil {
		return nil
	}

	// Data is in LinearInverse field for futures
	if response.Data.LinearInverse == nil {
		c.logger.Warnw("Received ticker without LinearInverse data",
			"topic", response.Topic,
		)
		return nil
	}

	data := response.Data.LinearInverse
	symbol := string(data.Symbol)

	// Bybit sends two types of messages:
	// - "snapshot": full data (on connect and periodically)
	// - "delta": incremental updates (only changed fields)
	// We only process snapshot messages to keep implementation simple
	if response.Type != "snapshot" {
		return nil
	}

	// Convert to our generic MarkPriceEvent format
	event := &websocket.MarkPriceEvent{
		Exchange:        c.exchange,
		Symbol:          symbol,
		MarkPrice:       data.MarkPrice,
		IndexPrice:      data.IndexPrice,
		FundingRate:     data.FundingRate,
		NextFundingTime: parseBybitTimestamp(data.NextFundingTime),
		EventTime:       time.UnixMilli(response.TimeStamp),
	}

	// Send to handler
	if err := c.handler.OnMarkPrice(event); err != nil {
		c.logger.Errorw("Failed to handle mark price event",
			"symbol", data.Symbol,
			"error", err,
		)
		c.errorCount.Add(1)
		return err
	}

	return nil
}

// Start begins receiving events
func (c *Client) Start(ctx context.Context) error {
	if !c.connected.Load() {
		return errors.New("client not connected")
	}

	c.logger.Infow("Bybit WebSocket client started",
		"exchange", c.exchange,
	)

	// Define error handler for WebSocket connection issues
	errHandler := func(isWebsocketClosed bool, err error) {
		c.errorCount.Add(1)
		c.logger.Errorw("Bybit WebSocket error",
			"is_closed", isWebsocketClosed,
			"error", err,
		)

		if c.handler != nil {
			c.handler.OnError(err)
		}
	}

	// Start the WebSocket service in a goroutine
	go func() {
		defer close(c.doneChan)

		// Start() is blocking and handles reconnections
		if err := c.wsPublic.Start(ctx, errHandler); err != nil {
			c.logger.Errorw("WebSocket service start error",
				"error", err,
			)
			c.errorCount.Add(1)
		}
	}()

	// Monitor context cancellation
	go func() {
		<-ctx.Done()
		if err := c.Stop(context.Background()); err != nil {
			c.logger.Errorw("Error stopping Bybit WebSocket client",
				"error", err,
			)
		}
	}()

	return nil
}

// Stop gracefully closes the WebSocket connection
func (c *Client) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected.Load() {
		return nil
	}

	// Set stopping flag
	c.stopping.Store(true)

	c.logger.Infow("Stopping Bybit WebSocket client",
		"exchange", c.exchange,
	)

	// Close the WebSocket service
	if c.wsPublic != nil {
		if err := c.wsPublic.Close(); err != nil {
			c.logger.Errorw("Error closing WebSocket service",
				"error", err,
			)
		}
	}

	// Wait for Run() to finish with timeout
	timeout := 5 * time.Second
	select {
	case <-c.doneChan:
		c.logger.Info("Bybit WebSocket handler stopped gracefully")
	case <-time.After(timeout):
		c.logger.Warn("Bybit WebSocket handler stop timeout")
	}

	c.connected.Store(false)

	c.logger.Info("✓ Bybit WebSocket client stopped")

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

// parseBybitTimestamp converts Bybit timestamp string (ms) to time.Time
func parseBybitTimestamp(ts string) time.Time {
	if ts == "" {
		return time.Time{}
	}

	var msec int64
	fmt.Sscanf(ts, "%d", &msec)
	return time.UnixMilli(msec)
}
