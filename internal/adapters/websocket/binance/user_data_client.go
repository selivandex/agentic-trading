package binance

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/google/uuid"

	"prometheus/internal/adapters/websocket"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// UserDataClient implements websocket.UserDataStreamer for Binance Futures
// Handles real-time order updates, position updates, balance updates, and margin calls
type UserDataClient struct {
	accountID uuid.UUID
	userID    uuid.UUID
	exchange  string
	handler   websocket.UserDataEventHandler

	// Authentication
	apiKey     string
	secretKey  string
	useTestnet bool

	// ListenKey management
	listenKeyService *ListenKeyService
	currentListenKey string
	keyExpiresAt     time.Time
	keyMu            sync.RWMutex

	// Connection management
	mu          sync.RWMutex
	connected   atomic.Bool
	stopping    atomic.Bool
	stopChannel chan struct{}
	doneChannel chan struct{}
	renewTicker *time.Ticker
	renewStop   chan struct{}

	// Statistics
	stats            websocket.UserDataStats
	statsMu          sync.RWMutex
	messagesReceived atomic.Int64
	orderUpdates     atomic.Int64
	positionUpdates  atomic.Int64
	balanceUpdates   atomic.Int64
	marginCalls      atomic.Int64
	errorCount       atomic.Int64
	reconnectCount   atomic.Int32

	logger *logger.Logger
}

// NewUserDataClient creates a new Binance User Data WebSocket client
func NewUserDataClient(
	accountID, userID uuid.UUID,
	exchange string,
	handler websocket.UserDataEventHandler,
	useTestnet bool,
	log *logger.Logger,
) *UserDataClient {
	return &UserDataClient{
		accountID:        accountID,
		userID:           userID,
		exchange:         exchange,
		handler:          handler,
		useTestnet:       useTestnet,
		listenKeyService: NewListenKeyService(useTestnet, log),
		logger:           log,
	}
}

// CreateListenKey generates a new listenKey via REST API
func (c *UserDataClient) CreateListenKey(ctx context.Context, apiKey, secret string) (string, time.Time, error) {
	c.apiKey = apiKey
	c.secretKey = secret

	listenKey, expiresAt, err := c.listenKeyService.Create(ctx, apiKey, secret)
	if err != nil {
		return "", time.Time{}, err
	}

	c.keyMu.Lock()
	c.currentListenKey = listenKey
	c.keyExpiresAt = expiresAt
	c.keyMu.Unlock()

	c.logger.Infow("Created listenKey for User Data Stream",
		"account_id", c.accountID,
		"user_id", c.userID,
		"expires_at", expiresAt,
	)

	return listenKey, expiresAt, nil
}

// RenewListenKey extends the validity period of the current listenKey
func (c *UserDataClient) RenewListenKey(ctx context.Context, apiKey, secret, listenKey string) error {
	expiresAt, err := c.listenKeyService.Renew(ctx, apiKey, secret, listenKey)
	if err != nil {
		return err
	}

	c.keyMu.Lock()
	c.keyExpiresAt = expiresAt
	c.keyMu.Unlock()

	c.statsMu.Lock()
	c.stats.ListenKeyExpiresAt = expiresAt
	c.statsMu.Unlock()

	c.logger.Debugw("Renewed listenKey for User Data Stream",
		"account_id", c.accountID,
		"new_expires_at", expiresAt,
	)

	return nil
}

// DeleteListenKey closes the User Data Stream and invalidates the listenKey
func (c *UserDataClient) DeleteListenKey(ctx context.Context, apiKey, secret, listenKey string) error {
	if err := c.listenKeyService.Delete(ctx, apiKey, secret, listenKey); err != nil {
		return err
	}

	c.keyMu.Lock()
	c.currentListenKey = ""
	c.keyExpiresAt = time.Time{}
	c.keyMu.Unlock()

	return nil
}

