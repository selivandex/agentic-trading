package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"

	"prometheus/internal/adapters/exchanges"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

const (
	binanceWSURL        = "wss://stream.binance.com:9443/ws"
	binanceFuturesWSURL = "wss://fstream.binance.com/ws"
	binanceTestnetWSURL = "wss://testnet.binance.vision/ws"
	binancePingInterval = 3 * time.Minute
	binanceReadTimeout  = 10 * time.Second
	binanceWriteTimeout = 5 * time.Second
)

// BinanceWSClient implements WebSocket client for Binance
type BinanceWSClient struct {
	apiKey    string
	secretKey string
	testnet   bool
	futures   bool
	conn      *websocket.Conn
	connected bool
	mu        sync.RWMutex
	wg        sync.WaitGroup // Tracks active goroutines for graceful shutdown
	log       *logger.Logger

	// Callbacks
	orderBookCallbacks map[string]func(*exchanges.OrderBook)
	tradesCallbacks    map[string]func(*exchanges.Trade)
	tickerCallbacks    map[string]func(*exchanges.Ticker)
	liqCallbacks       []func(*Liquidation)

	// Control channels
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}
	errChan chan error
}

// NewBinanceWSClient creates a new Binance WebSocket client
func NewBinanceWSClient(apiKey, secretKey string, testnet, futures bool) *BinanceWSClient {
	return &BinanceWSClient{
		apiKey:             apiKey,
		secretKey:          secretKey,
		testnet:            testnet,
		futures:            futures,
		log:                logger.Get().With("component", "binance_ws"),
		orderBookCallbacks: make(map[string]func(*exchanges.OrderBook)),
		tradesCallbacks:    make(map[string]func(*exchanges.Trade)),
		tickerCallbacks:    make(map[string]func(*exchanges.Ticker)),
		liqCallbacks:       make([]func(*Liquidation), 0),
		done:               make(chan struct{}),
		errChan:            make(chan error, 1),
	}
}

// Connect establishes WebSocket connection
func (c *BinanceWSClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	url := c.getBaseURL()
	c.log.Infof("Connecting to Binance WebSocket: %s", url)

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	conn, _, err := dialer.DialContext(ctx, url, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to Binance WebSocket")
	}

	c.conn = conn
	c.connected = true

	// Create child context for graceful shutdown
	c.ctx, c.cancel = context.WithCancel(ctx)

	// Start message reader goroutine
	c.wg.Add(1)
	go c.readMessages()

	// Start ping ticker goroutine
	c.wg.Add(1)
	go c.pingLoop()

	c.log.Info("Binance WebSocket connected successfully")
	return nil
}

// Disconnect closes WebSocket connection gracefully
// Waits for all goroutines to finish (similar to workers.Scheduler.Stop)
func (c *BinanceWSClient) Disconnect() error {
	c.mu.Lock()

	if !c.connected {
		c.mu.Unlock()
		return nil
	}

	c.log.Info("Initiating graceful WebSocket shutdown...")

	// Cancel context to signal all goroutines to stop
	if c.cancel != nil {
		c.cancel()
	}

	// Close done channel
	select {
	case <-c.done:
		// Already closed
	default:
		close(c.done)
	}

	// Send close message to server
	if c.conn != nil {
		err := c.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(1*time.Second),
		)
		if err != nil {
			c.log.Warnf("Error sending close message: %v", err)
		}

		// Close underlying connection
		c.conn.Close()
		c.conn = nil
	}

	c.connected = false
	c.mu.Unlock()

	// Wait for all goroutines to finish with timeout (like scheduler does)
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		c.log.Info("All WebSocket goroutines stopped gracefully")
	case <-time.After(10 * time.Second):
		c.log.Warn("WebSocket shutdown timed out after 10s")
		return errors.Wrap(errors.ErrTimeout, "websocket shutdown timeout")
	}

	c.log.Info("Binance WebSocket disconnected gracefully")
	return nil
}

