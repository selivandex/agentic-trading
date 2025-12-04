package reconnect

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Manager manages reconnections with exponential backoff and circuit breaker
// This is a general-purpose reconnection manager that can be used for WebSocket, Redis, Kafka, etc.
type Manager struct {
	// Configuration
	minBackoff          time.Duration
	maxBackoff          time.Duration
	backoffMultiplier   float64
	maxRetries          int
	healthCheckInterval time.Duration
	heartbeatTimeout    time.Duration // Max time without messages before considering connection dead

	// State
	mu                  sync.RWMutex
	currentBackoff      time.Duration
	consecutiveFailures int
	totalReconnects     int
	circuitOpen         bool // Circuit breaker open = stop trying
	circuitOpenedAt     time.Time
	circuitResetAfter   time.Duration

	// Heartbeat tracking
	lastMessageTime atomic.Int64 // Unix timestamp in seconds
	lastHealthCheck atomic.Int64

	logger *logger.Logger
}

// Config configures the reconnect manager
type Config struct {
	MinBackoff          time.Duration // Initial backoff (e.g. 1s)
	MaxBackoff          time.Duration // Max backoff (e.g. 5min)
	BackoffMultiplier   float64       // Multiplier for exponential backoff (e.g. 2.0)
	MaxRetries          int           // Max consecutive retries before opening circuit (0 = unlimited)
	HealthCheckInterval time.Duration // How often to check connection health (e.g. 10s)
	HeartbeatTimeout    time.Duration // Max time without messages before reconnect (e.g. 60s)
	CircuitResetAfter   time.Duration // How long to wait before trying again after circuit opens (e.g. 5min)
}

// NewManager creates a new reconnect manager with sensible defaults
func NewManager(config Config, log *logger.Logger) *Manager {
	// Set defaults
	if config.MinBackoff == 0 {
		config.MinBackoff = 1 * time.Second
	}
	if config.MaxBackoff == 0 {
		config.MaxBackoff = 5 * time.Minute
	}
	if config.BackoffMultiplier == 0 {
		config.BackoffMultiplier = 2.0
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 10 // Default: stop after 10 consecutive failures
	}
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 10 * time.Second
	}
	if config.HeartbeatTimeout == 0 {
		config.HeartbeatTimeout = 60 * time.Second
	}
	if config.CircuitResetAfter == 0 {
		config.CircuitResetAfter = 5 * time.Minute
	}

	return &Manager{
		minBackoff:          config.MinBackoff,
		maxBackoff:          config.MaxBackoff,
		backoffMultiplier:   config.BackoffMultiplier,
		maxRetries:          config.MaxRetries,
		healthCheckInterval: config.HealthCheckInterval,
		heartbeatTimeout:    config.HeartbeatTimeout,
		circuitResetAfter:   config.CircuitResetAfter,
		currentBackoff:      config.MinBackoff,
		logger:              log,
	}
}

// RecordMessageReceived updates the last message timestamp
// Call this every time a message is received from the connection
func (m *Manager) RecordMessageReceived() {
	m.lastMessageTime.Store(time.Now().Unix())
}

// IsHealthy checks if connection is healthy based on recent message activity
func (m *Manager) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check circuit breaker
	if m.circuitOpen {
		// Check if we can try again
		if time.Since(m.circuitOpenedAt) > m.circuitResetAfter {
			m.logger.Infow("Circuit breaker reset period elapsed, allowing retry",
				"wait_time", m.circuitResetAfter,
			)
			return false // Not healthy yet, but we can try reconnecting
		}
		return false
	}

	// Check heartbeat
	lastMsg := time.Unix(m.lastMessageTime.Load(), 0)
	if lastMsg.IsZero() {
		// No messages received yet - consider healthy (just connected)
		return true
	}

	timeSinceLastMessage := time.Since(lastMsg)
	if timeSinceLastMessage > m.heartbeatTimeout {
		m.logger.Warnw("Connection appears dead - no messages received",
			"time_since_last_message", timeSinceLastMessage,
			"heartbeat_timeout", m.heartbeatTimeout,
		)
		return false
	}

	return true
}

// ShouldRetry returns whether we should attempt reconnection
func (m *Manager) ShouldRetry() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check circuit breaker
	if m.circuitOpen {
		if time.Since(m.circuitOpenedAt) < m.circuitResetAfter {
			return false
		}
		// Circuit reset period elapsed - allow retry
		return true
	}

	// Check max retries
	if m.maxRetries > 0 && m.consecutiveFailures >= m.maxRetries {
		return false
	}

	return true
}

// GetBackoff returns current backoff duration
func (m *Manager) GetBackoff() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentBackoff
}

