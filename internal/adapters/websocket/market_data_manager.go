package websocket

import (
	"context"
	"sync"
	"time"

	"prometheus/internal/metrics"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// MarketDataManager manages Market Data WebSocket connections with health monitoring and auto-reconnect
// Unlike UserDataManager (which manages per-user connections), this manages centralized market data streams
type MarketDataManager struct {
	client Client
	config ConnectionConfig
	logger *logger.Logger

	// Health monitoring
	mu                     sync.RWMutex
	connected              bool
	lastHealthCheck        time.Time
	consecutiveFailures    int
	healthCheckInterval    time.Duration
	maxConsecutiveFailures int

	// Control channels
	stopChan chan struct{}
	doneChan chan struct{}

	// Stats
	totalReconnects int
	statsMu         sync.RWMutex
}

// MarketDataManagerConfig configures the MarketDataManager
type MarketDataManagerConfig struct {
	HealthCheckInterval    time.Duration // How often to check connection health
	MaxConsecutiveFailures int           // Max failures before logging warning
}

// NewMarketDataManager creates a new Market Data WebSocket manager
func NewMarketDataManager(
	client Client,
	config ConnectionConfig,
	managerConfig MarketDataManagerConfig,
	log *logger.Logger,
) *MarketDataManager {
	if managerConfig.HealthCheckInterval == 0 {
		managerConfig.HealthCheckInterval = 60 * time.Second
	}
	if managerConfig.MaxConsecutiveFailures == 0 {
		managerConfig.MaxConsecutiveFailures = 3
	}

	return &MarketDataManager{
		client:                 client,
		config:                 config,
		logger:                 log,
		stopChan:               make(chan struct{}),
		doneChan:               make(chan struct{}),
		healthCheckInterval:    managerConfig.HealthCheckInterval,
		maxConsecutiveFailures: managerConfig.MaxConsecutiveFailures,
	}
}

// Start initializes the Market Data WebSocket connection and starts health monitoring
func (m *MarketDataManager) Start(ctx context.Context) error {
	m.logger.Infow("Starting Market Data Manager...")

	// Initial connection
	if err := m.connect(ctx); err != nil {
		m.logger.Errorw("Failed initial connection", "error", err)
		// Don't fail startup - health check will retry
	}

	// Start background health monitoring
	go m.healthCheckLoop(ctx)

	m.logger.Infow("✓ Market Data Manager started",
		"health_check_interval", m.healthCheckInterval,
	)

	return nil
}

// connect establishes WebSocket connection
func (m *MarketDataManager) connect(ctx context.Context) error {
	m.logger.Infow("🔌 Connecting Market Data WebSocket...")

	// Connect
	if err := m.client.Connect(ctx, m.config); err != nil {
		return errors.Wrap(err, "failed to connect")
	}

	// Start receiving events
	if err := m.client.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start")
	}

	m.mu.Lock()
	m.connected = true
	m.consecutiveFailures = 0
	m.mu.Unlock()

	m.logger.Infow("✅ Market Data WebSocket connected",
		"streams", len(m.config.Streams),
	)

	return nil
}

// Stop gracefully shuts down the manager
func (m *MarketDataManager) Stop(ctx context.Context) error {
	m.logger.Infow("Stopping Market Data Manager...")

	// Signal health check loop to stop
	close(m.stopChan)

	// Stop client
	if err := m.client.Stop(ctx); err != nil {
		m.logger.Errorw("Failed to stop client", "error", err)
	}

	m.mu.Lock()
	m.connected = false
	m.mu.Unlock()

	// Wait for health check loop to finish
	select {
	case <-m.doneChan:
		m.logger.Infow("✓ Market Data Manager stopped")
	case <-time.After(5 * time.Second):
		m.logger.Warnw("Market Data Manager stop timeout")
	}

	return nil
}

// healthCheckLoop periodically checks connection health and reconnects if needed
func (m *MarketDataManager) healthCheckLoop(ctx context.Context) {
	defer close(m.doneChan)

	ticker := time.NewTicker(m.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.performHealthCheck(ctx)
		case <-m.stopChan:
			m.logger.Debugw("Health check loop stopped")
			return
		case <-ctx.Done():
			m.logger.Debugw("Health check loop cancelled by context")
			return
		}
	}
}

// performHealthCheck checks connection health and reconnects if needed
func (m *MarketDataManager) performHealthCheck(ctx context.Context) {
	m.mu.Lock()
	m.lastHealthCheck = time.Now()
	m.mu.Unlock()

	isConnected := m.client.IsConnected()

	if !isConnected {
		m.mu.Lock()
		m.consecutiveFailures++
		failures := m.consecutiveFailures
		m.mu.Unlock()

		if failures >= m.maxConsecutiveFailures {
			m.logger.Warnw("⚠️ Market Data WebSocket disconnected, attempting reconnect",
				"consecutive_failures", failures,
			)
		} else {
			m.logger.Debugw("Market Data WebSocket disconnected, attempting reconnect",
				"consecutive_failures", failures,
			)
		}

		// Attempt reconnect
		if err := m.reconnect(ctx); err != nil {
			m.logger.Errorw("Failed to reconnect Market Data WebSocket",
				"error", err,
				"consecutive_failures", failures,
			)

			// Update metrics
			metrics.MarketDataReconnects.WithLabelValues("failed").Inc()
		} else {
			m.logger.Infow("✅ Market Data WebSocket reconnected successfully",
				"total_reconnects", m.getTotalReconnects(),
			)

			// Update metrics
			metrics.MarketDataReconnects.WithLabelValues("success").Inc()
		}
	} else {
		// Connection is healthy
		m.mu.Lock()
		if m.consecutiveFailures > 0 {
			m.logger.Debugw("Market Data WebSocket connection healthy",
				"consecutive_failures_cleared", m.consecutiveFailures,
			)
			m.consecutiveFailures = 0
		}
		m.mu.Unlock()
	}
}

// reconnect stops the current connection and establishes a new one
func (m *MarketDataManager) reconnect(ctx context.Context) error {
	m.logger.Debugw("Stopping old connection before reconnect...")

	// Stop old connection
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := m.client.Stop(stopCtx); err != nil {
		m.logger.Warnw("Failed to stop old connection gracefully",
			"error", err,
		)
	}
	cancel()

	m.mu.Lock()
	m.connected = false
	m.mu.Unlock()

	// Wait a bit before reconnecting
	time.Sleep(2 * time.Second)

	// Reconnect
	if err := m.connect(ctx); err != nil {
		return errors.Wrap(err, "failed to reconnect")
	}

	m.statsMu.Lock()
	m.totalReconnects++
	m.statsMu.Unlock()

	return nil
}

// IsConnected returns current connection status
func (m *MarketDataManager) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// GetStats returns manager statistics
func (m *MarketDataManager) GetStats() map[string]interface{} {
	m.statsMu.RLock()
	totalReconnects := m.totalReconnects
	m.statsMu.RUnlock()

	m.mu.RLock()
	connected := m.connected
	lastCheck := m.lastHealthCheck
	failures := m.consecutiveFailures
	m.mu.RUnlock()

	return map[string]interface{}{
		"connected":            connected,
		"total_reconnects":     totalReconnects,
		"last_health_check":    lastCheck,
		"consecutive_failures": failures,
	}
}

// getTotalReconnects returns total reconnect count (thread-safe)
func (m *MarketDataManager) getTotalReconnects() int {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()
	return m.totalReconnects
}


