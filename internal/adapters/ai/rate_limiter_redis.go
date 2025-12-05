package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"prometheus/pkg/errors"
)

// RedisRateLimiter implements distributed token bucket rate limiting via Redis.
// Thread-safe across multiple pods/instances.
type RedisRateLimiter struct {
	client      *redis.Client
	provider    ProviderName
	rate        float64 // Requests per second
	burst       int     // Maximum burst size
	keyPrefix   string  // Redis key prefix
	tokenScript *redis.Script
	allowScript *redis.Script
}

// Lua script for token bucket algorithm (atomic operation)
// KEYS[1] = token bucket key
// ARGV[1] = rate (tokens per second)
// ARGV[2] = burst (max tokens)
// ARGV[3] = current timestamp
// Returns: 1 if allowed, 0 if denied
const luaTokenBucketScript = `
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Get current token count and last update time
local data = redis.call('HMGET', key, 'tokens', 'last_update')
local tokens = tonumber(data[1])
local last_update = tonumber(data[2])

-- Initialize if not exists
if not tokens then
    tokens = burst
    last_update = now
end

-- Refill tokens based on elapsed time
local elapsed = now - last_update
tokens = math.min(burst, tokens + elapsed * rate)

-- Check if we can consume a token
if tokens >= 1.0 then
    tokens = tokens - 1.0

    -- Save updated state
    redis.call('HMSET', key, 'tokens', tokens, 'last_update', now)
    redis.call('EXPIRE', key, 3600) -- Expire after 1 hour of inactivity

    return 1
else
    -- Save state without consuming
    redis.call('HMSET', key, 'tokens', tokens, 'last_update', now)
    redis.call('EXPIRE', key, 3600)

    return 0
end
`

// NewRedisRateLimiter creates a new Redis-based rate limiter.
func NewRedisRateLimiter(
	client *redis.Client,
	provider ProviderName,
	reqPerMinute float64,
	burst int,
) *RedisRateLimiter {
	if burst <= 0 {
		burst = int(reqPerMinute / 10) // Default: 10% of rate
		if burst < 1 {
			burst = 1
		}
	}

	return &RedisRateLimiter{
		client:      client,
		provider:    provider,
		rate:        reqPerMinute / 60.0, // Convert to requests per second
		burst:       burst,
		keyPrefix:   fmt.Sprintf("rate_limit:ai:%s", provider),
		tokenScript: redis.NewScript(luaTokenBucketScript),
	}
}

// NewRedisRateLimiterFromClient creates a Redis rate limiter from interface{} client.
// This is used by RateLimiterFactory to avoid circular dependencies.
// If client is not *redis.Client, returns a NoOpLimiter.
func NewRedisRateLimiterFromClient(
	clientInterface interface{},
	provider ProviderName,
	reqPerMinute float64,
	burst int,
) RateLimiter {
	client, ok := clientInterface.(*redis.Client)
	if !ok {
		// Fallback to no-op if wrong type
		return NewNoOpLimiter()
	}

	return NewRedisRateLimiter(client, provider, reqPerMinute, burst)
}

// Wait blocks until a token is available or context is cancelled.
func (l *RedisRateLimiter) Wait(ctx context.Context) error {
	for {
		// Try to get token
		allowed, err := l.tryAcquire(ctx)
		if err != nil {
			return errors.Wrapf(err, "redis rate limiter error for provider %s", l.provider)
		}

		if allowed {
			return nil
		}

		// Calculate wait time based on rate
		waitTime := time.Duration(float64(time.Second) / l.rate)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return &RateLimitError{
				Provider: l.provider,
				Limit:    l.Limit(),
				Err:      errors.Wrap(ctx.Err(), "rate limiter wait cancelled"),
			}
		case <-time.After(waitTime):
			// Continue loop to try again
		}
	}
}

// Allow checks if a request can proceed without blocking.
func (l *RedisRateLimiter) Allow() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	allowed, err := l.tryAcquire(ctx)
	if err != nil {
		// On error, be conservative and deny
		return false
	}

	return allowed
}

// Limit returns the current rate limit in requests per minute.
func (l *RedisRateLimiter) Limit() float64 {
	return l.rate * 60.0 // Convert back to requests per minute
}

// tryAcquire attempts to acquire a token using Redis Lua script.
func (l *RedisRateLimiter) tryAcquire(ctx context.Context) (bool, error) {
	now := float64(time.Now().UnixNano()) / float64(time.Second)

	result, err := l.tokenScript.Run(
		ctx,
		l.client,
		[]string{l.keyPrefix},
		l.rate,
		l.burst,
		now,
	).Int()

	if err != nil {
		return false, errors.Wrap(err, "failed to execute token bucket script")
	}

	return result == 1, nil
}

// Reset clears the rate limiter state (useful for testing).
func (l *RedisRateLimiter) Reset(ctx context.Context) error {
	return l.client.Del(ctx, l.keyPrefix).Err()
}

// GetStats returns current rate limiter stats (for monitoring).
func (l *RedisRateLimiter) GetStats(ctx context.Context) (tokens float64, lastUpdate time.Time, err error) {
	data, err := l.client.HMGet(ctx, l.keyPrefix, "tokens", "last_update").Result()
	if err != nil {
		return 0, time.Time{}, errors.Wrap(err, "failed to get rate limiter stats")
	}

	if data[0] != nil {
		fmt.Sscanf(data[0].(string), "%f", &tokens)
	} else {
		tokens = float64(l.burst) // Not initialized yet
	}

	if data[1] != nil {
		var timestamp float64
		fmt.Sscanf(data[1].(string), "%f", &timestamp)
		lastUpdate = time.Unix(0, int64(timestamp*float64(time.Second)))
	} else {
		lastUpdate = time.Now()
	}

	return tokens, lastUpdate, nil
}
