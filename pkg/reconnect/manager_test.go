package reconnect

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"prometheus/pkg/logger"
)

func newTestLogger() *logger.Logger {
	zapLog, _ := zap.NewDevelopment()
	return &logger.Logger{SugaredLogger: zapLog.Sugar()}
}

func TestNewManager(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		expectedMin    time.Duration
		expectedMax    time.Duration
		expectedMult   float64
		expectedRetry  int
		expectedHealth time.Duration
		expectedHB     time.Duration
		expectedReset  time.Duration
	}{
		{
			name:           "all defaults",
			config:         Config{},
			expectedMin:    1 * time.Second,
			expectedMax:    5 * time.Minute,
			expectedMult:   2.0,
			expectedRetry:  10,
			expectedHealth: 10 * time.Second,
			expectedHB:     60 * time.Second,
			expectedReset:  5 * time.Minute,
		},
		{
			name: "custom config",
			config: Config{
				MinBackoff:          2 * time.Second,
				MaxBackoff:          10 * time.Minute,
				BackoffMultiplier:   3.0,
				MaxRetries:          5,
				HealthCheckInterval: 15 * time.Second,
				HeartbeatTimeout:    30 * time.Second,
				CircuitResetAfter:   10 * time.Minute,
			},
			expectedMin:    2 * time.Second,
			expectedMax:    10 * time.Minute,
			expectedMult:   3.0,
			expectedRetry:  5,
			expectedHealth: 15 * time.Second,
			expectedHB:     30 * time.Second,
			expectedReset:  10 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := newTestLogger()
			m := NewManager(tt.config, log)

			assert.Equal(t, tt.expectedMin, m.minBackoff)
			assert.Equal(t, tt.expectedMax, m.maxBackoff)
			assert.Equal(t, tt.expectedMult, m.backoffMultiplier)
			assert.Equal(t, tt.expectedRetry, m.maxRetries)
			assert.Equal(t, tt.expectedHealth, m.healthCheckInterval)
			assert.Equal(t, tt.expectedHB, m.heartbeatTimeout)
			assert.Equal(t, tt.expectedReset, m.circuitResetAfter)
			assert.Equal(t, tt.expectedMin, m.currentBackoff)
			assert.False(t, m.circuitOpen)
		})
	}
}

func TestRecordMessageReceived(t *testing.T) {
	log := newTestLogger()
	m := NewManager(Config{}, log)

	// Initially should be 0
	assert.Equal(t, int64(0), m.lastMessageTime.Load())

	// Record message
	before := time.Now().Unix()
	m.RecordMessageReceived()
	after := time.Now().Unix()

	lastMsg := m.lastMessageTime.Load()
	assert.GreaterOrEqual(t, lastMsg, before)
	assert.LessOrEqual(t, lastMsg, after)
}

