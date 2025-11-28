package memory

import (
	"context"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// EmbeddingProvider defines interface for generating text embeddings
type EmbeddingProvider interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
	Name() string    // Returns provider/model name (e.g., "text-embedding-3-small")
	Dimensions() int // Returns embedding dimensions
}

// Service provides operations for user and collective memories.
type Service struct {
	repo              Repository
	embeddingProvider EmbeddingProvider
	log               *logger.Logger
}

// NewService constructs a memory service with embedding support.
func NewService(repo Repository, embeddingProvider EmbeddingProvider) *Service {
	return &Service{
		repo:              repo,
		embeddingProvider: embeddingProvider,
		log:               logger.Get(),
	}
}

// Store adds a new memory item for a user.
func (s *Service) Store(ctx context.Context, memory *Memory) error {
	if memory == nil {
		return errors.ErrInvalidInput
	}
	// For user-scope memories, UserID is required
	if memory.Scope == MemoryScopeUser && (memory.UserID == nil || *memory.UserID == uuid.Nil) {
		return errors.ErrInvalidInput
	}
	if err := s.repo.Store(ctx, memory); err != nil {
		return errors.Wrap(err, "store memory")
	}
	return nil
}

// StoreWithText adds a new memory item and automatically generates embedding from text.
// This is the recommended method for storing memories as it handles embedding generation.
func (s *Service) StoreWithText(ctx context.Context, memory *Memory, searchText string) error {
	if memory == nil {
		return errors.ErrInvalidInput
	}
	// For user-scope memories, UserID is required
	if memory.Scope == MemoryScopeUser && (memory.UserID == nil || *memory.UserID == uuid.Nil) {
		return errors.ErrInvalidInput
	}
	if searchText == "" {
		return errors.Wrapf(errors.ErrInvalidInput, "search text is required for embedding generation")
	}

	// Generate embedding
	if s.embeddingProvider == nil {
		return errors.Wrapf(errors.ErrInternal, "embedding provider not configured")
	}

	embeddingVec, err := s.embeddingProvider.GenerateEmbedding(ctx, searchText)
	if err != nil {
		return errors.Wrap(err, "failed to generate embedding")
	}

	// Set embedding and metadata about the model
	memory.Embedding = pgvector.NewVector(embeddingVec)
	memory.EmbeddingModel = s.embeddingProvider.Name()
	memory.EmbeddingDimensions = s.embeddingProvider.Dimensions()

	// Store with generated embedding
	if err := s.repo.Store(ctx, memory); err != nil {
		return errors.Wrap(err, "store memory")
	}

	return nil
}

// SearchSimilar performs semantic search for related memories using text query.
// Automatically generates embedding from query text and filters by model.
func (s *Service) SearchSimilar(ctx context.Context, userID uuid.UUID, queryText string, limit int) ([]*Memory, error) {
	if userID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}
	if queryText == "" {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "query text is required")
	}
	if limit <= 0 {
		limit = 10
	}

	// Generate embedding from query text
	if s.embeddingProvider == nil {
		return nil, errors.Wrapf(errors.ErrInternal, "embedding provider not configured")
	}

	embeddingVec, err := s.embeddingProvider.GenerateEmbedding(ctx, queryText)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate embedding for query")
	}

	embedding := pgvector.NewVector(embeddingVec)
	modelName := s.embeddingProvider.Name()

	// Perform vector search with model filtering
	results, err := s.repo.SearchSimilar(ctx, userID, modelName, embedding, limit)
	if err != nil {
		return nil, errors.Wrap(err, "search memory")
	}

	return results, nil
}

// SearchCollectiveSimilar searches collective memories by embedding.
func (s *Service) SearchCollectiveSimilar(ctx context.Context, agentID string, embedding pgvector.Vector, limit int) ([]*Memory, error) {
	if agentID == "" {
		return nil, errors.ErrInvalidInput
	}
	if limit <= 0 {
		limit = 10
	}
	results, err := s.repo.SearchCollectiveSimilar(ctx, embedding, agentID, limit)
	if err != nil {
		return nil, errors.Wrap(err, "search collective memory")
	}
	return results, nil
}

// GetValidatedLessons returns the highest scoring collective lessons.
func (s *Service) GetValidatedLessons(ctx context.Context, agentID string, minScore float64, limit int) ([]*Memory, error) {
	if agentID == "" {
		return nil, errors.ErrInvalidInput
	}
	if limit <= 0 {
		limit = 10
	}
	lessons, err := s.repo.GetValidated(ctx, agentID, minScore, limit)
	if err != nil {
		return nil, errors.Wrap(err, "get validated lessons")
	}
	return lessons, nil
}

// GetRecentByAgent retrieves recent memories for a specific agent.
func (s *Service) GetRecentByAgent(ctx context.Context, userID uuid.UUID, agentID string, limit int) ([]*Memory, error) {
	if userID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}
	if agentID == "" {
		return nil, errors.ErrInvalidInput
	}
	if limit <= 0 {
		limit = 10
	}
	memories, err := s.repo.GetByAgent(ctx, userID, agentID, limit)
	if err != nil {
		return nil, errors.Wrap(err, "get memories by agent")
	}
	return memories, nil
}
