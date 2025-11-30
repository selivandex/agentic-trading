package okx

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	okexapi "github.com/amir-the-h/okex"
	"github.com/amir-the-h/okex/api"
	"github.com/amir-the-h/okex/events/public"
	ws_public_requests "github.com/amir-the-h/okex/requests/ws/public"

	"prometheus/internal/adapters/websocket"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Client implements websocket.Client for OKX using amir-the-h/okex library
type Client struct {
	exchange string
	handler  websocket.EventHandler
	config   websocket.ConnectionConfig
	testnet  bool

	// Authentication (optional - only for private streams)
	apiKey     string
	secretKey  string
	passphrase string

	// OKX API client
	client *api.Client

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

// NewClient creates a new OKX WebSocket client using amir-the-h/okex library
func NewClient(exchange string, handler websocket.EventHandler, apiKey, secretKey string, testnet bool, log *logger.Logger) *Client {
	return &Client{
		exchange:   exchange,
		handler:    handler,
		apiKey:     apiKey,
		secretKey:  secretKey,
		passphrase: "", // Will be set from config if needed
		testnet:    testnet,
		logger:     log,
		stopChan:   make(chan struct{}),
		doneChan:   make(chan struct{}),
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

	c.logger.Infow("Connecting to OKX WebSocket streams",
		"exchange", c.exchange,
		"stream_count", len(config.Streams),
		"testnet", c.testnet,
	)

	// Determine server destination
	dest := okexapi.NormalServer
	if c.testnet {
		dest = okexapi.AwsServer // OKX doesn't have a clear testnet, using AWS as alternative
	}

	// Create OKX API client
	c.logger.Infow("Creating OKX API client",
		"has_api_key", c.apiKey != "",
		"has_secret", c.secretKey != "",
		"has_passphrase", c.passphrase != "",
		"dest", dest,
	)

	client, err := api.NewClient(ctx, c.apiKey, c.secretKey, c.passphrase, dest)
	if err != nil {
		return errors.Wrap(err, "failed to create OKX client")
	}

	c.client = client

	c.logger.Info("✓ OKX API client created successfully")

	// Group streams by type
	var fundingRateSymbols []string

	for _, stream := range config.Streams {
		// Only support futures market for OKX
		if stream.MarketType != websocket.MarketTypeFutures {
			c.logger.Warnw("OKX WebSocket only supports futures market",
				"requested_market", stream.MarketType,
			)
			continue
		}

		switch stream.Type {
		case websocket.StreamTypeMarkPrice:
			fundingRateSymbols = append(fundingRateSymbols, stream.Symbol)
		default:
			c.logger.Warnw("Stream type not yet implemented for OKX",
				"type", stream.Type,
			)
		}
	}

	// Subscribe to funding rates
	if len(fundingRateSymbols) > 0 {
		if err := c.subscribeToFundingRates(fundingRateSymbols); err != nil {
			return errors.Wrap(err, "failed to subscribe to funding rates")
		}
	}

	c.connected.Store(true)
	c.statsMu.Lock()
	c.stats.ConnectedSince = time.Now()
	c.stats.ActiveStreams = len(config.Streams)
	c.statsMu.Unlock()

	return nil
}

// subscribeToFundingRates subscribes to funding rate streams
func (c *Client) subscribeToFundingRates(symbols []string) error {
	c.logger.Infow("Subscribing to OKX funding rate streams",
		"symbols", symbols,
	)

	// Create channel for funding rate events
	fundingRateCh := make(chan *public.FundingRate)

	// Subscribe to funding rates for each symbol
	// OKX requires instrument format like BTC-USDT-SWAP
	for _, symbol := range symbols {
		instID := c.convertToOKXInstrument(symbol)

		req := ws_public_requests.FundingRate{
			InstID: instID,
		}

		// Subscribe to funding rate for this instrument
		if err := c.client.Ws.Public.FundingRate(req, fundingRateCh); err != nil {
			return errors.Wrapf(err, "failed to subscribe to funding rate for %s", instID)
		}

		c.logger.Debugw("Subscribed to funding rate",
			"symbol", symbol,
			"inst_id", instID,
		)
	}

	// Start goroutine to handle events
	go c.handleFundingRateEvents(fundingRateCh)

	return nil
}

// handleFundingRateEvents processes funding rate events from OKX
func (c *Client) handleFundingRateEvents(ch chan *public.FundingRate) {
	defer close(c.doneChan)

	c.logger.Info("OKX funding rate event handler started, waiting for events...")

	for {
		select {
		case <-c.stopChan:
			c.logger.Info("OKX funding rate event handler stopping")
			return
		case event, ok := <-ch:
			if !ok {
				c.logger.Warn("OKX funding rate channel closed")
				return
			}

			c.messagesReceived.Add(1)

			if c.stopping.Load() {
				continue
			}

			if c.handler == nil {
				continue
			}

			c.logger.Debugw("Received OKX funding rate event",
				"rates_count", len(event.Rates),
			)

			// Process each funding rate in the event
			for _, rate := range event.Rates {
				// Convert to standard symbol format
				symbol := c.convertFromOKXInstrument(rate.InstID)

				c.logger.Debugw("Processing OKX funding rate",
					"inst_id", rate.InstID,
					"symbol", symbol,
					"funding_rate", rate.FundingRate,
					"next_funding_time", rate.NextFundingTime,
				)

				// Create mark price event (OKX doesn't provide mark price in funding rate stream)
				// We'll need to get mark price separately or use funding rate only
				markPriceEvent := &websocket.MarkPriceEvent{
					Exchange:        c.exchange,
					Symbol:          symbol,
					MarkPrice:       "", // Not available in funding rate stream
					IndexPrice:      "", // Not available in funding rate stream
					FundingRate:     fmt.Sprintf("%f", float64(rate.FundingRate)),
					NextFundingTime: time.Time(rate.NextFundingTime),
					EventTime:       time.Now(),
				}

				if err := c.handler.OnMarkPrice(markPriceEvent); err != nil {
					c.logger.Errorw("Failed to handle mark price event",
						"symbol", symbol,
						"error", err,
					)
					c.errorCount.Add(1)
				} else {
					c.logger.Infow("Published OKX funding rate to Kafka",
						"symbol", symbol,
						"funding_rate", rate.FundingRate,
					)
				}
			}
		}
	}
}

// convertToOKXInstrument converts symbol to OKX instrument format
// Example: BTCUSDT -> BTC-USDT-SWAP
func (c *Client) convertToOKXInstrument(symbol string) string {
	// Simple heuristic: if symbol ends with USDT, split it
	if len(symbol) > 4 && symbol[len(symbol)-4:] == "USDT" {
		base := symbol[:len(symbol)-4]
		return fmt.Sprintf("%s-USDT-SWAP", base)
	}
	c.logger.Warnw("Unable to convert symbol to OKX format",
		"symbol", symbol,
	)
	return symbol
}

// convertFromOKXInstrument converts OKX instrument format back to standard symbol
// Example: BTC-USDT-SWAP -> BTCUSDT
func (c *Client) convertFromOKXInstrument(instID string) string {
	// Remove -SWAP suffix
	if len(instID) > 5 && instID[len(instID)-5:] == "-SWAP" {
		instID = instID[:len(instID)-5]
	}
	// Remove dashes: BTC-USDT -> BTCUSDT
	symbol := ""
	for _, char := range instID {
		if char != '-' {
			symbol += string(char)
		}
	}
	return symbol
}

// Start begins receiving events (already started in Connect)
func (c *Client) Start(ctx context.Context) error {
	if !c.connected.Load() {
		return errors.New("client not connected")
	}

	c.logger.Infow("OKX WebSocket client started",
		"exchange", c.exchange,
	)

	// Monitor context cancellation
	go func() {
		<-ctx.Done()
		if err := c.Stop(context.Background()); err != nil {
			c.logger.Errorw("Error stopping OKX WebSocket client",
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

	c.logger.Infow("Stopping OKX WebSocket client",
		"exchange", c.exchange,
	)

	// Signal stop to event handler
	close(c.stopChan)

	// Wait for handler to finish with timeout
	timeout := 5 * time.Second
	select {
	case <-c.doneChan:
		c.logger.Info("OKX WebSocket handler stopped gracefully")
	case <-time.After(timeout):
		c.logger.Warn("OKX WebSocket handler stop timeout")
	}

	c.connected.Store(false)

	c.logger.Info("✓ OKX WebSocket client stopped")

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
