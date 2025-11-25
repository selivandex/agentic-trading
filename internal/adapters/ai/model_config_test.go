package ai

import (
	"context"
	"testing"
	"time"
)

type selectorProvider struct {
	name   string
	models []ModelInfo
}

func (s *selectorProvider) Name() string { return s.name }
func (s *selectorProvider) GetModel(_ context.Context, model string) (ModelInfo, error) {
	for _, m := range s.models {
		if m.Name == model {
			return m, nil
		}
	}

	return ModelInfo{}, errNotFound(model)
}
func (s *selectorProvider) ListModels(_ context.Context) ([]ModelInfo, error) { return s.models, nil }
func (s *selectorProvider) SupportsStreaming() bool                           { return true }
func (s *selectorProvider) SupportsTools() bool                               { return true }

func TestModelSelectorUsesConfiguredValues(t *testing.T) {
	registry := NewProviderRegistry()
	provider := &selectorProvider{name: "mock", models: []ModelInfo{{Name: "chosen", MaxTokens: 1000}}}

	if err := registry.Register(provider); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	selector := NewModelSelector(registry, []AgentModelConfig{{
		Agent:    "agent-1",
		Provider: "mock",
		Model:    "chosen",
		Timeout:  5 * time.Second,
	}})

	cfg, info, err := selector.Get(context.Background(), "agent-1", "mock")
	if err != nil {
		t.Fatalf("selector failed: %v", err)
	}

	if cfg.Model != "chosen" || cfg.Provider != "mock" {
		t.Fatalf("unexpected config %+v", cfg)
	}

	if info.Name != "chosen" || info.MaxTokens != 1000 {
		t.Fatalf("unexpected info %+v", info)
	}

	if cfg.Timeout != 5*time.Second {
		t.Fatalf("expected timeout to be preserved, got %v", cfg.Timeout)
	}
}

func TestModelSelectorFallsBackToDefaultModelAndTimeout(t *testing.T) {
	registry := NewProviderRegistry()
	provider := &selectorProvider{name: "mock", models: []ModelInfo{{Name: "alpha"}, {Name: "beta"}}}
	if err := registry.Register(provider); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	selector := NewModelSelector(registry, nil)
	cfg, info, err := selector.Get(context.Background(), "agent-missing", "mock")
	if err != nil {
		t.Fatalf("selector failed: %v", err)
	}

	if cfg.Model != "alpha" || info.Name != "alpha" {
		t.Fatalf("expected fallback model alpha, got cfg=%s info=%s", cfg.Model, info.Name)
	}

	if cfg.Timeout != defaultTimeout() {
		t.Fatalf("expected default timeout %v, got %v", defaultTimeout(), cfg.Timeout)
	}
}

func TestModelSelectorErrorsOnMissingProvider(t *testing.T) {
	registry := NewProviderRegistry()
	selector := NewModelSelector(registry, nil)

	if _, _, err := selector.Get(context.Background(), "agent", "unknown"); err == nil {
		t.Fatal("expected error for missing provider")
	}
}

func TestModelSelectorErrorsOnMissingModel(t *testing.T) {
	registry := NewProviderRegistry()
	provider := &selectorProvider{name: "mock", models: []ModelInfo{{Name: "available"}}}
	if err := registry.Register(provider); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	selector := NewModelSelector(registry, []AgentModelConfig{{Agent: "agent", Provider: "mock", Model: "missing"}})
	if _, _, err := selector.Get(context.Background(), "agent", "mock"); err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestUsageTrackerAccumulatesUsage(t *testing.T) {
	tracker := NewUsageTracker()
	model := ModelInfo{Name: "test", InputCostPer1K: 0.002, OutputCostPer1K: 0.004}

	first := tracker.Record(model, "provider", 500, 500)
	if first.CostUSD <= 0 {
		t.Fatalf("expected positive cost, got %f", first.CostUSD)
	}

	second := tracker.Record(model, "provider", 500, 500)
	if second.InputTokens != 1000 || second.OutputTokens != 1000 {
		t.Fatalf("expected accumulated tokens, got %+v", second)
	}

	snapshot := tracker.Snapshot()
	if len(snapshot) != 1 {
		t.Fatalf("expected single snapshot entry, got %d", len(snapshot))
	}
}
