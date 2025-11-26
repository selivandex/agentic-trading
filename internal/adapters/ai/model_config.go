package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"prometheus/pkg/errors"
)

// AgentModelConfig represents model selection for a specific agent type.
type AgentModelConfig struct {
	Agent    string
	Provider string
	Model    string
	Timeout  time.Duration
}

// ModelSelector handles model resolution per agent with graceful fallbacks.
type ModelSelector struct {
	registry *ProviderRegistry
	configs  map[string]AgentModelConfig
	mu       sync.RWMutex
}

// NewModelSelector constructs a selector with optional defaults.
func NewModelSelector(registry *ProviderRegistry, defaults []AgentModelConfig) *ModelSelector {
	configMap := make(map[string]AgentModelConfig)
	for _, cfg := range defaults {
		configMap[cfg.Agent] = cfg
	}

	return &ModelSelector{
		registry: registry,
		configs:  configMap,
	}
}

// Set assigns model configuration for an agent.
func (s *ModelSelector) Set(agent string, cfg AgentModelConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.configs[agent] = cfg
}

// Get returns model metadata for agent or default provider model if missing.
func (s *ModelSelector) Get(ctx context.Context, agent string, defaultProvider string) (AgentModelConfig, ModelInfo, error) {
	s.mu.RLock()
	cfg, ok := s.configs[agent]
	s.mu.RUnlock()

	if !ok {
		cfg = AgentModelConfig{Agent: agent, Provider: defaultProvider}
	}

	providerName := cfg.Provider
	if providerName == "" {
		providerName = defaultProvider
	}

	provider, err := s.registry.Get(providerName)
	if err != nil {
		return AgentModelConfig{}, ModelInfo{}, err
	}

	modelName := cfg.Model
	if modelName == "" {
		models, err := provider.ListModels(ctx)
		if err != nil {
			return AgentModelConfig{}, ModelInfo{}, errors.Wrapf(err, "failed to list models for provider %s", providerName)
		}
		if len(models) == 0 {
			return AgentModelConfig{}, ModelInfo{}, errors.Wrapf(errors.ErrUnavailable, "provider %s has no available models", providerName)
		}
		modelName = models[0].Name
	}

	info, err := provider.GetModel(ctx, modelName)
	if err != nil {
		return AgentModelConfig{}, ModelInfo{}, err
	}

	cfg.Provider = providerName
	cfg.Model = modelName
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout()
	}

	return cfg, info, nil
}

// ProviderUsage captures usage metrics for a provider.
type ProviderUsage struct {
	Model        string
	Provider     string
	InputTokens  int64
	OutputTokens int64
	CostUSD      float64
}

// UsageTracker tracks token and cost usage per provider/model.
type UsageTracker struct {
	mu    sync.Mutex
	usage map[string]*ProviderUsage
}

// NewUsageTracker creates a new tracker instance.
func NewUsageTracker() *UsageTracker {
	return &UsageTracker{usage: make(map[string]*ProviderUsage)}
}

// Record calculates cost based on model pricing and records the usage.
func (t *UsageTracker) Record(model ModelInfo, provider string, inputTokens int64, outputTokens int64) ProviderUsage {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := fmt.Sprintf("%s:%s", provider, model.Name)
	entry, ok := t.usage[key]
	if !ok {
		entry = &ProviderUsage{Model: model.Name, Provider: provider}
		t.usage[key] = entry
	}

	entry.InputTokens += inputTokens
	entry.OutputTokens += outputTokens
	entry.CostUSD += calculateCost(model, inputTokens, outputTokens)

	return *entry
}

// Snapshot returns a copy of the current usage map.
func (t *UsageTracker) Snapshot() map[string]ProviderUsage {
	t.mu.Lock()
	defer t.mu.Unlock()

	copyMap := make(map[string]ProviderUsage, len(t.usage))
	for k, v := range t.usage {
		copyMap[k] = *v
	}

	return copyMap
}

func calculateCost(model ModelInfo, inputTokens int64, outputTokens int64) float64 {
	return (float64(inputTokens)/1000.0)*model.InputCostPer1K + (float64(outputTokens)/1000.0)*model.OutputCostPer1K
}
