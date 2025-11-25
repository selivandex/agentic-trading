package memory

import (
	"context"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// Repository defines the interface for memory data access
type Repository interface {
	// User memory operations
	Store(ctx context.Context, memory *Memory) error
	SearchSimilar(ctx context.Context, userID uuid.UUID, embedding pgvector.Vector, limit int) ([]*Memory, error)
	GetByAgent(ctx context.Context, userID uuid.UUID, agentID string, limit int) ([]*Memory, error)
	GetByType(ctx context.Context, userID uuid.UUID, memType MemoryType, limit int) ([]*Memory, error)
	DeleteExpired(ctx context.Context) (int64, error)

	// Collective memory operations
	StoreCollective(ctx context.Context, memory *CollectiveMemory) error
	SearchCollectiveSimilar(ctx context.Context, agentType string, embedding pgvector.Vector, limit int) ([]*CollectiveMemory, error)
	GetValidatedLessons(ctx context.Context, agentType string, minScore float64, limit int) ([]*CollectiveMemory, error)
}