// SubscribeOrderBook subscribes to order book updates
func (c *BinanceWSClient) SubscribeOrderBook(symbol string, callback func(*exchanges.OrderBook)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Binance depth stream: <symbol>@depth<levels>@<updateSpeed>
	// Example: btcusdt@depth20@100ms
	stream := fmt.Sprintf("%s@depth20@100ms", c.normalizeSymbol(symbol))

	if err := c.subscribe(stream); err != nil {
		return errors.Wrapf(err, "failed to subscribe to order book")
	}

	c.orderBookCallbacks[symbol] = callback
	c.log.Infof("Subscribed to order book for %s (stream: %s)", symbol, stream)

	return nil
}

// SubscribeTrades subscribes to trades stream
func (c *BinanceWSClient) SubscribeTrades(symbol string, callback func(*exchanges.Trade)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Binance trade stream: <symbol>@trade
	// Example: btcusdt@trade
	stream := fmt.Sprintf("%s@trade", c.normalizeSymbol(symbol))

	if err := c.subscribe(stream); err != nil {
		return errors.Wrapf(err, "failed to subscribe to trades")
	}

	c.tradesCallbacks[symbol] = callback
	c.log.Infof("Subscribed to trades for %s (stream: %s)", symbol, stream)

	return nil
}

// SubscribeTicker subscribes to ticker updates
func (c *BinanceWSClient) SubscribeTicker(symbol string, callback func(*exchanges.Ticker)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Binance ticker stream: <symbol>@ticker
	// Example: btcusdt@ticker
	stream := fmt.Sprintf("%s@ticker", c.normalizeSymbol(symbol))

	if err := c.subscribe(stream); err != nil {
		return errors.Wrapf(err, "failed to subscribe to ticker")
	}

	c.tickerCallbacks[symbol] = callback
	c.log.Infof("Subscribed to ticker for %s (stream: %s)", symbol, stream)

	return nil
}

// SubscribeLiquidations subscribes to liquidation events (futures only)
func (c *BinanceWSClient) SubscribeLiquidations(callback func(*Liquidation)) error {
	if !c.futures {
		return errors.Wrapf(errors.ErrInvalidInput, "liquidations only available on futures")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Binance futures liquidation stream: !forceOrder@arr
	stream := "!forceOrder@arr"

	if err := c.subscribe(stream); err != nil {
		return errors.Wrapf(err, "failed to subscribe to liquidations")
	}

	c.liqCallbacks = append(c.liqCallbacks, callback)
	c.log.Info("Subscribed to liquidations stream")

	return nil
}

// IsConnected returns connection status
func (c *BinanceWSClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// Ping sends ping to keep connection alive
func (c *BinanceWSClient) Ping() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return errors.ErrWSNotConnected
	}

	if err := c.conn.SetWriteDeadline(time.Now().Add(binanceWriteTimeout)); err != nil {
		return err
	}

	if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		return errors.Wrapf(err, "failed to send ping")
	}

	return nil
}

// subscribe sends subscription message
func (c *BinanceWSClient) subscribe(stream string) error {
	if !c.connected || c.conn == nil {
		return errors.ErrWSNotConnected
	}

	// Binance subscription message format:
	// {"method":"SUBSCRIBE","params":["<streamName>"],"id":1}
	msg := map[string]interface{}{
		"method": "SUBSCRIBE",
		"params": []string{stream},
		"id":     time.Now().Unix(),
	}

	if err := c.conn.SetWriteDeadline(time.Now().Add(binanceWriteTimeout)); err != nil {
		return err
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		return errors.Wrapf(err, "failed to send subscription")
	}

	return nil
}

