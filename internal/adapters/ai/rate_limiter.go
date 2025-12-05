package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"prometheus/pkg/errors"
)

// RateLimiter defines the interface for rate limiting AI provider requests.
type RateLimiter interface {
	// Wait blocks until request can proceed or context is cancelled.
	Wait(ctx context.Context) error

	// Allow checks if request can proceed without blocking.
	Allow() bool

	// Limit returns current rate limit (requests per minute).
	Limit() float64
}

// TokenBucketLimiter implements token bucket rate limiting algorithm.
// Thread-safe and efficient for high-concurrency scenarios.
type TokenBucketLimiter struct {
	rate       float64      // Requests per second
	burst      int          // Maximum burst size
	tokens     float64      // Current available tokens
	lastUpdate time.Time    // Last token refill time
	mu         sync.Mutex   // Protects tokens and lastUpdate
	provider   ProviderName // Provider name for logging
}

// NewTokenBucketLimiter creates a new token bucket rate limiter.
// reqPerMinute: maximum requests per minute (e.g., 500 for OpenAI Tier 1)
// burst: maximum burst size (typically 10-20% of rate)
func NewTokenBucketLimiter(provider ProviderName, reqPerMinute float64, burst int) *TokenBucketLimiter {
	if burst <= 0 {
		burst = int(reqPerMinute / 10) // Default: 10% of rate
		if burst < 1 {
			burst = 1
		}
	}

	return &TokenBucketLimiter{
		rate:       reqPerMinute / 60.0, // Convert to requests per second
		burst:      burst,
		tokens:     float64(burst), // Start with full bucket
		lastUpdate: time.Now(),
		provider:   provider,
	}
}

// Wait blocks until a token is available or context is cancelled.
func (l *TokenBucketLimiter) Wait(ctx context.Context) error {
	for {
		// Try to get token
		if l.Allow() {
			return nil
		}

		// Calculate wait time
		l.mu.Lock()
		waitTime := time.Duration(float64(time.Second) / l.rate)
		l.mu.Unlock()

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return errors.Wrapf(ctx.Err(), "rate limiter wait cancelled for provider %s", l.provider)
		case <-time.After(waitTime):
			// Continue loop to try again
		}
	}
}

// Allow checks if a request can proceed and consumes a token if available.
func (l *TokenBucketLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(l.lastUpdate).Seconds()
	l.tokens += elapsed * l.rate

	// Cap tokens at burst size
	if l.tokens > float64(l.burst) {
		l.tokens = float64(l.burst)
	}

	l.lastUpdate = now

	// Check if we have tokens available
	if l.tokens >= 1.0 {
		l.tokens -= 1.0
		return true
	}

	return false
}

// Limit returns the current rate limit in requests per minute.
func (l *TokenBucketLimiter) Limit() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.rate * 60.0 // Convert back to requests per minute
}

// NoOpLimiter is a rate limiter that never blocks (for testing or disabled rate limiting).
type NoOpLimiter struct{}

// NewNoOpLimiter creates a no-op rate limiter.
func NewNoOpLimiter() *NoOpLimiter {
	return &NoOpLimiter{}
}

// Wait always returns immediately without error.
func (l *NoOpLimiter) Wait(ctx context.Context) error {
	return nil
}

// Allow always returns true.
func (l *NoOpLimiter) Allow() bool {
	return true
}

// Limit returns -1 to indicate unlimited.
func (l *NoOpLimiter) Limit() float64 {
	return -1
}

// RateLimitConfig contains rate limit configuration for a provider.
type RateLimitConfig struct {
	Enabled      bool
	ReqPerMinute float64
	Burst        int
}

// DefaultRateLimits returns default rate limits for each provider based on their free/basic tiers.
// These are conservative limits to avoid hitting API rate limits.
func DefaultRateLimits() map[ProviderName]RateLimitConfig {
	return map[ProviderName]RateLimitConfig{
		ProviderNameAnthropic: {
			Enabled:      true,
			ReqPerMinute: 50, // Claude free tier: 50 req/min
			Burst:        10,
		},
		ProviderNameOpenAI: {
			Enabled:      true,
			ReqPerMinute: 500, // OpenAI Tier 1: 500 req/min
			Burst:        50,
		},
		ProviderNameDeepSeek: {
			Enabled:      false, // DeepSeek has no rate limits
			ReqPerMinute: 0,
			Burst:        0,
		},
		ProviderNameGoogle: {
			Enabled:      true,
			ReqPerMinute: 60, // Gemini free tier: 60 req/min
			Burst:        10,
		},
	}
}

// GetRateLimiter creates appropriate rate limiter based on config.
// Deprecated: Use NewRateLimiter instead for Redis support.
func GetRateLimiter(provider ProviderName, config RateLimitConfig) RateLimiter {
	if !config.Enabled || config.ReqPerMinute <= 0 {
		return NewNoOpLimiter()
	}
	return NewTokenBucketLimiter(provider, config.ReqPerMinute, config.Burst)
}

// RateLimiterFactory creates rate limiters with optional Redis support.
type RateLimiterFactory struct {
	redisClient interface{} // *redis.Client, but we don't import redis here to avoid circular deps
	useRedis    bool
}

// NewRateLimiterFactory creates a factory for rate limiters.
// If redisClient is nil, local in-memory limiters will be used (suitable for single-pod deployment).
// If redisClient is provided, distributed Redis-based limiters will be used (required for multi-pod deployment).
func NewRateLimiterFactory(redisClient interface{}) *RateLimiterFactory {
	return &RateLimiterFactory{
		redisClient: redisClient,
		useRedis:    redisClient != nil,
	}
}

// Create creates a rate limiter for the specified provider.
func (f *RateLimiterFactory) Create(provider ProviderName, config RateLimitConfig) RateLimiter {
	// Check if rate limiting is disabled
	if !config.Enabled || config.ReqPerMinute <= 0 {
		return NewNoOpLimiter()
	}

	// Use Redis-based limiter for distributed deployment
	if f.useRedis {
		return NewRedisRateLimiterFromClient(f.redisClient, provider, config.ReqPerMinute, config.Burst)
	}

	// Fall back to local in-memory limiter
	return NewTokenBucketLimiter(provider, config.ReqPerMinute, config.Burst)
}

// RateLimitError wraps rate limit related errors with provider context.
type RateLimitError struct {
	Provider ProviderName
	Limit    float64
	Err      error
}

// Error implements error interface.
func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit error for provider %s (limit: %.0f req/min): %v", e.Provider, e.Limit, e.Err)
}

// Unwrap returns the underlying error.
func (e *RateLimitError) Unwrap() error {
	return e.Err
}