// RecordFailure records a reconnection failure and updates backoff
func (m *Manager) RecordFailure() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.consecutiveFailures++

	// Exponential backoff
	newBackoff := time.Duration(float64(m.currentBackoff) * m.backoffMultiplier)
	if newBackoff > m.maxBackoff {
		newBackoff = m.maxBackoff
	}
	m.currentBackoff = newBackoff

	m.logger.Warnw("Reconnection failed",
		"consecutive_failures", m.consecutiveFailures,
		"next_backoff", m.currentBackoff,
	)

	// Open circuit breaker if too many failures
	if m.maxRetries > 0 && m.consecutiveFailures >= m.maxRetries {
		m.circuitOpen = true
		m.circuitOpenedAt = time.Now()

		m.logger.Errorw("🔴 Circuit breaker OPENED - too many consecutive failures",
			"consecutive_failures", m.consecutiveFailures,
			"max_retries", m.maxRetries,
			"circuit_reset_after", m.circuitResetAfter,
		)
	}
}

// RecordSuccess records a successful reconnection
func (m *Manager) RecordSuccess() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.consecutiveFailures > 0 {
		m.logger.Infow("✅ Reconnection successful, resetting backoff",
			"previous_consecutive_failures", m.consecutiveFailures,
		)
	}

	// Reset backoff and failure counter
	m.currentBackoff = m.minBackoff
	m.consecutiveFailures = 0
	m.totalReconnects++

	// Close circuit breaker
	if m.circuitOpen {
		m.logger.Infow("🟢 Circuit breaker CLOSED - connection restored",
			"total_reconnects", m.totalReconnects,
		)
		m.circuitOpen = false
		m.circuitOpenedAt = time.Time{}
	}

	// Update heartbeat
	m.lastMessageTime.Store(time.Now().Unix())
}

// ResetCircuit manually resets the circuit breaker
func (m *Manager) ResetCircuit() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.circuitOpen {
		m.logger.Infow("Manually resetting circuit breaker")
		m.circuitOpen = false
		m.circuitOpenedAt = time.Time{}
		m.consecutiveFailures = 0
		m.currentBackoff = m.minBackoff
	}
}

// GetStats returns current reconnect manager stats
func (m *Manager) GetStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	lastMsg := time.Unix(m.lastMessageTime.Load(), 0)
	timeSinceLastMessage := time.Duration(0)
	if !lastMsg.IsZero() {
		timeSinceLastMessage = time.Since(lastMsg)
	}

	return Stats{
		ConsecutiveFailures:  m.consecutiveFailures,
		TotalReconnects:      m.totalReconnects,
		CurrentBackoff:       m.currentBackoff,
		CircuitOpen:          m.circuitOpen,
		CircuitOpenedAt:      m.circuitOpenedAt,
		LastMessageTime:      lastMsg,
		TimeSinceLastMessage: timeSinceLastMessage,
		IsHealthy:            m.IsHealthy(),
	}
}

// Stats contains reconnection statistics
type Stats struct {
	ConsecutiveFailures  int
	TotalReconnects      int
	CurrentBackoff       time.Duration
	CircuitOpen          bool
	CircuitOpenedAt      time.Time
	LastMessageTime      time.Time
	TimeSinceLastMessage time.Duration
	IsHealthy            bool
}

// ReconnectWithBackoff executes a reconnect function with exponential backoff
// Returns nil on success, error on failure (including circuit breaker open)
func (m *Manager) ReconnectWithBackoff(ctx context.Context, reconnectFn func(context.Context) error) error {
	if !m.ShouldRetry() {
		m.mu.RLock()
		circuitOpen := m.circuitOpen
		failures := m.consecutiveFailures
		m.mu.RUnlock()

		if circuitOpen {
			return errors.New("circuit breaker is open - too many consecutive failures")
		}
		return errors.Newf("max retries reached: %d consecutive failures", failures)
	}

	// Get backoff duration
	backoff := m.GetBackoff()

	// Wait backoff duration before attempting
	if backoff > 0 {
		m.logger.Infow("⏳ Waiting before reconnect attempt",
			"backoff", backoff,
		)

		select {
		case <-time.After(backoff):
			// Continue to reconnect
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Attempt reconnection
	m.logger.Infow("🔄 Attempting reconnection...")

	if err := reconnectFn(ctx); err != nil {
		m.RecordFailure()
		return errors.Wrap(err, "reconnection failed")
	}

	m.RecordSuccess()
	return nil
}

// CalculateJitter adds jitter to backoff to prevent thundering herd
func CalculateJitter(duration time.Duration, jitterPercent float64) time.Duration {
	if jitterPercent <= 0 || jitterPercent > 1 {
		return duration
	}

	maxJitter := float64(duration) * jitterPercent
	jitter := time.Duration(math.Float64frombits(uint64(time.Now().UnixNano())) * maxJitter)

	return duration + jitter
}
