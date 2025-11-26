package memory

import (
	"context"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// CollectiveRepository defines operations for collective memory
type CollectiveRepository interface {
	// Store saves a collective memory entry
	Store(ctx context.Context, memory *CollectiveMemory) error

	// SearchSimilar performs semantic search across collective memories
	SearchSimilar(ctx context.Context, embedding pgvector.Vector, agentType string, limit int) ([]*CollectiveMemory, error)

	// SearchByPersonality searches memories for a specific personality type
	SearchByPersonality(ctx context.Context, embedding pgvector.Vector, agentType string, personality string, limit int) ([]*CollectiveMemory, error)

	// GetValidated returns only validated memories (high confidence)
	GetValidated(ctx context.Context, agentType string, minScore float64, limit int) ([]*CollectiveMemory, error)

	// GetByType returns memories filtered by type
	GetByType(ctx context.Context, agentType string, memType MemoryType, limit int) ([]*CollectiveMemory, error)

	// UpdateValidation updates validation score and trade count
	UpdateValidation(ctx context.Context, id uuid.UUID, score float64, tradeCount int) error

	// PromoteToCollective promotes a user memory to collective after validation
	PromoteToCollective(ctx context.Context, userMemory *Memory, validationScore float64, validationTrades int) (*CollectiveMemory, error)

	// DeleteExpired removes expired collective memories
	DeleteExpired(ctx context.Context) (int64, error)
}
