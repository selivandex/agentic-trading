package embeddings

import (
	"fmt"
	"time"

	"prometheus/pkg/errors"
)

// ProviderType defines supported embedding providers
type ProviderType string

const (
	ProviderOpenAI ProviderType = "openai"
	// Future providers:
	// ProviderCohere   ProviderType = "cohere"
	// ProviderLocal    ProviderType = "local"    // ollama/llama.cpp
	// ProviderVertexAI ProviderType = "vertexai"
)

// Config holds configuration for embedding provider
type Config struct {
	Provider ProviderType
	APIKey   string
	Model    string
	Timeout  time.Duration
}

// NewProvider creates an embedding provider based on config
func NewProvider(cfg Config) (Provider, error) {
	switch cfg.Provider {
	case ProviderOpenAI:
		return NewOpenAIProvider(cfg.APIKey, cfg.Model, cfg.Timeout)

	// Future providers can be added here:
	// case ProviderCohere:
	//     return NewCohereProvider(cfg.APIKey, cfg.Model, cfg.Timeout)
	// case ProviderLocal:
	//     return NewLocalProvider(cfg.Model, cfg.Timeout)

	default:
		return nil, errors.Wrapf(errors.ErrInvalidInput,
			"unsupported embedding provider: %s", cfg.Provider)
	}
}

// MustNewProvider creates a provider or panics
func MustNewProvider(cfg Config) Provider {
	provider, err := NewProvider(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create embedding provider: %v", err))
	}
	return provider
}
