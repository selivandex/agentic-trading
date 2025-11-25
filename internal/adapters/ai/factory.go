package ai

import (
	"fmt"
	"strings"
	"time"

	"prometheus/internal/adapters/config"
)

// BuildRegistry initializes a ProviderRegistry with all enabled providers based on configuration.
func BuildRegistry(cfg config.AIConfig) (*ProviderRegistry, error) {
	registry := NewProviderRegistry()

	if cfg.ClaudeKey != "" {
		if err := registry.Register(NewClaudeProvider(cfg.ClaudeKey, defaultTimeout())); err != nil {
			return nil, err
		}
	}

	if cfg.OpenAIKey != "" {
		if err := registry.Register(NewOpenAIProvider(cfg.OpenAIKey, defaultTimeout())); err != nil {
			return nil, err
		}
	}

	if cfg.DeepSeekKey != "" {
		if err := registry.Register(NewDeepSeekProvider(cfg.DeepSeekKey, defaultTimeout())); err != nil {
			return nil, err
		}
	}

	if cfg.GeminiKey != "" {
		if err := registry.Register(NewGeminiProvider(cfg.GeminiKey, defaultTimeout())); err != nil {
			return nil, err
		}
	}

	if len(registry.List()) == 0 {
		return nil, fmt.Errorf("no AI providers registered: ensure API keys are set")
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