// readMessages reads and processes incoming WebSocket messages
// Runs in separate goroutine and respects context cancellation
func (c *BinanceWSClient) readMessages() {
	defer c.wg.Done()
	defer func() {
		c.log.Debug("Message reader goroutine exiting")
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
	}()

	for {
		select {
		case <-c.ctx.Done():
			c.log.Debug("Context cancelled, stopping message reader")
			return
		case <-c.done:
			c.log.Debug("Done channel closed, stopping message reader")
			return
		default:
			// Set read deadline to allow checking context periodically
			if err := c.conn.SetReadDeadline(time.Now().Add(binanceReadTimeout)); err != nil {
				c.log.Errorf("Failed to set read deadline: %v", err)
				return
			}

			_, message, err := c.conn.ReadMessage()
			if err != nil {
				// Check if it's a normal closure
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					c.log.Info("WebSocket closed normally")
					return
				}

				// Check if it's a timeout (expected due to read deadline)
				if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
					// Timeout is expected - allows us to check context
					continue
				}

				c.log.Errorf("Error reading message: %v", err)
				select {
				case c.errChan <- err:
				default:
				}
				return
			}

			// Process message in separate goroutine to avoid blocking reader
			go c.processMessage(message)
		}
	}
}

// processMessage routes incoming message to appropriate handler
func (c *BinanceWSClient) processMessage(message []byte) {
	var base map[string]interface{}
	if err := json.Unmarshal(message, &base); err != nil {
		c.log.Errorf("Failed to unmarshal message: %v", err)
		return
	}

	// Check for event type
	eventType, ok := base["e"].(string)
	if !ok {
		return
	}

	switch eventType {
	case "depthUpdate":
		c.handleDepthUpdate(base)
	case "trade":
		c.handleTrade(base)
	case "24hrTicker":
		c.handleTicker(base)
	case "forceOrder":
		c.handleLiquidation(base)
	default:
		c.log.Debugf("Unhandled event type: %s", eventType)
	}
}

// handleDepthUpdate processes order book depth updates
func (c *BinanceWSClient) handleDepthUpdate(data map[string]interface{}) {
	symbol, _ := data["s"].(string)

	c.mu.RLock()
	callback, ok := c.orderBookCallbacks[symbol]
	c.mu.RUnlock()

	if !ok {
		return
	}

	// Parse bids and asks
	bidsRaw, _ := data["b"].([]interface{})
	asksRaw, _ := data["a"].([]interface{})

	bids := c.parseLevels(bidsRaw)
	asks := c.parseLevels(asksRaw)

	orderBook := &exchanges.OrderBook{
		Symbol:    symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now(),
	}

	callback(orderBook)
}

// handleTrade processes trade updates
func (c *BinanceWSClient) handleTrade(data map[string]interface{}) {
	symbol, _ := data["s"].(string)

	c.mu.RLock()
	callback, ok := c.tradesCallbacks[symbol]
	c.mu.RUnlock()

	if !ok {
		return
	}

	tradeID, _ := data["t"].(float64)
	priceStr, _ := data["p"].(string)
	amountStr, _ := data["q"].(string)
	isBuyerMaker, _ := data["m"].(bool)
	timestamp := time.Unix(0, int64(data["T"].(float64))*int64(time.Millisecond))

	price, _ := decimal.NewFromString(priceStr)
	amount, _ := decimal.NewFromString(amountStr)

	side := exchanges.OrderSideBuy
	if isBuyerMaker {
		side = exchanges.OrderSideSell
	}

	trade := &exchanges.Trade{
		ID:        fmt.Sprintf("%d", int64(tradeID)),
		Symbol:    symbol,
		Price:     price,
		Amount:    amount,
		Side:      side,
		Timestamp: timestamp,
	}

	callback(trade)
}

// handleTicker processes ticker updates
func (c *BinanceWSClient) handleTicker(data map[string]interface{}) {
	symbol, _ := data["s"].(string)

	c.mu.RLock()
	callback, ok := c.tickerCallbacks[symbol]
	c.mu.RUnlock()

	if !ok {
		return
	}

	lastPrice, _ := decimal.NewFromString(data["c"].(string))
	bidPrice, _ := decimal.NewFromString(data["b"].(string))
	askPrice, _ := decimal.NewFromString(data["a"].(string))
	high24h, _ := decimal.NewFromString(data["h"].(string))
	low24h, _ := decimal.NewFromString(data["l"].(string))
	volumeBase, _ := decimal.NewFromString(data["v"].(string))
	volumeQuote, _ := decimal.NewFromString(data["q"].(string))
	change24hPct, _ := decimal.NewFromString(data["P"].(string))

	ticker := &exchanges.Ticker{
		Symbol:       symbol,
		LastPrice:    lastPrice,
		BidPrice:     bidPrice,
		AskPrice:     askPrice,
		High24h:      high24h,
		Low24h:       low24h,
		VolumeBase:   volumeBase,
		VolumeQuote:  volumeQuote,
		Change24hPct: change24hPct,
		Timestamp:    time.Now(),
	}

	callback(ticker)
}

