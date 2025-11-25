package ai

import (
	"context"
	"fmt"
	"testing"
)

func TestUsageTrackerCalculatesCost(t *testing.T) {
	tracker := NewUsageTracker()
	model := ModelInfo{InputCostPer1K: 0.002, OutputCostPer1K: 0.004, Name: "test-model"}

	usage := tracker.Record(model, "test", 500, 1500)

	if usage.CostUSD != 0.002*0.5+0.004*1.5 {
		t.Fatalf("unexpected cost: %f", usage.CostUSD)
	}

	snapshot := tracker.Snapshot()
	if len(snapshot) != 1 {
		t.Fatalf("expected snapshot size 1, got %d", len(snapshot))
	}
}

func TestModelSelectorFallsBackToFirstModel(t *testing.T) {
	registry := NewProviderRegistry()
	mock := &mockProvider{models: []ModelInfo{{Name: "alpha"}}}
	if err := registry.Register(mock); err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	selector := NewModelSelector(registry, nil)
	cfg, info, err := selector.Get(context.Background(), "agent-test", "mock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Model != "alpha" || info.Name != "alpha" {
		t.Fatalf("expected model alpha, got cfg=%s info=%s", cfg.Model, info.Name)
	}
}

type mockProvider struct {
	models []ModelInfo
}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) GetModel(_ context.Context, model string) (ModelInfo, error) {
	for _, item := range m.models {
		if item.Name == model {
			return item, nil
		}
	}
	return ModelInfo{}, fmt.Errorf("not found")
}
func (m *mockProvider) ListModels(_ context.Context) ([]ModelInfo, error) { return m.models, nil }
func (m *mockProvider) SupportsStreaming() bool                           { return true }
func (m *mockProvider) SupportsTools() bool                               { return true }
