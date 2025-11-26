package ai

import (
	"context"
	"fmt"
	"sync"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// ProviderRegistry stores all available AI providers.
type ProviderRegistry struct {
	providers map[string]Provider
	mu        sync.RWMutex
	log       *logger.Logger
}

// NewProviderRegistry creates an empty registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
		log:       logger.Get().With("component", "ai_registry"),
	}
}

// Register adds a provider to the registry.
func (r *ProviderRegistry) Register(provider Provider) error {
	if provider == nil {
		r.log.Error("Attempted to register nil AI provider")
		return errors.ErrInvalidInput
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	name := provider.Name()
	if _, exists := r.providers[name]; exists {
		r.log.Warn("AI provider already registered", "provider", name)
		return errors.Wrapf(errors.ErrAlreadyExists, "provider %s already registered", name)
	}

	r.providers[name] = provider
	r.log.Info("AI provider registered successfully", "provider", name)
	return nil
}

// Get returns the provider by name.
func (r *ProviderRegistry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, ok := r.providers[name]
	if !ok {
		r.log.Warn("AI provider not found", "provider", name, "available", len(r.providers))
		return nil, errors.Wrapf(errors.ErrNotFound, "provider %s not found", name)
	}

	return provider, nil
}

// MustGet returns the provider by name and panics if missing.
// Deprecated: Use Get instead and handle errors properly.
func (r *ProviderRegistry) MustGet(name string) Provider {
	provider, err := r.Get(name)
	if err != nil {
		// This should never happen in production if registry is properly initialized
		// Log the error before panicking to ensure it's captured
		panic(fmt.Errorf("critical: AI provider '%s' not found in registry: %w", name, err))
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
			r.log.Error("Failed to list models for provider", "provider", name, "error", err)
			return nil, errors.Wrapf(err, "failed to list models for provider %s", name)
		}
		result[name] = models
	}

	r.log.Debug("Listed models from all providers", "providers", len(result))
	return result, nil
}

// ResolveModel fetches model metadata for provider+model combination.
func (r *ProviderRegistry) ResolveModel(ctx context.Context, providerName string, model string) (ModelInfo, error) {
	provider, err := r.Get(providerName)
	if err != nil {
		r.log.Error("Failed to resolve model - provider not found", "provider", providerName, "model", model, "error", err)
		return ModelInfo{}, err
	}

	modelInfo, err := provider.GetModel(ctx, model)
	if err != nil {
		r.log.Error("Failed to get model info", "provider", providerName, "model", model, "error", err)
		return ModelInfo{}, err
	}

	r.log.Debug("Model resolved successfully", "provider", providerName, "model", model)
	return modelInfo, nil
}
