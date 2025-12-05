package ai

import (
	"context"
	"testing"
	"time"

	"prometheus/pkg/errors"
)

func TestTokenBucketLimiter_Basic(t *testing.T) {
	// Create limiter: 60 req/min = 1 req/sec, burst=2
	limiter := NewTokenBucketLimiter(ProviderNameOpenAI, 60, 2)

	ctx := context.Background()

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
}

func TestTokenBucketLimiter_Allow(t *testing.T) {
	// Create limiter: 60 req/min, burst=2
	limiter := NewTokenBucketLimiter(ProviderNameOpenAI, 60, 2)

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
}

func TestTokenBucketLimiter_ContextCancellation(t *testing.T) {
	// Create limiter with low rate
	limiter := NewTokenBucketLimiter(ProviderNameOpenAI, 6, 1) // 6 req/min = 0.1 req/sec

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

	// Check error is related to context
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context error, got: %v", err)
	}
}

func TestNoOpLimiter(t *testing.T) {
	limiter := NewNoOpLimiter()

	ctx := context.Background()

	// Should never block
	for i := 0; i < 1000; i++ {
		if err := limiter.Wait(ctx); err != nil {
			t.Fatalf("NoOpLimiter should never fail: %v", err)
		}
		if !limiter.Allow() {
			t.Fatal("NoOpLimiter should always allow")
		}
	}

	// Should return -1 for limit
	if limiter.Limit() != -1 {
		t.Errorf("Expected limit -1, got %f", limiter.Limit())
	}
}

func TestGetRateLimiter_Disabled(t *testing.T) {
	config := RateLimitConfig{
		Enabled:      false,
		ReqPerMinute: 100,
		Burst:        10,
	}

	limiter := GetRateLimiter(ProviderNameOpenAI, config)

	// Should be NoOpLimiter
	if _, ok := limiter.(*NoOpLimiter); !ok {
		t.Errorf("Expected NoOpLimiter when disabled, got %T", limiter)
	}
}

func TestGetRateLimiter_ZeroRate(t *testing.T) {
	config := RateLimitConfig{
		Enabled:      true,
		ReqPerMinute: 0,
		Burst:        10,
	}

	limiter := GetRateLimiter(ProviderNameOpenAI, config)

	// Should be NoOpLimiter
	if _, ok := limiter.(*NoOpLimiter); !ok {
		t.Errorf("Expected NoOpLimiter when rate is 0, got %T", limiter)
	}
}

func TestGetRateLimiter_Enabled(t *testing.T) {
	config := RateLimitConfig{
		Enabled:      true,
		ReqPerMinute: 100,
		Burst:        10,
	}

	limiter := GetRateLimiter(ProviderNameOpenAI, config)

	// Should be TokenBucketLimiter
	if _, ok := limiter.(*TokenBucketLimiter); !ok {
		t.Errorf("Expected TokenBucketLimiter when enabled, got %T", limiter)
	}

	// Check limit
	if limit := limiter.Limit(); limit != 100 {
		t.Errorf("Expected limit 100, got %f", limit)
	}
}

func TestRateLimiterFactory_NoRedis(t *testing.T) {
	factory := NewRateLimiterFactory(nil)

	config := RateLimitConfig{
		Enabled:      true,
		ReqPerMinute: 100,
		Burst:        10,
	}

	limiter := factory.Create(ProviderNameOpenAI, config)

	// Should create local TokenBucketLimiter
	if _, ok := limiter.(*TokenBucketLimiter); !ok {
		t.Errorf("Expected TokenBucketLimiter without Redis, got %T", limiter)
	}
}

func TestDefaultRateLimits(t *testing.T) {
	limits := DefaultRateLimits()

	// Check Claude
	claudeLimit, ok := limits[ProviderNameAnthropic]
	if !ok {
		t.Error("Claude limit not found")
	}
	if !claudeLimit.Enabled {
		t.Error("Claude should be enabled")
	}
	if claudeLimit.ReqPerMinute != 50 {
		t.Errorf("Expected Claude 50 req/min, got %f", claudeLimit.ReqPerMinute)
	}

	// Check DeepSeek (should be disabled)
	deepseekLimit, ok := limits[ProviderNameDeepSeek]
	if !ok {
		t.Error("DeepSeek limit not found")
	}
	if deepseekLimit.Enabled {
		t.Error("DeepSeek should be disabled by default (no limits)")
	}

	// Check OpenAI
	openaiLimit, ok := limits[ProviderNameOpenAI]
	if !ok {
		t.Error("OpenAI limit not found")
	}
	if openaiLimit.ReqPerMinute != 500 {
		t.Errorf("Expected OpenAI 500 req/min, got %f", openaiLimit.ReqPerMinute)
	}
}
