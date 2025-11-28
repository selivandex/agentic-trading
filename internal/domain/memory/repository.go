package memory

import (
	"context"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// Repository handles all memory operations (user, collective, working)
type Repository interface {
	// User memory operations
	Store(ctx context.Context, memory *Memory) error
	GetByID(ctx context.Context, id uuid.UUID) (*Memory, error)
	SearchSimilar(ctx context.Context, userID uuid.UUID, agentID string, embedding pgvector.Vector, limit int) ([]*Memory, error)
	GetByAgent(ctx context.Context, userID uuid.UUID, agentID string, limit int) ([]*Memory, error)
	GetByType(ctx context.Context, userID uuid.UUID, memType MemoryType, limit int) ([]*Memory, error)
	DeleteExpired(ctx context.Context) (int64, error)

	// Collective memory operations
	SearchCollectiveSimilar(ctx context.Context, embedding pgvector.Vector, agentID string, limit int) ([]*Memory, error)
	SearchCollectiveByPersonality(ctx context.Context, embedding pgvector.Vector, agentID string, personality string, limit int) ([]*Memory, error)
	GetValidated(ctx context.Context, agentID string, minScore float64, limit int) ([]*Memory, error)
	GetCollectiveByType(ctx context.Context, agentID string, memType MemoryType, limit int) ([]*Memory, error)
	UpdateValidation(ctx context.Context, id uuid.UUID, score float64, tradeCount int) error
	PromoteToCollective(ctx context.Context, userMemory *Memory, validationScore float64, validationTrades int) (*Memory, error)
}
