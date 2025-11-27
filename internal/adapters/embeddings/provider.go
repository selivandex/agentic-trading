package embeddings

import "context"

// Provider defines the interface for embedding generation services
// Implementations can use different backends: OpenAI, Cohere, local models, etc.
type Provider interface {
	// GenerateEmbedding creates a vector embedding for a single text
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)

	// GenerateBatchEmbeddings creates embeddings for multiple texts in one call
	// More efficient than calling GenerateEmbedding multiple times
	GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error)

	// Dimensions returns the dimensionality of embeddings produced by this provider
	Dimensions() int

	// Name returns the provider name (e.g., "openai", "cohere", "local")
	Name() string
}
