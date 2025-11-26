package memory

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	"prometheus/pkg/logger"
)

// Service provides operations for user and collective memories.
type Service struct {
	repo Repository
	log  *logger.Logger
}

// NewService constructs a memory service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, log: logger.Get()}
}

// Store adds a new memory item for a user.
func (s *Service) Store(ctx context.Context, memory *Memory) error {
	if memory == nil {
		return fmt.Errorf("store memory: memory is nil")
	}
	if memory.UserID == uuid.Nil {
		return fmt.Errorf("store memory: user id is required")
	}
	if err := s.repo.Store(ctx, memory); err != nil {
		return fmt.Errorf("store memory: %w", err)
	}
	return nil
}

// SearchSimilar performs vector search for related memories.
func (s *Service) SearchSimilar(ctx context.Context, userID uuid.UUID, embedding pgvector.Vector, limit int) ([]*Memory, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("search memory: user id is required")
	}
	if limit <= 0 {
		limit = 10
	}
	results, err := s.repo.SearchSimilar(ctx, userID, embedding, limit)
	if err != nil {
		return nil, fmt.Errorf("search memory: %w", err)
	}
	return results, nil
}

// StoreCollective saves shared lessons across agents.
func (s *Service) StoreCollective(ctx context.Context, memory *CollectiveMemory) error {
	if memory == nil {
		return fmt.Errorf("store collective memory: memory is nil")
	}
	if memory.AgentType == "" {
		return fmt.Errorf("store collective memory: agent type is required")
	}
	if err := s.repo.StoreCollective(ctx, memory); err != nil {
		return fmt.Errorf("store collective memory: %w", err)
	}
	return nil
}

// SearchCollectiveSimilar searches collective memories by embedding.
func (s *Service) SearchCollectiveSimilar(ctx context.Context, agentType string, embedding pgvector.Vector, limit int) ([]*CollectiveMemory, error) {
	if agentType == "" {
		return nil, fmt.Errorf("search collective memory: agent type is required")
	}
	if limit <= 0 {
		limit = 10
	}
	results, err := s.repo.SearchCollectiveSimilar(ctx, agentType, embedding, limit)
	if err != nil {
		return nil, fmt.Errorf("search collective memory: %w", err)
	}
	return results, nil
}

// GetValidatedLessons returns the highest scoring collective lessons.
func (s *Service) GetValidatedLessons(ctx context.Context, agentType string, minScore float64, limit int) ([]*CollectiveMemory, error) {
	if agentType == "" {
		return nil, fmt.Errorf("get validated lessons: agent type is required")
	}
	if limit <= 0 {
		limit = 10
	}
	lessons, err := s.repo.GetValidatedLessons(ctx, agentType, minScore, limit)
	if err != nil {
		return nil, fmt.Errorf("get validated lessons: %w", err)
	}
	return lessons, nil
}
