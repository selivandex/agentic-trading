package ai

import "context"

// Provider defines the contract each AI provider implementation must satisfy.
type Provider interface {
	Name() string

	// GetModel returns metadata for a specific model.
	GetModel(ctx context.Context, model string) (ModelInfo, error)

	// ListModels returns the list of available models for the provider.
	ListModels(ctx context.Context) ([]ModelInfo, error)

	// SupportsStreaming indicates whether the provider can stream responses.
	SupportsStreaming() bool

	// SupportsTools indicates whether the provider supports tool/function calling.
	SupportsTools() bool
}

// ModelInfo describes the capabilities and pricing of a model.
type ModelInfo struct {
	Name              string  // Provider-specific model identifier
	Family            string  // Family/category name (e.g., "claude-3")
	MaxTokens         int     // Maximum context length
	InputCostPer1K    float64 // USD per 1K input tokens
	OutputCostPer1K   float64 // USD per 1K output tokens
	SupportsImages    bool    // Whether the model accepts image inputs
	SupportsAudio     bool    // Whether the model accepts audio inputs
	SupportsTools     bool    // Whether tool calling is supported
	SupportsStreaming bool    // Whether streaming responses are available
}
