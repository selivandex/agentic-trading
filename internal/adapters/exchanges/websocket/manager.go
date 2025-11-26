package websocket

import (
	"context"
	"sync"
	"time"

	"prometheus/internal/adapters/exchanges"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Manager manages WebSocket connections to multiple exchanges
type Manager struct {
	clients map[string]WebSocketClient
	mu      sync.RWMutex
	log     *logger.Logger
}

// WebSocketClient represents a WebSocket client for a specific exchange
type WebSocketClient interface {
	// Connect establishes WebSocket connection
	Connect(ctx context.Context) error

	// Disconnect closes WebSocket connection
	Disconnect() error

	// SubscribeOrderBook subscribes to order book updates
	SubscribeOrderBook(symbol string, callback func(*exchanges.OrderBook)) error

	// SubscribeTrades subscribes to trades (tape) stream
	SubscribeTrades(symbol string, callback func(*exchanges.Trade)) error

	// SubscribeTicker subscribes to ticker updates
	SubscribeTicker(symbol string, callback func(*exchanges.Ticker)) error

	// SubscribeLiquidations subscribes to liquidation events
	SubscribeLiquidations(callback func(*Liquidation)) error

	// IsConnected returns connection status
	IsConnected() bool

	// Ping sends ping to keep connection alive
	Ping() error
}

// Liquidation represents a liquidation event
type Liquidation struct {
	Exchange  string
	Symbol    string
	Side      string // long, short
	Price     float64
	Quantity  float64
	Value     float64 // USD value
	Timestamp time.Time
}

// NewManager creates a new WebSocket manager
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]WebSocketClient),
		log:     logger.Get().With("component", "ws_manager"),
	}
}

// RegisterClient registers a WebSocket client for an exchange
func (m *Manager) RegisterClient(exchange string, client WebSocketClient) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients[exchange] = client
	m.log.Infof("Registered WebSocket client for %s", exchange)
}

// GetClient returns WebSocket client for an exchange
func (m *Manager) GetClient(exchange string) (WebSocketClient, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, ok := m.clients[exchange]
	if !ok {
		return nil, errors.Wrapf(errors.ErrNotFound, "no WebSocket client for exchange: %s", exchange)
	}

	return client, nil
}

// ConnectAll connects all registered clients
func (m *Manager) ConnectAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for exchange, client := range m.clients {
		if err := client.Connect(ctx); err != nil {
			m.log.Errorf("Failed to connect WebSocket for %s: %v", exchange, err)
			// Continue with other exchanges
			continue
		}
		m.log.Infof("WebSocket connected for %s", exchange)
	}

	return nil
}

// DisconnectAll disconnects all clients
func (m *Manager) DisconnectAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var lastErr error
	for exchange, client := range m.clients {
		if err := client.Disconnect(); err != nil {
			m.log.Errorf("Failed to disconnect WebSocket for %s: %v", exchange, err)
			lastErr = err
		} else {
			m.log.Infof("WebSocket disconnected for %s", exchange)
		}
	}

	return lastErr
}

// HealthCheck checks health of all connections
func (m *Manager) HealthCheck() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health := make(map[string]bool)
	for exchange, client := range m.clients {
		health[exchange] = client.IsConnected()
	}

	return health
}

// MonitorConnections monitors connection health and auto-reconnects
func (m *Manager) MonitorConnections(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkAndReconnect(ctx)
		}
	}
}

// checkAndReconnect checks connection health and reconnects if needed
func (m *Manager) checkAndReconnect(ctx context.Context) {
	m.mu.RLock()
	clients := make(map[string]WebSocketClient, len(m.clients))
	for exchange, client := range m.clients {
		clients[exchange] = client
	}
	m.mu.RUnlock()

	for exchange, client := range clients {
		if !client.IsConnected() {
			m.log.Warnf("WebSocket for %s is disconnected, attempting reconnect", exchange)
			if err := client.Connect(ctx); err != nil {
				m.log.Errorf("Failed to reconnect WebSocket for %s: %v", exchange, err)
			} else {
				m.log.Infof("Successfully reconnected WebSocket for %s", exchange)
			}
		}
	}
}

// ListExchanges returns list of registered exchanges
func (m *Manager) ListExchanges() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	exchanges := make([]string, 0, len(m.clients))
	for exchange := range m.clients {
		exchanges = append(exchanges, exchange)
	}

	return exchanges
}
