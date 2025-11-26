package retry

import (
	"context"
	"math"
	"net"
	"net/http"
	"prometheus/pkg/errors"
	"time"
)

// Strategy defines the retry strategy
type Strategy string

const (
	// StrategyExponential uses exponential backoff
	StrategyExponential Strategy = "exponential"
	// StrategyLinear uses linear backoff
	StrategyLinear Strategy = "linear"
	// StrategyFixed uses fixed delay
	StrategyFixed Strategy = "fixed"
)

// Config contains retry configuration
type Config struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Strategy     Strategy
	Multiplier   float64 // For exponential backoff
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() Config {
	return Config{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Strategy:     StrategyExponential,
		Multiplier:   2.0,
	}
}

// Middleware provides retry functionality with exponential backoff
type Middleware struct {
	config Config
}

// New creates a new retry middleware
func New(config Config) *Middleware {
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = 100 * time.Millisecond
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 5 * time.Second
	}
	if config.Multiplier <= 0 {
		config.Multiplier = 2.0
	}
	if config.Strategy == "" {
		config.Strategy = StrategyExponential
	}

	return &Middleware{config: config}
}

// Do executes the function with retry logic
func (m *Middleware) Do(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= m.config.MaxRetries; attempt++ {
		// Execute function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return err
		}

		// Don't sleep after last attempt
		if attempt == m.config.MaxRetries {
			break
		}

		// Calculate backoff delay
		delay := m.calculateDelay(attempt)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "retry cancelled")
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return errors.Wrapf(lastErr, "max retries (%d) exceeded", m.config.MaxRetries)
}

// DoWithResult executes the function with retry logic and returns a result
func (m *Middleware) DoWithResult(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	var result interface{}
	var lastErr error

	for attempt := 0; attempt <= m.config.MaxRetries; attempt++ {
		// Execute function
		var err error
		result, err = fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return nil, err
		}

		// Don't sleep after last attempt
		if attempt == m.config.MaxRetries {
			break
		}

		// Calculate backoff delay
		delay := m.calculateDelay(attempt)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return nil, errors.Wrap(ctx.Err(), "retry cancelled")
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return nil, errors.Wrapf(lastErr, "max retries (%d) exceeded", m.config.MaxRetries)
}

// calculateDelay calculates the backoff delay based on the strategy
func (m *Middleware) calculateDelay(attempt int) time.Duration {
	var delay time.Duration

	switch m.config.Strategy {
	case StrategyExponential:
		// Exponential: delay = initial * (multiplier ^ attempt)
		delay = time.Duration(float64(m.config.InitialDelay) * math.Pow(m.config.Multiplier, float64(attempt)))

	case StrategyLinear:
		// Linear: delay = initial * (1 + attempt)
		delay = m.config.InitialDelay * time.Duration(1+attempt)

	case StrategyFixed:
		// Fixed: always use initial delay
		delay = m.config.InitialDelay

	default:
		delay = m.config.InitialDelay
	}

	// Cap at max delay
	if delay > m.config.MaxDelay {
		delay = m.config.MaxDelay
	}

	return delay
}

// isRetryableError determines if an error is worth retrying
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Network errors are generally retryable
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Temporary() || netErr.Timeout()
	}

	// HTTP status codes that are retryable
	var httpErr interface{ StatusCode() int }
	if errors.As(err, &httpErr) {
		code := httpErr.StatusCode()
		return code == http.StatusTooManyRequests ||
			code == http.StatusRequestTimeout ||
			code == http.StatusServiceUnavailable ||
			code == http.StatusGatewayTimeout ||
			code >= 500
	}

	// Context errors are not retryable
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Check for specific error messages
	errStr := err.Error()
	retryableMessages := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"no such host",
		"timeout",
		"temporary failure",
		"too many requests",
		"rate limit",
		"throttled",
	}

	for _, msg := range retryableMessages {
		if contains(errStr, msg) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
