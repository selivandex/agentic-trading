package ai

import (
	"context"
	"strings"
	"time"

	"prometheus/pkg/errors"
)

// ClaudeProvider implements the Anthropic Claude integration metadata.
type ClaudeProvider struct {
	apiKey  string
	timeout time.Duration
	models  []ModelInfo
}

// NewClaudeProvider creates a new Claude provider.
func NewClaudeProvider(apiKey string, timeout time.Duration) *ClaudeProvider {
	return &ClaudeProvider{apiKey: apiKey, timeout: timeout, models: claudeModels()}
}

// Name returns provider name.
func (p *ClaudeProvider) Name() string {
	return "claude"
}

// GetModel returns model info by name.
func (p *ClaudeProvider) GetModel(_ context.Context, model string) (ModelInfo, error) {
	for _, m := range p.models {
		if strings.EqualFold(m.Name, model) {
			return m, nil
		}
	}
	return ModelInfo{}, errors.Wrapf(errors.ErrNotFound, "claude model %s not found", model)
}

// ListModels lists available models.
func (p *ClaudeProvider) ListModels(_ context.Context) ([]ModelInfo, error) {
	return p.models, nil
}

// SupportsStreaming indicates streaming support.
func (p *ClaudeProvider) SupportsStreaming() bool { return true }

// SupportsTools indicates tool calling support.
func (p *ClaudeProvider) SupportsTools() bool { return true }

func claudeModels() []ModelInfo {
	return []ModelInfo{
		{
			Name:              "claude-3-5-sonnet-latest",
			Family:            "claude-3.5",
			MaxTokens:         200000,
			InputCostPer1K:    0.003,
			OutputCostPer1K:   0.015,
			SupportsImages:    true,
			SupportsAudio:     true,
			SupportsTools:     true,
			SupportsStreaming: true,
		},
		{
			Name:              "claude-3-5-haiku-latest",
			Family:            "claude-3.5",
			MaxTokens:         200000,
			InputCostPer1K:    0.001,
			OutputCostPer1K:   0.005,
			SupportsImages:    true,
			SupportsAudio:     false,
			SupportsTools:     true,
			SupportsStreaming: true,
		},
		{
			Name:              "claude-3-opus",
			Family:            "claude-3",
			MaxTokens:         200000,
			InputCostPer1K:    0.015,
			OutputCostPer1K:   0.075,
			SupportsImages:    true,
			SupportsAudio:     false,
			SupportsTools:     true,
			SupportsStreaming: true,
		},
	}
}
