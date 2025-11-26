package ai

import (
	"context"
	"strings"
	"time"

	"prometheus/pkg/errors"
)

// OpenAIProvider implements OpenAI metadata.
type OpenAIProvider struct {
	apiKey  string
	timeout time.Duration
	models  []ModelInfo
}

// NewOpenAIProvider creates a new OpenAI provider instance.
func NewOpenAIProvider(apiKey string, timeout time.Duration) *OpenAIProvider {
	return &OpenAIProvider{apiKey: apiKey, timeout: timeout, models: openAIModels()}
}

// Name returns provider name.
func (p *OpenAIProvider) Name() string { return "openai" }

// GetModel returns model info by name.
func (p *OpenAIProvider) GetModel(_ context.Context, model string) (ModelInfo, error) {
	for _, m := range p.models {
		if strings.EqualFold(m.Name, model) {
			return m, nil
		}
	}
	return ModelInfo{}, errors.Wrapf(errors.ErrNotFound, "openai model %s not found", model)
}

// ListModels lists available models.
func (p *OpenAIProvider) ListModels(_ context.Context) ([]ModelInfo, error) {
	return p.models, nil
}

// SupportsStreaming indicates streaming support.
func (p *OpenAIProvider) SupportsStreaming() bool { return true }

// SupportsTools indicates tool calling support.
func (p *OpenAIProvider) SupportsTools() bool { return true }

func openAIModels() []ModelInfo {
	return []ModelInfo{
		{
			Provider:          ProviderNameOpenAI,
			Name:              "gpt-4o-mini",
			Family:            "gpt-4o",
			MaxTokens:         128000,
			InputCostPer1K:    0.00015,
			OutputCostPer1K:   0.0006,
			SupportsImages:    true,
			SupportsAudio:     true,
			SupportsTools:     true,
			SupportsStreaming: true,
		},
		{
			Provider:          ProviderNameOpenAI,
			Name:              "gpt-4o",
			Family:            "gpt-4o",
			MaxTokens:         128000,
			InputCostPer1K:    0.0025,
			OutputCostPer1K:   0.01,
			SupportsImages:    true,
			SupportsAudio:     true,
			SupportsTools:     true,
			SupportsStreaming: true,
		},
		{
			Provider:          ProviderNameOpenAI,
			Name:              "o1-mini",
			Family:            "o1",
			MaxTokens:         65536,
			InputCostPer1K:    0.008,
			OutputCostPer1K:   0.008,
			SupportsImages:    false,
			SupportsAudio:     false,
			SupportsTools:     true,
			SupportsStreaming: true,
		},
	}
}
