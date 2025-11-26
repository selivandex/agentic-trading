package ratelimit

import (
	"context"
	"prometheus/pkg/errors"
	"sync"

	"golang.org/x/time/rate"
)

// Limiter provides rate limiting functionality for exchange API calls
type Limiter struct {
	limiter *rate.Limiter
	name    string
}

// NewLimiter creates a new rate limiter
// requestsPerMinute: maximum number of requests allowed per minute
func NewLimiter(name string, requestsPerMinute int) *Limiter {
	// Convert to requests per second
	rps := float64(requestsPerMinute) / 60.0

	// Allow burst of 10% of per-minute limit
	burst := requestsPerMinute / 10
	if burst < 1 {
		burst = 1
	}

	return &Limiter{
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
		name:    name,
	}
}

// Wait blocks until the rate limiter allows the request
func (l *Limiter) Wait(ctx context.Context) error {
	if err := l.limiter.Wait(ctx); err != nil {
		return errors.Wrapf(err, "rate limiter %s", l.name)
	}
	return nil
}

// Allow checks if a request is allowed without blocking
func (l *Limiter) Allow() bool {
	return l.limiter.Allow()
}

// Reserve reserves a token for future use
func (l *Limiter) Reserve() *rate.Reservation {
	return l.limiter.Reserve()
}

// MultiLimiter manages multiple rate limiters (per endpoint, global, etc.)
type MultiLimiter struct {
	limiters map[string]*Limiter
	mu       sync.RWMutex
}

// NewMultiLimiter creates a new multi-limiter
func NewMultiLimiter() *MultiLimiter {
	return &MultiLimiter{
		limiters: make(map[string]*Limiter),
	}
}

// AddLimiter adds a rate limiter for a specific key
func (m *MultiLimiter) AddLimiter(key string, limiter *Limiter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.limiters[key] = limiter
}

// Wait waits for all specified limiters
func (m *MultiLimiter) Wait(ctx context.Context, keys ...string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, key := range keys {
		if limiter, ok := m.limiters[key]; ok {
			if err := limiter.Wait(ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// ExchangeLimiters contains rate limiters for different exchanges
type ExchangeLimiters struct {
	Binance *MultiLimiter
	Bybit   *MultiLimiter
	OKX     *MultiLimiter
}

// NewExchangeLimiters creates rate limiters for all exchanges
func NewExchangeLimiters() *ExchangeLimiters {
	el := &ExchangeLimiters{
		Binance: NewMultiLimiter(),
		Bybit:   NewMultiLimiter(),
		OKX:     NewMultiLimiter(),
	}

	// Binance rate limits
	// https://binance-docs.github.io/apidocs/spot/en/#limits
	el.Binance.AddLimiter("global", NewLimiter("binance-global", 1200)) // 1200 req/min
	el.Binance.AddLimiter("order", NewLimiter("binance-order", 100))    // 100 orders/10s = 600/min

	// Bybit rate limits
	// https://bybit-exchange.github.io/docs/v5/rate-limit
	el.Bybit.AddLimiter("global", NewLimiter("bybit-global", 120))   // 120 req/min
	el.Bybit.AddLimiter("trading", NewLimiter("bybit-trading", 100)) // 100 orders/min

	// OKX rate limits
	// https://www.okx.com/docs-v5/en/#overview-rate-limit
	el.OKX.AddLimiter("global", NewLimiter("okx-global", 60))   // 60 req/min
	el.OKX.AddLimiter("trading", NewLimiter("okx-trading", 60)) // 60 orders/min

	return el
}
