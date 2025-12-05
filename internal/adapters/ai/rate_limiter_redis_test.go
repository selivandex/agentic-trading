package ai

import (
	"context"
	"sync"
	"testing"
	"time"

	"prometheus/internal/adapters/config"
	"prometheus/internal/testsupport"
)

func TestRedisRateLimiter_Basic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := config.RedisConfig{
		Host: "localhost",
		Port: 6379,
		DB:   1, // Use different DB for tests
	}

	redisClient := testsupport.NewRedisClient(t, cfg)
	ctx := context.Background()

	// Create limiter: 60 req/min = 1 req/sec, burst=2
	limiter := NewRedisRateLimiter(redisClient, ProviderNameOpenAI, 60, 2)

	// First request should pass immediately
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("First request should succeed: %v", err)
	}

	// Second request should pass immediately (burst)
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("Second request should succeed: %v", err)
	}

	// Third request should wait (bucket empty)
	start := time.Now()
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("Third request should eventually succeed: %v", err)
	}
	elapsed := time.Since(start)

	// Should have waited ~1 second (1 req/sec rate)
	if elapsed < 500*time.Millisecond {
		t.Errorf("Expected to wait ~1s, waited only %v", elapsed)
	}
	if elapsed > 2*time.Second {
		t.Errorf("Waited too long: %v", elapsed)
	}
}

func TestRedisRateLimiter_Allow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := config.RedisConfig{
		Host: "localhost",
		Port: 6379,
		DB:   1,
	}

	redisClient := testsupport.NewRedisClient(t, cfg)

	// Create limiter: 60 req/min, burst=2
	limiter := NewRedisRateLimiter(redisClient, ProviderNameOpenAI, 60, 2)

	// First two should be allowed (burst)
	if !limiter.Allow() {
		t.Error("First request should be allowed")
	}
	if !limiter.Allow() {
		t.Error("Second request should be allowed")
	}

	// Third should be denied (bucket empty)
	if limiter.Allow() {
		t.Error("Third request should be denied")
	}

	// Wait for refill
	time.Sleep(1100 * time.Millisecond)

	// Should be allowed again
	if !limiter.Allow() {
		t.Error("Request after refill should be allowed")
	}
}

func TestRedisRateLimiter_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := config.RedisConfig{
		Host: "localhost",
		Port: 6379,
		DB:   1,
	}

	redisClient := testsupport.NewRedisClient(t, cfg)

	// Create limiter with low rate
	limiter := NewRedisRateLimiter(redisClient, ProviderNameOpenAI, 6, 1) // 6 req/min = 0.1 req/sec

	// Consume the burst
	_ = limiter.Allow()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should fail due to context cancellation
	err := limiter.Wait(ctx)
	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	// Check it's a rate limit error
	rateLimitErr, ok := err.(*RateLimitError)
	if !ok {
		t.Fatalf("Expected RateLimitError, got %T: %v", err, err)
	}

	if rateLimitErr.Provider != ProviderNameOpenAI {
		t.Errorf("Expected provider OpenAI, got %s", rateLimitErr.Provider)
	}
}

func TestRedisRateLimiter_Distributed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := config.RedisConfig{
		Host: "localhost",
		Port: 6379,
		DB:   1,
	}

	redisClient := testsupport.NewRedisClient(t, cfg)
	ctx := context.Background()

	// Create two limiters for same provider (simulating two pods)
	limiter1 := NewRedisRateLimiter(redisClient, ProviderNameAnthropic, 60, 2)
	limiter2 := NewRedisRateLimiter(redisClient, ProviderNameAnthropic, 60, 2)

	// Both should share the same bucket
	// Consume burst from limiter1
	if err := limiter1.Wait(ctx); err != nil {
		t.Fatalf("First request from limiter1 should succeed: %v", err)
	}
	if err := limiter1.Wait(ctx); err != nil {
		t.Fatalf("Second request from limiter1 should succeed: %v", err)
	}

	// limiter2 should see empty bucket (shared state)
	start := time.Now()
	if err := limiter2.Wait(ctx); err != nil {
		t.Fatalf("Request from limiter2 should eventually succeed: %v", err)
	}
	elapsed := time.Since(start)

	// Should have waited because bucket was exhausted by limiter1
	if elapsed < 500*time.Millisecond {
		t.Errorf("Expected limiter2 to wait due to shared state, but it didn't wait long enough: %v", elapsed)
	}
}