// Connect establishes User Data WebSocket connection
func (c *UserDataClient) Connect(ctx context.Context, listenKey, apiKey, secret string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected.Load() {
		return errors.New("client already connected")
	}

	c.apiKey = apiKey
	c.secretKey = secret

	c.keyMu.Lock()
	c.currentListenKey = listenKey
	c.keyMu.Unlock()

	futures.UseTestnet = c.useTestnet

	c.logger.Infow("Connecting to Binance User Data Stream",
		"account_id", c.accountID,
		"user_id", c.userID,
		"testnet", c.useTestnet,
	)

	errHandler := func(err error) {
		c.errorCount.Add(1)
		c.statsMu.Lock()
		c.stats.LastError = err
		c.statsMu.Unlock()

		c.logger.Error("User Data WebSocket error",
			"account_id", c.accountID,
			"error", err.Error(),
		)

		if c.handler != nil {
			c.handler.OnError(err)
		}
	}

	// Handler for all User Data events
	handler := func(event *futures.WsUserDataEvent) {
		c.messagesReceived.Add(1)

		if c.stopping.Load() {
			return
		}

		if c.handler == nil {
			return
		}

		// Route event by type
		switch event.Event {
		case futures.UserDataEventTypeOrderTradeUpdate:
			c.handleOrderUpdate(ctx, event)
		case futures.UserDataEventTypeAccountUpdate:
			c.handleAccountUpdate(ctx, event)
		case futures.UserDataEventTypeMarginCall:
			c.handleMarginCall(ctx, event)
		case futures.UserDataEventTypeAccountConfigUpdate:
			c.handleAccountConfigUpdate(ctx, event)
		default:
			c.logger.Warn("Unknown User Data event type",
				"account_id", c.accountID,
				"event_type", event.Event,
			)
		}
	}

	// Start User Data WebSocket with listenKey
	doneC, stopC, err := futures.WsUserDataServe(listenKey, handler, errHandler)
	if err != nil {
		return errors.Wrap(err, "failed to start User Data WebSocket")
	}

	c.stopChannel = stopC
	c.doneChannel = doneC
	c.connected.Store(true)

	c.statsMu.Lock()
	c.stats.AccountID = c.accountID
	c.stats.Exchange = c.exchange
	c.stats.ConnectedSince = time.Now()
	c.stats.ListenKeyExpiresAt = c.keyExpiresAt
	c.statsMu.Unlock()

	c.logger.Infow("✓ Connected to User Data Stream",
		"account_id", c.accountID,
		"user_id", c.userID,
	)

	return nil
}

// Start begins receiving events and starts listenKey renewal
func (c *UserDataClient) Start(ctx context.Context) error {
	if !c.connected.Load() {
		return errors.New("client not connected")
	}

	// Start listenKey renewal worker (every 30 minutes)
	c.renewTicker = time.NewTicker(30 * time.Minute)
	c.renewStop = make(chan struct{})

	go c.listenKeyRenewalLoop(ctx)

	c.logger.Infow("User Data WebSocket client started",
		"account_id", c.accountID,
	)

	// Monitor context cancellation
	go func() {
		<-ctx.Done()
		if err := c.Stop(context.Background()); err != nil {
			c.logger.Error("Error stopping User Data WebSocket",
				"account_id", c.accountID,
				"error", err.Error(),
			)
		}
	}()

	return nil
}

// listenKeyRenewalLoop periodically renews the listenKey
func (c *UserDataClient) listenKeyRenewalLoop(ctx context.Context) {
	for {
		select {
		case <-c.renewTicker.C:
			c.keyMu.RLock()
			listenKey := c.currentListenKey
			c.keyMu.RUnlock()

			if listenKey == "" {
				c.logger.Warn("No listenKey to renew",
					"account_id", c.accountID,
				)
				continue
			}

			if err := c.RenewListenKey(ctx, c.apiKey, c.secretKey, listenKey); err != nil {
				c.logger.Error("Failed to renew listenKey",
					"account_id", c.accountID,
					"error", err,
				)
			}

		case <-c.renewStop:
			c.renewTicker.Stop()
			return
		case <-ctx.Done():
			c.renewTicker.Stop()
			return
		}
	}
}

