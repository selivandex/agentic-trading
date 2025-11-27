package embeddings

import (
	"context"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// OpenAIProvider implements embedding generation using official OpenAI Go SDK
type OpenAIProvider struct {
	client     openai.Client // NewClient returns Client (not *Client)
	model      openai.EmbeddingModel
	dimensions int
	timeout    time.Duration
	log        *logger.Logger
}

// NewOpenAIProvider creates a new OpenAI embedding provider using official SDK
func NewOpenAIProvider(apiKey string, model string, timeout time.Duration) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "openai API key is required")
	}

	if model == "" {
		model = openai.EmbeddingModelTextEmbedding3Small // text-embedding-3-small
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Determine dimensions based on model
	dimensions := getDimensions(model)

	// Create client using official SDK (returns *openai.Client)
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &OpenAIProvider{
		client:     client,
		model:      openai.EmbeddingModel(model),
		dimensions: dimensions,
		timeout:    timeout,
		log:        logger.Get().With("component", "openai_embeddings", "model", model),
	}, nil
}

// GenerateEmbedding creates a vector embedding for the given text using official OpenAI SDK
func (p *OpenAIProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "text cannot be empty")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Create embedding using official SDK
	response, err := p.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(text),
		},
		Model: p.model,
	})
	if err != nil {
		return nil, errors.Wrap(err, "openai API call failed")
	}

	if len(response.Data) == 0 {
		return nil, errors.Wrapf(errors.ErrInternal, "no embedding data returned")
	}

	// Convert []float64 to []float32 for pgvector compatibility
	embeddingData := response.Data[0].Embedding
	result := make([]float32, len(embeddingData))
	for i, val := range embeddingData {
		result[i] = float32(val)
	}

	p.log.Debug("Generated embedding",
		"text_length", len(text),
		"embedding_dims", len(result),
		"tokens_used", response.Usage.TotalTokens)

	return result, nil
}

// GenerateBatchEmbeddings creates embeddings for multiple texts in one API call using official SDK
func (p *OpenAIProvider) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "texts cannot be empty")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Create batch embedding using official SDK (pass array of strings directly)
	response, err := p.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
		Model: p.model,
	})
	if err != nil {
		return nil, errors.Wrap(err, "openai batch API call failed")
	}

	if len(response.Data) != len(texts) {
		return nil, errors.Wrapf(errors.ErrInternal, "expected %d embeddings, got %d", len(texts), len(response.Data))
	}

	// Extract and convert embeddings from []float64 to [][]float32
	embeddings := make([][]float32, len(response.Data))
	for i, data := range response.Data {
		result := make([]float32, len(data.Embedding))
		for j, val := range data.Embedding {
			result[j] = float32(val)
		}
		embeddings[i] = result
	}

	p.log.Debug("Generated batch embeddings",
		"batch_size", len(texts),
		"embedding_dims", len(embeddings[0]),
		"tokens_used", response.Usage.TotalTokens)

	return embeddings, nil
}

// Dimensions returns the dimensionality of embeddings
func (p *OpenAIProvider) Dimensions() int {
	return p.dimensions
}

// Name returns the model name (e.g., "text-embedding-3-small")
// IMPORTANT: This is used for filtering embeddings during search
func (p *OpenAIProvider) Name() string {
	return string(p.model)
}

// getDimensions returns embedding dimensions for known OpenAI models
func getDimensions(model string) int {
	switch model {
	case openai.EmbeddingModelTextEmbedding3Small:
		return 1536 // text-embedding-3-small
	case openai.EmbeddingModelTextEmbedding3Large:
		return 3072 // text-embedding-3-large
	case openai.EmbeddingModelTextEmbeddingAda002:
		return 1536 // text-embedding-ada-002
	default:
		return 1536 // default to 1536
	}
}
