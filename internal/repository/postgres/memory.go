package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pgvector/pgvector-go"

	"prometheus/internal/domain/memory"
)

// Compile-time check
var _ memory.Repository = (*MemoryRepository)(nil)

// MemoryRepository implements memory.Repository using sqlx and pgvector
type MemoryRepository struct {
	db *sqlx.DB
}

// NewMemoryRepository creates a new memory repository
func NewMemoryRepository(db *sqlx.DB) *MemoryRepository {
	return &MemoryRepository{db: db}
}

// Store inserts a new memory
func (r *MemoryRepository) Store(ctx context.Context, m *memory.Memory) error {
	query := `
		INSERT INTO memories (
			id, user_id, agent_id, session_id, type, content, embedding,
			symbol, timeframe, importance, related_ids, trade_id, created_at, expires_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)`

	_, err := r.db.ExecContext(ctx, query,
		m.ID, m.UserID, m.AgentID, m.SessionID, m.Type, m.Content, m.Embedding,
		m.Symbol, m.Timeframe, m.Importance, m.RelatedIDs, m.TradeID, m.CreatedAt, m.ExpiresAt,
	)

	return err
}

// SearchSimilar performs semantic search using pgvector cosine similarity
func (r *MemoryRepository) SearchSimilar(ctx context.Context, userID uuid.UUID, embedding pgvector.Vector, limit int) ([]*memory.Memory, error) {
	var memories []*memory.Memory

	query := `
		SELECT *, 1 - (embedding <=> $2) as similarity
		FROM memories
		WHERE user_id = $1
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY embedding <=> $2
		LIMIT $3`

	err := r.db.SelectContext(ctx, &memories, query, userID, embedding, limit)
	if err != nil {
		return nil, err
	}

	return memories, nil
}

// GetByAgent retrieves memories for a specific agent
func (r *MemoryRepository) GetByAgent(ctx context.Context, userID uuid.UUID, agentID string, limit int) ([]*memory.Memory, error) {
	var memories []*memory.Memory

	query := `
		SELECT * FROM memories
		WHERE user_id = $1 AND agent_id = $2
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC
		LIMIT $3`

	err := r.db.SelectContext(ctx, &memories, query, userID, agentID, limit)
	if err != nil {
		return nil, err
	}

	return memories, nil
}

// GetByType retrieves memories by type
func (r *MemoryRepository) GetByType(ctx context.Context, userID uuid.UUID, memType memory.MemoryType, limit int) ([]*memory.Memory, error) {
	var memories []*memory.Memory

	query := `
		SELECT * FROM memories
		WHERE user_id = $1 AND type = $2
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY importance DESC, created_at DESC
		LIMIT $3`

	err := r.db.SelectContext(ctx, &memories, query, userID, memType, limit)
	if err != nil {
		return nil, err
	}

	return memories, nil
}

// DeleteExpired removes expired memories
func (r *MemoryRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM memories WHERE expires_at < NOW()`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// StoreCollective inserts a new collective memory
func (r *MemoryRepository) StoreCollective(ctx context.Context, m *memory.CollectiveMemory) error {
	query := `
		INSERT INTO collective_memories (
			id, agent_type, personality, type, content, embedding,
			validation_score, validation_trades, validated_at,
			symbol, timeframe, importance,
			source_user_id, source_trade_id, created_at, expires_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)`

	_, err := r.db.ExecContext(ctx, query,
		m.ID, m.AgentType, m.Personality, m.Type, m.Content, m.Embedding,
		m.ValidationScore, m.ValidationTrades, m.ValidatedAt,
		m.Symbol, m.Timeframe, m.Importance,
		m.SourceUserID, m.SourceTradeID, m.CreatedAt, m.ExpiresAt,
	)

	return err
}

// SearchCollectiveSimilar performs semantic search on collective memory
func (r *MemoryRepository) SearchCollectiveSimilar(ctx context.Context, agentType string, embedding pgvector.Vector, limit int) ([]*memory.CollectiveMemory, error) {
	var memories []*memory.CollectiveMemory

	query := `
		SELECT *, 1 - (embedding <=> $2) as similarity
		FROM collective_memories
		WHERE agent_type = $1
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY embedding <=> $2
		LIMIT $3`

	err := r.db.SelectContext(ctx, &memories, query, agentType, embedding, limit)
	if err != nil {
		return nil, err
	}

	return memories, nil
}

// GetValidatedLessons retrieves validated collective lessons
func (r *MemoryRepository) GetValidatedLessons(ctx context.Context, agentType string, minScore float64, limit int) ([]*memory.CollectiveMemory, error) {
	var memories []*memory.CollectiveMemory

	query := `
		SELECT * FROM collective_memories
		WHERE agent_type = $1
		  AND validation_score >= $2
		  AND validated_at IS NOT NULL
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY validation_score DESC, validation_trades DESC
		LIMIT $3`

	err := r.db.SelectContext(ctx, &memories, query, agentType, minScore, limit)
	if err != nil {
		return nil, err
	}

	return memories, nil
}