// handleOrderUpdate processes ORDER_TRADE_UPDATE events
func (c *UserDataClient) handleOrderUpdate(ctx context.Context, event *futures.WsUserDataEvent) {
	c.orderUpdates.Add(1)

	orderEvent := &websocket.OrderUpdateEvent{
		UserID:          c.userID,
		AccountID:       c.accountID,
		Exchange:        c.exchange,
		OrderID:         fmt.Sprintf("%d", event.OrderTradeUpdate.ID),
		ClientOrderID:   event.OrderTradeUpdate.ClientOrderID,
		Symbol:          event.OrderTradeUpdate.Symbol,
		Side:            string(event.OrderTradeUpdate.Side),
		PositionSide:    string(event.OrderTradeUpdate.PositionSide),
		Type:            string(event.OrderTradeUpdate.Type),
		Status:          string(event.OrderTradeUpdate.Status),
		ExecutionType:   string(event.OrderTradeUpdate.ExecutionType),
		OriginalQty:     event.OrderTradeUpdate.OriginalQty,
		FilledQty:       event.OrderTradeUpdate.AccumulatedFilledQty,
		AvgPrice:        event.OrderTradeUpdate.AveragePrice,
		StopPrice:       event.OrderTradeUpdate.StopPrice,
		LastFilledQty:   event.OrderTradeUpdate.LastFilledQty,
		LastFilledPrice: event.OrderTradeUpdate.LastFilledPrice,
		Commission:      event.OrderTradeUpdate.Commission,
		CommissionAsset: event.OrderTradeUpdate.CommissionAsset,
		TradeTime:       time.UnixMilli(event.OrderTradeUpdate.TradeTime),
		EventTime:       time.UnixMilli(event.Time),
	}

	if err := c.handler.OnOrderUpdate(ctx, orderEvent); err != nil {
		c.logger.Errorw("Failed to handle order update",
			"account_id", c.accountID,
			"order_id", orderEvent.OrderID,
			"error", err,
		)
		c.errorCount.Add(1)
	}
}

// handleAccountUpdate processes ACCOUNT_UPDATE events (positions & balances)
func (c *UserDataClient) handleAccountUpdate(ctx context.Context, event *futures.WsUserDataEvent) {
	// Process position updates
	for _, pos := range event.AccountUpdate.Positions {
		c.positionUpdates.Add(1)

		posEvent := &websocket.PositionUpdateEvent{
			UserID:            c.userID,
			AccountID:         c.accountID,
			Exchange:          c.exchange,
			Symbol:            pos.Symbol,
			Side:              determineSide(pos.Amount),
			Amount:            pos.Amount,
			EntryPrice:        pos.EntryPrice,
			MarkPrice:         "", // Not provided in event
			UnrealizedPnL:     pos.UnrealizedPnL,
			MaintenanceMargin: "",                        // Not provided in event
			PositionSide:      determineSide(pos.Amount), // Derive from amount
			EventTime:         time.UnixMilli(event.Time),
		}

		if err := c.handler.OnPositionUpdate(ctx, posEvent); err != nil {
			c.logger.Errorw("Failed to handle position update",
				"account_id", c.accountID,
				"symbol", pos.Symbol,
				"error", err,
			)
			c.errorCount.Add(1)
		}
	}

	// Process balance updates
	for _, bal := range event.AccountUpdate.Balances {
		c.balanceUpdates.Add(1)

		balEvent := &websocket.BalanceUpdateEvent{
			UserID:             c.userID,
			AccountID:          c.accountID,
			Exchange:           c.exchange,
			Asset:              bal.Asset,
			WalletBalance:      bal.Balance,
			CrossWalletBalance: bal.CrossWalletBalance,
			AvailableBalance:   "", // Calculate: Balance - margin used
			EventTime:          time.UnixMilli(event.Time),
			ReasonType:         string(event.AccountUpdate.Reason),
		}

		if err := c.handler.OnBalanceUpdate(ctx, balEvent); err != nil {
			c.logger.Errorw("Failed to handle balance update",
				"account_id", c.accountID,
				"asset", bal.Asset,
				"error", err,
			)
			c.errorCount.Add(1)
		}
	}
}