func TestIsHealthy(t *testing.T) {
	tests := []struct {
		name                 string
		setupCircuitOpen     bool
		setupCircuitOpenedAt time.Time
		setupLastMessage     time.Time
		circuitResetAfter    time.Duration
		heartbeatTimeout     time.Duration
		expectedHealthy      bool
	}{
		{
			name:             "no messages yet - healthy",
			setupCircuitOpen: false,
			expectedHealthy:  true,
		},
		{
			name:             "recent message - healthy",
			setupCircuitOpen: false,
			setupLastMessage: time.Now().Add(-5 * time.Second),
			heartbeatTimeout: 60 * time.Second,
			expectedHealthy:  true,
		},
		{
			name:             "old message - unhealthy",
			setupCircuitOpen: false,
			setupLastMessage: time.Now().Add(-120 * time.Second),
			heartbeatTimeout: 60 * time.Second,
			expectedHealthy:  false,
		},
		{
			name:                 "circuit open recent - unhealthy",
			setupCircuitOpen:     true,
			setupCircuitOpenedAt: time.Now().Add(-1 * time.Minute),
			circuitResetAfter:    5 * time.Minute,
			heartbeatTimeout:     60 * time.Second,
			expectedHealthy:      false,
		},
		{
			name:                 "circuit open expired - unhealthy but can retry",
			setupCircuitOpen:     true,
			setupCircuitOpenedAt: time.Now().Add(-6 * time.Minute),
			circuitResetAfter:    5 * time.Minute,
			heartbeatTimeout:     60 * time.Second,
			expectedHealthy:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := newTestLogger()
			config := Config{
				CircuitResetAfter: tt.circuitResetAfter,
				HeartbeatTimeout:  tt.heartbeatTimeout,
			}
			m := NewManager(config, log)

			// Setup state
			m.circuitOpen = tt.setupCircuitOpen
			m.circuitOpenedAt = tt.setupCircuitOpenedAt
			if !tt.setupLastMessage.IsZero() {
				m.lastMessageTime.Store(tt.setupLastMessage.Unix())
			}

			healthy := m.IsHealthy()
			assert.Equal(t, tt.expectedHealthy, healthy)
		})
	}
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name                string
		maxRetries          int
		consecutiveFailures int
		circuitOpen         bool
		circuitOpenedAt     time.Time
		circuitResetAfter   time.Duration
		expectedShouldRetry bool
	}{
		{
			name:                "no failures - should retry",
			maxRetries:          10,
			consecutiveFailures: 0,
			circuitOpen:         false,
			expectedShouldRetry: true,
		},
		{
			name:                "some failures - should retry",
			maxRetries:          10,
			consecutiveFailures: 5,
			circuitOpen:         false,
			expectedShouldRetry: true,
		},
		{
			name:                "max retries reached - should not retry",
			maxRetries:          10,
			consecutiveFailures: 10,
			circuitOpen:         false,
			expectedShouldRetry: false,
		},
		{
			name:                "circuit open recent - should not retry",
			maxRetries:          10,
			consecutiveFailures: 5,
			circuitOpen:         true,
			circuitOpenedAt:     time.Now().Add(-1 * time.Minute),
			circuitResetAfter:   5 * time.Minute,
			expectedShouldRetry: false,
		},
		{
			name:                "circuit open expired - should retry",
			maxRetries:          10,
			consecutiveFailures: 5,
			circuitOpen:         true,
			circuitOpenedAt:     time.Now().Add(-6 * time.Minute),
			circuitResetAfter:   5 * time.Minute,
			expectedShouldRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := newTestLogger()
			config := Config{
				MaxRetries:        tt.maxRetries,
				CircuitResetAfter: tt.circuitResetAfter,
			}
			m := NewManager(config, log)

			// Setup state
			m.consecutiveFailures = tt.consecutiveFailures
			m.circuitOpen = tt.circuitOpen
			m.circuitOpenedAt = tt.circuitOpenedAt

			shouldRetry := m.ShouldRetry()
			assert.Equal(t, tt.expectedShouldRetry, shouldRetry)
		})
	}
}

func TestGetBackoff(t *testing.T) {
	log := newTestLogger()
	m := NewManager(Config{MinBackoff: 1 * time.Second}, log)

	// Initial backoff
	assert.Equal(t, 1*time.Second, m.GetBackoff())

	// After failure
	m.currentBackoff = 5 * time.Second
	assert.Equal(t, 5*time.Second, m.GetBackoff())
}

func TestRecordFailure(t *testing.T) {
	tests := []struct {
		name                        string
		minBackoff                  time.Duration
		maxBackoff                  time.Duration
		backoffMultiplier           float64
		maxRetries                  int
		initialConsecutiveFailures  int
		initialBackoff              time.Duration
		expectedConsecutiveFailures int
		expectedBackoff             time.Duration
		expectedCircuitOpen         bool
	}{
		{
			name:                        "first failure",
			minBackoff:                  1 * time.Second,
			maxBackoff:                  5 * time.Minute,
			backoffMultiplier:           2.0,
			maxRetries:                  10,
			initialConsecutiveFailures:  0,
			initialBackoff:              1 * time.Second,
			expectedConsecutiveFailures: 1,
			expectedBackoff:             2 * time.Second,
			expectedCircuitOpen:         false,
		},
		{
			name:                        "exponential backoff",
			minBackoff:                  1 * time.Second,
			maxBackoff:                  5 * time.Minute,
			backoffMultiplier:           2.0,
			maxRetries:                  10,
			initialConsecutiveFailures:  3,
			initialBackoff:              8 * time.Second,
			expectedConsecutiveFailures: 4,
			expectedBackoff:             16 * time.Second,
			expectedCircuitOpen:         false,
		},
		{
			name:                        "max backoff reached",
			minBackoff:                  1 * time.Second,
			maxBackoff:                  10 * time.Second,
			backoffMultiplier:           2.0,
			maxRetries:                  10,
			initialConsecutiveFailures:  0,
			initialBackoff:              8 * time.Second,
			expectedConsecutiveFailures: 1,
			expectedBackoff:             10 * time.Second,
			expectedCircuitOpen:         false,
		},
		{
			name:                        "circuit breaker opens",
			minBackoff:                  1 * time.Second,
			maxBackoff:                  5 * time.Minute,
			backoffMultiplier:           2.0,
			maxRetries:                  5,
			initialConsecutiveFailures:  4,
			initialBackoff:              16 * time.Second,
			expectedConsecutiveFailures: 5,
			expectedBackoff:             32 * time.Second,
			expectedCircuitOpen:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := newTestLogger()
			config := Config{
				MinBackoff:        tt.minBackoff,
				MaxBackoff:        tt.maxBackoff,
				BackoffMultiplier: tt.backoffMultiplier,
				MaxRetries:        tt.maxRetries,
			}
			m := NewManager(config, log)

			// Setup initial state
			m.consecutiveFailures = tt.initialConsecutiveFailures
			m.currentBackoff = tt.initialBackoff

			// Record failure
			m.RecordFailure()

			assert.Equal(t, tt.expectedConsecutiveFailures, m.consecutiveFailures)
			assert.Equal(t, tt.expectedBackoff, m.currentBackoff)
			assert.Equal(t, tt.expectedCircuitOpen, m.circuitOpen)

			if tt.expectedCircuitOpen {
				assert.False(t, m.circuitOpenedAt.IsZero())
			}
		})
	}
}

