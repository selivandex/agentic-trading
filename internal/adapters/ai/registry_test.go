package ai

import (
	"context"
	"fmt"
	"testing"
)

type mockProvider struct {
	name           string
	models         []ModelInfo
	listErr        error
	getErr         error
	supportsStream bool
	supportsTools  bool
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) GetModel(_ context.Context, model string) (ModelInfo, error) {
	if m.getErr != nil {
		return ModelInfo{}, m.getErr
	}

	for _, item := range m.models {
		if item.Name == model {
			return item, nil
		}
	}

	return ModelInfo{}, errNotFound(model)
}
func (m *mockProvider) ListModels(_ context.Context) ([]ModelInfo, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.models, nil
}
func (m *mockProvider) SupportsStreaming() bool { return m.supportsStream }
func (m *mockProvider) SupportsTools() bool     { return m.supportsTools }

func errNotFound(model string) error { return fmt.Errorf("not found: %s", model) }

func TestProviderRegistryRegisterAndGet(t *testing.T) {
	registry := NewProviderRegistry()
	provider := &mockProvider{name: "mock", models: []ModelInfo{{Name: "alpha"}}}

	if err := registry.Register(provider); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	got, err := registry.Get("mock")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if got.Name() != provider.Name() {
		t.Fatalf("expected provider %s, got %s", provider.Name(), got.Name())
	}
}

func TestProviderRegistryRejectsDuplicate(t *testing.T) {
	registry := NewProviderRegistry()
	provider := &mockProvider{name: "mock"}

	if err := registry.Register(provider); err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	if err := registry.Register(provider); err == nil {
		t.Fatal("expected duplicate registration to error")
	}
}

func TestProviderRegistryListModels(t *testing.T) {
	registry := NewProviderRegistry()
	provider := &mockProvider{name: "mock", models: []ModelInfo{{Name: "alpha"}, {Name: "beta"}}}

	if err := registry.Register(provider); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	models, err := registry.ListModels(context.Background())
	if err != nil {
		t.Fatalf("list models failed: %v", err)
	}

	if len(models) != 1 || len(models["mock"]) != 2 {
		t.Fatalf("unexpected models map: %+v", models)
	}
}

func TestProviderRegistryResolveModel(t *testing.T) {
	registry := NewProviderRegistry()
	provider := &mockProvider{name: "mock", models: []ModelInfo{{Name: "alpha"}}}

	if err := registry.Register(provider); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	info, err := registry.ResolveModel(context.Background(), "mock", "alpha")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if info.Name != "alpha" {
		t.Fatalf("expected alpha, got %s", info.Name)
	}
}

func TestProviderRegistryResolveModelMissingProvider(t *testing.T) {
	registry := NewProviderRegistry()
	if _, err := registry.ResolveModel(context.Background(), "missing", "any"); err == nil {
		t.Fatal("expected error for missing provider")
	}
}