// handleMarginCall processes MARGIN_CALL events (CRITICAL!)
func (c *UserDataClient) handleMarginCall(ctx context.Context, event *futures.WsUserDataEvent) {
	c.marginCalls.Add(1)

	// TODO: Binance go-binance library may not have MarginCall struct exposed yet
	// For now, log the event and extract what we can
	c.logger.Error("⚠️ MARGIN CALL RECEIVED",
		"account_id", c.accountID,
		"user_id", c.userID,
		"event_type", event.Event,
	)

	// Create a minimal margin call event
	marginEvent := &websocket.MarginCallEvent{
		UserID:             c.userID,
		AccountID:          c.accountID,
		Exchange:           c.exchange,
		CrossWalletBalance: "",                           // Not accessible in current library version
		PositionsAtRisk:    []websocket.PositionAtRisk{}, // Not accessible in current library version
		EventTime:          time.UnixMilli(event.Time),
	}

	if err := c.handler.OnMarginCall(ctx, marginEvent); err != nil {
		c.logger.Errorw("Failed to handle margin call",
			"account_id", c.accountID,
			"error", err,
		)
		c.errorCount.Add(1)
	}
}

// handleAccountConfigUpdate processes leverage/margin mode changes
func (c *UserDataClient) handleAccountConfigUpdate(ctx context.Context, event *futures.WsUserDataEvent) {
	configEvent := &websocket.AccountConfigEvent{
		UserID:    c.userID,
		AccountID: c.accountID,
		Exchange:  c.exchange,
		Symbol:    event.AccountConfigUpdate.Symbol,
		Leverage:  int(event.AccountConfigUpdate.Leverage),
		EventTime: time.UnixMilli(event.Time),
	}

	if err := c.handler.OnAccountConfigUpdate(ctx, configEvent); err != nil {
		c.logger.Errorw("Failed to handle account config update",
			"account_id", c.accountID,
			"error", err,
		)
		c.errorCount.Add(1)
	}
}

// Stop gracefully closes the WebSocket connection
func (c *UserDataClient) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected.Load() {
		return nil
	}

	c.stopping.Store(true)

	// Stop listenKey renewal
	if c.renewStop != nil {
		close(c.renewStop)
	}

	c.logger.Infow("Stopping User Data WebSocket",
		"account_id", c.accountID,
	)

	// Signal stream to stop
	if c.stopChannel != nil {
		select {
		case c.stopChannel <- struct{}{}:
		default:
			close(c.stopChannel)
		}
	}

	// Wait for stream to finish
	if c.doneChannel != nil {
		timeout := 5 * time.Second
		select {
		case <-c.doneChannel:
			c.logger.Infow("User Data WebSocket stopped gracefully",
				"account_id", c.accountID,
			)
		case <-time.After(timeout):
			c.logger.Warn("User Data WebSocket stop timeout",
				"account_id", c.accountID,
				"timeout", timeout,
			)
		}
	}

	c.connected.Store(false)
	c.stopChannel = nil
	c.doneChannel = nil

	return nil
}

// IsConnected returns connection status
func (c *UserDataClient) IsConnected() bool {
	return c.connected.Load()
}

// GetStats returns current statistics
func (c *UserDataClient) GetStats() websocket.UserDataStats {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()

	stats := c.stats
	stats.MessagesReceived = c.messagesReceived.Load()
	stats.OrderUpdates = c.orderUpdates.Load()
	stats.PositionUpdates = c.positionUpdates.Load()
	stats.BalanceUpdates = c.balanceUpdates.Load()
	stats.MarginCalls = c.marginCalls.Load()
	stats.ErrorCount = c.errorCount.Load()
	stats.ReconnectCount = int(c.reconnectCount.Load())

	return stats
}

// determineSide determines position side from amount
func determineSide(amount string) string {
	if amount == "" || amount == "0" {
		return ""
	}
	if amount[0] == '-' {
		return "SHORT"
	}
	return "LONG"
}
