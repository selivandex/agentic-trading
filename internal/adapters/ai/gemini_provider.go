package ai

import (
	"context"
	"strings"
	"time"

	"prometheus/pkg/errors"
)

// GeminiProvider implements Google Gemini metadata.
type GeminiProvider struct {
	apiKey  string
	timeout time.Duration
	models  []ModelInfo
}

// NewGeminiProvider creates a new Gemini provider.
func NewGeminiProvider(apiKey string, timeout time.Duration) *GeminiProvider {
	return &GeminiProvider{apiKey: apiKey, timeout: timeout, models: geminiModels()}
}

// Name returns provider name.
func (p *GeminiProvider) Name() string { return "gemini" }

// GetModel returns model info by name.
func (p *GeminiProvider) GetModel(_ context.Context, model string) (ModelInfo, error) {
	for _, m := range p.models {
		if strings.EqualFold(m.Name, model) {
			return m, nil
		}
	}
	return ModelInfo{}, errors.Wrapf(errors.ErrNotFound, "gemini model %s not found", model)
}

// ListModels lists available models.
func (p *GeminiProvider) ListModels(_ context.Context) ([]ModelInfo, error) {
	return p.models, nil
}

// SupportsStreaming indicates streaming support.
func (p *GeminiProvider) SupportsStreaming() bool { return true }

// SupportsTools indicates tool calling support.
func (p *GeminiProvider) SupportsTools() bool { return true }

func geminiModels() []ModelInfo {
	return []ModelInfo{
		{
			Provider:          ProviderNameGoogle,
			Name:              "gemini-1.5-flash",
			Family:            "gemini-1.5",
			MaxTokens:         1000000,
			InputCostPer1K:    0.0002,
			OutputCostPer1K:   0.0004,
			SupportsImages:    true,
			SupportsAudio:     true,
			SupportsTools:     true,
			SupportsStreaming: true,
		},
		{
			Provider:          ProviderNameGoogle,
			Name:              "gemini-1.5-pro",
			Family:            "gemini-1.5",
			MaxTokens:         2000000,
			InputCostPer1K:    0.0035,
			OutputCostPer1K:   0.0105,
			SupportsImages:    true,
			SupportsAudio:     true,
			SupportsTools:     true,
			SupportsStreaming: true,
		},
	}
}