func TestRedisRateLimiter_Concurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := config.RedisConfig{
		Host: "localhost",
		Port: 6379,
		DB:   1,
	}

	redisClient := testsupport.NewRedisClient(t, cfg)
	ctx := context.Background()

	// Create limiter: 600 req/min = 10 req/sec, burst=20
	limiter := NewRedisRateLimiter(redisClient, ProviderNameOpenAI, 600, 20)

	// Run 30 concurrent requests
	concurrency := 30
	var wg sync.WaitGroup
	errors := make(chan error, concurrency)
	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := limiter.Wait(ctx); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)
	elapsed := time.Since(start)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent request failed: %v", err)
	}

	// First 20 should be instant (burst), remaining 10 should take ~1s (at 10 req/sec)
	// So total should be around 1 second
	if elapsed < 800*time.Millisecond {
		t.Logf("Warning: Completed faster than expected: %v (might be ok due to timing)", elapsed)
	}
	if elapsed > 3*time.Second {
		t.Errorf("Took too long for concurrent requests: %v", elapsed)
	}
}

func TestRedisRateLimiter_GetStats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := config.RedisConfig{
		Host: "localhost",
		Port: 6379,
		DB:   1,
	}

	redisClient := testsupport.NewRedisClient(t, cfg)
	ctx := context.Background()

	limiter := NewRedisRateLimiter(redisClient, ProviderNameOpenAI, 60, 5)

	// Consume some tokens
	_ = limiter.Allow()
	_ = limiter.Allow()

	// Get stats
	tokens, lastUpdate, err := limiter.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	// Should have 3 tokens left (started with 5, consumed 2)
	if tokens < 2.5 || tokens > 3.5 {
		t.Errorf("Expected ~3 tokens, got %f", tokens)
	}

	// Last update should be recent
	if time.Since(lastUpdate) > 5*time.Second {
		t.Errorf("Last update too old: %v", lastUpdate)
	}
}

func TestRedisRateLimiter_Reset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := config.RedisConfig{
		Host: "localhost",
		Port: 6379,
		DB:   1,
	}

	redisClient := testsupport.NewRedisClient(t, cfg)
	ctx := context.Background()

	limiter := NewRedisRateLimiter(redisClient, ProviderNameOpenAI, 60, 2)

	// Exhaust bucket
	_ = limiter.Allow()
	_ = limiter.Allow()

	// Should be denied now
	if limiter.Allow() {
		t.Error("Third request should be denied")
	}

	// Reset
	if err := limiter.Reset(ctx); err != nil {
		t.Fatalf("Failed to reset limiter: %v", err)
	}

	// Should be allowed again (bucket refilled)
	if !limiter.Allow() {
		t.Error("Request after reset should be allowed")
	}
}

func TestRedisRateLimiter_LuaScriptAtomic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := config.RedisConfig{
		Host: "localhost",
		Port: 6379,
		DB:   1,
	}

	redisClient := testsupport.NewRedisClient(t, cfg)

	// Create limiter with burst=1 to test atomicity
	limiter := NewRedisRateLimiter(redisClient, ProviderNameDeepSeek, 60, 1)

	// Run many concurrent Allow() calls
	// Only ONE should succeed due to atomic Lua script
	concurrency := 100
	var wg sync.WaitGroup
	successCount := int32(0)
	var mu sync.Mutex

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if limiter.Allow() {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Only 1 should have succeeded (burst=1 and all ran concurrently)
	if successCount != 1 {
		t.Errorf("Expected exactly 1 success (atomicity test), got %d", successCount)
	}
}

func TestRateLimiterFactory_WithRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := config.RedisConfig{
		Host: "localhost",
		Port: 6379,
		DB:   1,
	}

	redisClient := testsupport.NewRedisClient(t, cfg)

	factory := NewRateLimiterFactory(redisClient)

	config := RateLimitConfig{
		Enabled:      true,
		ReqPerMinute: 100,
		Burst:        10,
	}

	limiter := factory.Create(ProviderNameOpenAI, config)

	// Should create RedisRateLimiter
	if _, ok := limiter.(*RedisRateLimiter); !ok {
		t.Errorf("Expected RedisRateLimiter with Redis client, got %T", limiter)
	}

	// Should work
	if !limiter.Allow() {
		t.Error("First request should be allowed")
	}
}

func TestRateLimiterFactory_DisabledWithRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := config.RedisConfig{
		Host: "localhost",
		Port: 6379,
		DB:   1,
	}

	redisClient := testsupport.NewRedisClient(t, cfg)

	factory := NewRateLimiterFactory(redisClient)

	config := RateLimitConfig{
		Enabled:      false,
		ReqPerMinute: 100,
		Burst:        10,
	}

	limiter := factory.Create(ProviderNameOpenAI, config)

	// Should create NoOpLimiter even with Redis available
	if _, ok := limiter.(*NoOpLimiter); !ok {
		t.Errorf("Expected NoOpLimiter when disabled, got %T", limiter)
	}
}