func TestRecordSuccess(t *testing.T) {
	tests := []struct {
		name                       string
		initialConsecutiveFailures int
		initialBackoff             time.Duration
		initialCircuitOpen         bool
		initialTotalReconnects     int
		minBackoff                 time.Duration
	}{
		{
			name:                       "success after failures",
			initialConsecutiveFailures: 5,
			initialBackoff:             32 * time.Second,
			initialCircuitOpen:         false,
			initialTotalReconnects:     2,
			minBackoff:                 1 * time.Second,
		},
		{
			name:                       "success closes circuit breaker",
			initialConsecutiveFailures: 10,
			initialBackoff:             5 * time.Minute,
			initialCircuitOpen:         true,
			initialTotalReconnects:     5,
			minBackoff:                 1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := newTestLogger()
			config := Config{MinBackoff: tt.minBackoff}
			m := NewManager(config, log)

			// Setup initial state
			m.consecutiveFailures = tt.initialConsecutiveFailures
			m.currentBackoff = tt.initialBackoff
			m.circuitOpen = tt.initialCircuitOpen
			if tt.initialCircuitOpen {
				m.circuitOpenedAt = time.Now().Add(-1 * time.Minute)
			}
			m.totalReconnects = tt.initialTotalReconnects

			// Record success
			m.RecordSuccess()

			// Verify state reset
			assert.Equal(t, 0, m.consecutiveFailures)
			assert.Equal(t, tt.minBackoff, m.currentBackoff)
			assert.Equal(t, tt.initialTotalReconnects+1, m.totalReconnects)
			assert.False(t, m.circuitOpen)
			assert.True(t, m.circuitOpenedAt.IsZero())

			// Verify last message time updated
			lastMsg := time.Unix(m.lastMessageTime.Load(), 0)
			assert.False(t, lastMsg.IsZero())
		})
	}
}

func TestResetCircuit(t *testing.T) {
	tests := []struct {
		name        string
		circuitOpen bool
		shouldReset bool
	}{
		{
			name:        "reset open circuit",
			circuitOpen: true,
			shouldReset: true,
		},
		{
			name:        "reset already closed circuit",
			circuitOpen: false,
			shouldReset: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := newTestLogger()
			m := NewManager(Config{}, log)

			// Setup
			m.circuitOpen = tt.circuitOpen
			if tt.circuitOpen {
				m.circuitOpenedAt = time.Now()
				m.consecutiveFailures = 10
				m.currentBackoff = 5 * time.Minute
			}

			// Reset
			m.ResetCircuit()

			// Verify
			assert.False(t, m.circuitOpen)
			if tt.shouldReset {
				assert.True(t, m.circuitOpenedAt.IsZero())
				assert.Equal(t, 0, m.consecutiveFailures)
				assert.Equal(t, m.minBackoff, m.currentBackoff)
			}
		})
	}
}

func TestGetStats(t *testing.T) {
	log := newTestLogger()
	m := NewManager(Config{}, log)

	// Setup state
	m.consecutiveFailures = 3
	m.totalReconnects = 5
	m.currentBackoff = 8 * time.Second
	m.circuitOpen = false
	m.lastMessageTime.Store(time.Now().Add(-10 * time.Second).Unix())

	stats := m.GetStats()

	assert.Equal(t, 3, stats.ConsecutiveFailures)
	assert.Equal(t, 5, stats.TotalReconnects)
	assert.Equal(t, 8*time.Second, stats.CurrentBackoff)
	assert.False(t, stats.CircuitOpen)
	assert.False(t, stats.LastMessageTime.IsZero())
	assert.True(t, stats.TimeSinceLastMessage >= 10*time.Second)
	assert.True(t, stats.IsHealthy) // Within heartbeat timeout
}

