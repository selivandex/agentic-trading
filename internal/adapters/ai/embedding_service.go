package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

const openaiEmbeddingURL = "https://api.openai.com/v1/embeddings"

// EmbeddingService generates vector embeddings for text using OpenAI API
type EmbeddingService struct {
	apiKey  string
	model   string
	timeout time.Duration
	log     *logger.Logger
}

// NewEmbeddingService creates a new embedding service using OpenAI
func NewEmbeddingService(apiKey string, model string, timeout time.Duration) *EmbeddingService {
	if model == "" {
		model = "text-embedding-3-small" // Default: 1536 dimensions, $0.02/1M tokens
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &EmbeddingService{
		apiKey:  apiKey,
		model:   model,
		timeout: timeout,
		log:     logger.Get().With("component", "embedding_service", "model", model),
	}
}

// embeddingRequest represents OpenAI embedding API request
type embeddingRequest struct {
	Input          interface{} `json:"input"` // string or []string
	Model          string      `json:"model"`
	EncodingFormat string      `json:"encoding_format,omitempty"` // "float" or "base64"
}

// embeddingResponse represents OpenAI embedding API response
type embeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// GenerateEmbedding creates a vector embedding for the given text using OpenAI API
func (s *EmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "text cannot be empty")
	}

	if s.apiKey == "" {
		return nil, errors.Wrap(errors.ErrInvalidInput, "openai API key not configured")
	}

	// Prepare request
	req := embeddingRequest{
		Input:          text,
		Model:          s.model,
		EncodingFormat: "float",
	}

	// Marshal request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "marshal embedding request")
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", openaiEmbeddingURL, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "create HTTP request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	// Send request
	client := &http.Client{Timeout: s.timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "send embedding request")
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read embedding response")
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			return nil, errors.Wrapf(errors.ErrExternal, "openai embedding API error (%d): %s - %s",
				resp.StatusCode, errResp.Error.Type, errResp.Error.Message)
		}
		return nil, errors.Wrapf(errors.ErrExternal, "openai embedding API error (%d): %s",
			resp.StatusCode, string(respBody))
	}

	// Parse response
	var embResp embeddingResponse
	if err := json.Unmarshal(respBody, &embResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal embedding response")
	}

	if len(embResp.Data) == 0 {
		return nil, errors.Wrapf(errors.ErrInternal, "no embedding data returned")
	}

	// Convert []float64 to []float32 for pgvector compatibility
	embedding64 := embResp.Data[0].Embedding
	embedding32 := make([]float32, len(embedding64))
	for i, v := range embedding64 {
		embedding32[i] = float32(v)
	}

	s.log.Debug("Generated embedding",
		"text_length", len(text),
		"embedding_dims", len(embedding32),
		"tokens_used", embResp.Usage.TotalTokens)

	return embedding32, nil
}

// GenerateBatchEmbeddings creates embeddings for multiple texts in one API call
func (s *EmbeddingService) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "texts cannot be empty")
	}

	if s.apiKey == "" {
		return nil, errors.Wrap(errors.ErrInvalidInput, "openai API key not configured")
	}

	// Prepare batch request
	req := embeddingRequest{
		Input:          texts,
		Model:          s.model,
		EncodingFormat: "float",
	}

	// Marshal request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "marshal batch embedding request")
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", openaiEmbeddingURL, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "create HTTP request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	// Send request
	client := &http.Client{Timeout: s.timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "send batch embedding request")
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read batch embedding response")
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			return nil, errors.Wrapf(errors.ErrExternal, "openai batch embedding API error (%d): %s - %s",
				resp.StatusCode, errResp.Error.Type, errResp.Error.Message)
		}
		return nil, errors.Wrapf(errors.ErrExternal, "openai batch embedding API error (%d): %s",
			resp.StatusCode, string(respBody))
	}

	// Parse response
	var embResp embeddingResponse
	if err := json.Unmarshal(respBody, &embResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal batch embedding response")
	}

	if len(embResp.Data) != len(texts) {
		return nil, errors.Wrapf(errors.ErrInternal, "expected %d embeddings, got %d", len(texts), len(embResp.Data))
	}

	// Convert all embeddings
	embeddings := make([][]float32, len(embResp.Data))
	for i, data := range embResp.Data {
		embedding32 := make([]float32, len(data.Embedding))
		for j, v := range data.Embedding {
			embedding32[j] = float32(v)
		}
		embeddings[i] = embedding32
	}

	s.log.Debug("Generated batch embeddings",
		"batch_size", len(texts),
		"embedding_dims", len(embeddings[0]),
		"tokens_used", embResp.Usage.TotalTokens)

	return embeddings, nil
}