// handleLiquidation processes liquidation events
func (c *BinanceWSClient) handleLiquidation(data map[string]interface{}) {
	c.mu.RLock()
	callbacks := c.liqCallbacks
	c.mu.RUnlock()

	if len(callbacks) == 0 {
		return
	}

	order, _ := data["o"].(map[string]interface{})

	symbol, _ := order["s"].(string)
	sideStr, _ := order["S"].(string)
	priceStr, _ := order["p"].(string)
	qtyStr, _ := order["q"].(string)
	timestamp := time.Unix(0, int64(data["T"].(float64))*int64(time.Millisecond))

	price, _ := decimal.NewFromString(priceStr)
	qty, _ := decimal.NewFromString(qtyStr)
	value := price.Mul(qty)

	side := "long"
	if sideStr == "SELL" {
		side = "short"
	}

	liq := &Liquidation{
		Exchange:  "binance",
		Symbol:    symbol,
		Side:      side,
		Price:     price.InexactFloat64(),
		Quantity:  qty.InexactFloat64(),
		Value:     value.InexactFloat64(),
		Timestamp: timestamp,
	}

	for _, callback := range callbacks {
		go callback(liq)
	}
}

// parseLevels converts raw price levels to OrderBookEntry slice
func (c *BinanceWSClient) parseLevels(levels []interface{}) []exchanges.OrderBookEntry {
	entries := make([]exchanges.OrderBookEntry, 0, len(levels))

	for _, level := range levels {
		levelArr, ok := level.([]interface{})
		if !ok || len(levelArr) < 2 {
			continue
		}

		priceStr, _ := levelArr[0].(string)
		amountStr, _ := levelArr[1].(string)

		price, err := decimal.NewFromString(priceStr)
		if err != nil {
			continue
		}

		amount, err := decimal.NewFromString(amountStr)
		if err != nil {
			continue
		}

		entries = append(entries, exchanges.OrderBookEntry{
			Price:  price,
			Amount: amount,
		})
	}

	return entries
}

// pingLoop sends periodic pings to keep connection alive
// Runs in separate goroutine and respects context cancellation
func (c *BinanceWSClient) pingLoop() {
	defer c.wg.Done()
	ticker := time.NewTicker(binancePingInterval)
	defer ticker.Stop()
	defer c.log.Debug("Ping loop goroutine exiting")

	for {
		select {
		case <-c.ctx.Done():
			c.log.Debug("Context cancelled, stopping ping loop")
			return
		case <-c.done:
			c.log.Debug("Done channel closed, stopping ping loop")
			return
		case <-ticker.C:
			if err := c.Ping(); err != nil {
				c.log.Errorf("Ping failed: %v", err)
				// Don't return here - let reconnect handler deal with it
			}
		}
	}
}

// getBaseURL returns appropriate WebSocket URL based on configuration
func (c *BinanceWSClient) getBaseURL() string {
	if c.testnet {
		return binanceTestnetWSURL
	}
	if c.futures {
		return binanceFuturesWSURL
	}
	return binanceWSURL
}

// normalizeSymbol converts BTC/USDT to btcusdt for Binance
func (c *BinanceWSClient) normalizeSymbol(symbol string) string {
	// Remove slash and convert to lowercase
	normalized := ""
	for _, char := range symbol {
		if char != '/' {
			if char >= 'A' && char <= 'Z' {
				normalized += string(char + 32) // to lowercase
			} else {
				normalized += string(char)
			}
		}
	}
	return normalized
}