func TestReconnectWithBackoff(t *testing.T) {
	tests := []struct {
		name                     string
		setupCircuitOpen         bool
		setupMaxRetries          int
		setupConsecutiveFailures int
		reconnectFnError         error
		expectError              bool
		expectErrorContains      string
		expectSuccess            bool
	}{
		{
			name:             "successful reconnect",
			reconnectFnError: nil,
			expectError:      false,
			expectSuccess:    true,
		},
		{
			name:                "failed reconnect",
			reconnectFnError:    errors.New("connection refused"),
			expectError:         true,
			expectErrorContains: "reconnection failed",
			expectSuccess:       false,
		},
		{
			name:                     "circuit breaker open",
			setupCircuitOpen:         true,
			setupMaxRetries:          5,
			setupConsecutiveFailures: 5,
			expectError:              true,
			expectErrorContains:      "circuit breaker is open",
			expectSuccess:            false,
		},
		{
			name:                     "max retries reached",
			setupCircuitOpen:         false,
			setupMaxRetries:          5,
			setupConsecutiveFailures: 5,
			expectError:              true,
			expectErrorContains:      "max retries reached",
			expectSuccess:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := newTestLogger()
			config := Config{
				MinBackoff: 10 * time.Millisecond, // Short for testing
				MaxRetries: tt.setupMaxRetries,
			}
			m := NewManager(config, log)

			// Setup state
			m.circuitOpen = tt.setupCircuitOpen
			if tt.setupCircuitOpen {
				m.circuitOpenedAt = time.Now()
			}
			m.consecutiveFailures = tt.setupConsecutiveFailures

			// Reconnect function
			reconnectFn := func(ctx context.Context) error {
				return tt.reconnectFnError
			}

			ctx := context.Background()
			err := m.ReconnectWithBackoff(ctx, reconnectFn)

			if tt.expectError {
				require.Error(t, err)
				if tt.expectErrorContains != "" {
					assert.Contains(t, err.Error(), tt.expectErrorContains)
				}
			} else {
				require.NoError(t, err)
			}

			if tt.expectSuccess {
				assert.Equal(t, 0, m.consecutiveFailures)
				assert.False(t, m.circuitOpen)
			}
		})
	}
}

func TestReconnectWithBackoff_ContextCancellation(t *testing.T) {
	log := newTestLogger()
	config := Config{
		MinBackoff: 1 * time.Second, // Long backoff
	}
	m := NewManager(config, log)

	// Record a failure to set up backoff
	m.RecordFailure()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	reconnectFn := func(ctx context.Context) error {
		return nil // Should not reach here
	}

	err := m.ReconnectWithBackoff(ctx, reconnectFn)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestCalculateJitter(t *testing.T) {
	tests := []struct {
		name          string
		duration      time.Duration
		jitterPercent float64
		checkFunc     func(t *testing.T, result time.Duration)
	}{
		{
			name:          "no jitter",
			duration:      10 * time.Second,
			jitterPercent: 0,
			checkFunc: func(t *testing.T, result time.Duration) {
				assert.Equal(t, 10*time.Second, result)
			},
		},
		{
			name:          "invalid jitter percent",
			duration:      10 * time.Second,
			jitterPercent: 1.5,
			checkFunc: func(t *testing.T, result time.Duration) {
				assert.Equal(t, 10*time.Second, result)
			},
		},
		{
			name:          "with jitter",
			duration:      10 * time.Second,
			jitterPercent: 0.2,
			checkFunc: func(t *testing.T, result time.Duration) {
				// Result should be >= original duration
				assert.GreaterOrEqual(t, result, 10*time.Second)
				// Result should be <= original + max jitter (20%)
				assert.LessOrEqual(t, result, 12*time.Second)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateJitter(tt.duration, tt.jitterPercent)
			tt.checkFunc(t, result)
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	log := newTestLogger()
	m := NewManager(Config{}, log)

	// Simulate concurrent access
	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			m.RecordMessageReceived()
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			m.IsHealthy()
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			m.RecordFailure()
			time.Sleep(2 * time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			m.GetStats()
			time.Sleep(2 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}

	// Should not panic - test passes if we get here
	stats := m.GetStats()
	assert.NotNil(t, stats)
}
