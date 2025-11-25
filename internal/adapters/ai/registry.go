package ai

import (
	"context"
	"fmt"
	"sync"
)

// ProviderRegistry stores all available AI providers.
type ProviderRegistry struct {
	providers map[string]Provider
	mu        sync.RWMutex
}

// NewProviderRegistry creates an empty registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry.
func (r *ProviderRegistry) Register(provider Provider) error {
	if provider == nil {
		return fmt.Errorf("provider is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	name := provider.Name()
	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}

	r.providers[name] = provider
	return nil
}

// Get returns the provider by name.
func (r *ProviderRegistry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return provider, nil
}

// MustGet returns the provider by name and panics if missing.
func (r *ProviderRegistry) MustGet(name string) Provider {
	provider, err := r.Get(name)
	if err != nil {
		panic(err)
	}

	return provider
}

// List returns all registered providers.
func (r *ProviderRegistry) List() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}

	return providers
}

// ListModels aggregates all models across providers.
func (r *ProviderRegistry) ListModels(ctx context.Context) (map[string][]ModelInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]ModelInfo, len(r.providers))
	for name, provider := range r.providers {
		models, err := provider.ListModels(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list models for provider %s: %w", name, err)
		}
		result[name] = models
	}

	return result, nil
}

// ResolveModel fetches model metadata for provider+model combination.
func (r *ProviderRegistry) ResolveModel(ctx context.Context, providerName string, model string) (ModelInfo, error) {
	provider, err := r.Get(providerName)
	if err != nil {
		return ModelInfo{}, err
	}

	return provider.GetModel(ctx, model)
}
