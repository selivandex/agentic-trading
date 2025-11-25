package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// DeepSeekProvider implements DeepSeek metadata.
type DeepSeekProvider struct {
	apiKey  string
	timeout time.Duration
	models  []ModelInfo
}

// NewDeepSeekProvider creates a new DeepSeek provider.
func NewDeepSeekProvider(apiKey string, timeout time.Duration) *DeepSeekProvider {
	return &DeepSeekProvider{apiKey: apiKey, timeout: timeout, models: deepSeekModels()}
}

// Name returns provider name.
func (p *DeepSeekProvider) Name() string { return "deepseek" }

// GetModel returns model info by name.
func (p *DeepSeekProvider) GetModel(_ context.Context, model string) (ModelInfo, error) {
	for _, m := range p.models {
		if strings.EqualFold(m.Name, model) {
			return m, nil
		}
	}
	return ModelInfo{}, fmt.Errorf("deepseek model %s not found", model)
}

// ListModels lists available models.
func (p *DeepSeekProvider) ListModels(_ context.Context) ([]ModelInfo, error) {
	return p.models, nil
}

// SupportsStreaming indicates streaming support.
func (p *DeepSeekProvider) SupportsStreaming() bool { return true }

// SupportsTools indicates tool calling support.
func (p *DeepSeekProvider) SupportsTools() bool { return true }

func deepSeekModels() []ModelInfo {
	return []ModelInfo{
		{
			Name:              "deepseek-reasoner",
			Family:            "deepseek",
			MaxTokens:         64000,
			InputCostPer1K:    0.00014,
			OutputCostPer1K:   0.00028,
			SupportsImages:    false,
			SupportsAudio:     false,
			SupportsTools:     true,
			SupportsStreaming: true,
		},
		{
			Name:              "deepseek-chat",
			Family:            "deepseek",
			MaxTokens:         64000,
			InputCostPer1K:    0.00007,
			OutputCostPer1K:   0.00014,
			SupportsImages:    false,
			SupportsAudio:     false,
			SupportsTools:     true,
			SupportsStreaming: true,
		},
	}
}
