package ai

import (
	"strings"
	"time"

	"prometheus/internal/adapters/config"
	"prometheus/pkg/errors"
)

// BuildRegistry initializes a ProviderRegistry with all enabled providers based on configuration.
// redisClient is optional - if provided, distributed rate limiting will be used (required for multi-pod deployment).
// If nil, local in-memory rate limiting will be used (suitable for single-pod deployment).
func BuildRegistry(cfg config.AIConfig, redisClient interface{}) (*ProviderRegistry, error) {
	registry := NewProviderRegistry()

	// Create rate limiter factory
	limiterFactory := NewRateLimiterFactory(redisClient)

	// Register Claude provider
	if cfg.ClaudeKey != "" {
		rateLimitCfg := cfg.GetRateLimitConfig("claude")
		limiter := limiterFactory.Create(ProviderNameAnthropic, RateLimitConfig{
			Enabled:      rateLimitCfg.Enabled,
			ReqPerMinute: rateLimitCfg.ReqPerMinute,
			Burst:        rateLimitCfg.Burst,
		})
		if err := registry.Register(NewClaudeProvider(cfg.ClaudeKey, defaultTimeout(), limiter)); err != nil {
			return nil, err
		}
	}

	// Register OpenAI provider
	if cfg.OpenAIKey != "" {
		rateLimitCfg := cfg.GetRateLimitConfig("openai")
		limiter := limiterFactory.Create(ProviderNameOpenAI, RateLimitConfig{
			Enabled:      rateLimitCfg.Enabled,
			ReqPerMinute: rateLimitCfg.ReqPerMinute,
			Burst:        rateLimitCfg.Burst,
		})
		if err := registry.Register(NewOpenAIProvider(cfg.OpenAIKey, defaultTimeout(), limiter)); err != nil {
			return nil, err
		}
	}

	// Register DeepSeek provider
	if cfg.DeepSeekKey != "" {
		rateLimitCfg := cfg.GetRateLimitConfig("deepseek")
		limiter := limiterFactory.Create(ProviderNameDeepSeek, RateLimitConfig{
			Enabled:      rateLimitCfg.Enabled,
			ReqPerMinute: rateLimitCfg.ReqPerMinute,
			Burst:        rateLimitCfg.Burst,
		})
		if err := registry.Register(NewDeepSeekProvider(cfg.DeepSeekKey, defaultTimeout(), limiter)); err != nil {
			return nil, err
		}
	}

	// Register Gemini provider
	if cfg.GeminiKey != "" {
		rateLimitCfg := cfg.GetRateLimitConfig("gemini")
		limiter := limiterFactory.Create(ProviderNameGoogle, RateLimitConfig{
			Enabled:      rateLimitCfg.Enabled,
			ReqPerMinute: rateLimitCfg.ReqPerMinute,
			Burst:        rateLimitCfg.Burst,
		})
		if err := registry.Register(NewGeminiProvider(cfg.GeminiKey, defaultTimeout(), limiter)); err != nil {
			return nil, err
		}
	}

	if len(registry.List()) == 0 {
		return nil, errors.ErrUnavailable
	}

	return registry, nil
}

func defaultTimeout() time.Duration {
	return 60 * time.Second
}

// NormalizeProviderName makes provider lookup more forgiving.
func NormalizeProviderName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
