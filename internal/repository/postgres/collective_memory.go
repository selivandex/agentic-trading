package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pgvector/pgvector-go"

	"prometheus/internal/domain/memory"
	pkgerrors "prometheus/pkg/errors"
)

// Compile-time check that we implement the interface
var _ memory.CollectiveRepository = (*CollectiveMemoryRepository)(nil)

// CollectiveMemoryRepository implements collective memory storage with pgvector
type CollectiveMemoryRepository struct {
	db *sqlx.DB
}

// NewCollectiveMemoryRepository creates a new collective memory repository
func NewCollectiveMemoryRepository(db *sqlx.DB) *CollectiveMemoryRepository {
	return &CollectiveMemoryRepository{db: db}
}

// Store saves a collective memory entry
func (r *CollectiveMemoryRepository) Store(ctx context.Context, mem *memory.CollectiveMemory) error {
	query := `
		INSERT INTO collective_memories (
			id, agent_type, personality, type, content, embedding,
			validation_score, validation_trades, validated_at,
			symbol, timeframe, importance, tags,
			source_user_id, source_trade_id,
			created_at, updated_at, expires_at
		) VALUES (
			:id, :agent_type, :personality, :type, :content, :embedding,
			:validation_score, :validation_trades, :validated_at,
			:symbol, :timeframe, :importance, :tags,
			:source_user_id, :source_trade_id,
			:created_at, :updated_at, :expires_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, mem)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to store collective memory")
	}

	return nil
}

// SearchSimilar performs semantic search using cosine similarity
func (r *CollectiveMemoryRepository) SearchSimilar(
	ctx context.Context,
	embedding pgvector.Vector,
	agentType string,
	limit int,
) ([]*memory.CollectiveMemory, error) {
	var memories []*memory.CollectiveMemory

	query := `
		SELECT *,
			1 - (embedding <=> $1) as similarity
		FROM collective_memories
		WHERE agent_type = $2
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY embedding <=> $1
		LIMIT $3`

	if err := r.db.SelectContext(ctx, &memories, query, embedding, agentType, limit); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to search collective memories")
	}

	return memories, nil
}

// SearchByPersonality searches memories for a specific personality type
func (r *CollectiveMemoryRepository) SearchByPersonality(
	ctx context.Context,
	embedding pgvector.Vector,
	agentType string,
	personality string,
	limit int,
) ([]*memory.CollectiveMemory, error) {
	var memories []*memory.CollectiveMemory

	query := `
		SELECT *,
			1 - (embedding <=> $1) as similarity
		FROM collective_memories
		WHERE agent_type = $2
			AND (personality = $3 OR personality IS NULL)
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY embedding <=> $1
		LIMIT $4`

	if err := r.db.SelectContext(ctx, &memories, query, embedding, agentType, personality, limit); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to search collective memories by personality")
	}

	return memories, nil
}

// GetValidated returns only validated memories with high confidence
func (r *CollectiveMemoryRepository) GetValidated(
	ctx context.Context,
	agentType string,
	minScore float64,
	limit int,
) ([]*memory.CollectiveMemory, error) {
	var memories []*memory.CollectiveMemory

	query := `
		SELECT *
		FROM collective_memories
		WHERE agent_type = $1
			AND validated_at IS NOT NULL
			AND validation_score >= $2
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY validation_score DESC, validation_trades DESC, importance DESC
		LIMIT $3`

	if err := r.db.SelectContext(ctx, &memories, query, agentType, minScore, limit); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to get validated memories")
	}

	return memories, nil
}

// GetByType returns memories filtered by type
func (r *CollectiveMemoryRepository) GetByType(
	ctx context.Context,
	agentType string,
	memType memory.MemoryType,
	limit int,
) ([]*memory.CollectiveMemory, error) {
	var memories []*memory.CollectiveMemory

	query := `
		SELECT *
		FROM collective_memories
		WHERE agent_type = $1
			AND type = $2
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY validation_score DESC, created_at DESC
		LIMIT $3`

	if err := r.db.SelectContext(ctx, &memories, query, agentType, memType, limit); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to get memories by type")
	}

	return memories, nil
}

// UpdateValidation updates validation metrics for a memory
func (r *CollectiveMemoryRepository) UpdateValidation(
	ctx context.Context,
	id uuid.UUID,
	score float64,
	tradeCount int,
) error {
	query := `
		UPDATE collective_memories
		SET 
			validation_score = $2,
			validation_trades = $3,
			validated_at = CASE WHEN validated_at IS NULL THEN NOW() ELSE validated_at END,
			updated_at = NOW()
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, score, tradeCount)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to update validation")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return pkgerrors.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return pkgerrors.Wrap(sql.ErrNoRows, "collective memory not found")
	}

	return nil
}

// PromoteToCollective converts a validated user memory to collective memory
func (r *CollectiveMemoryRepository) PromoteToCollective(
	ctx context.Context,
	userMemory *memory.Memory,
	validationScore float64,
	validationTrades int,
) (*memory.CollectiveMemory, error) {
	now := time.Now()

	collective := &memory.CollectiveMemory{
		ID:               uuid.New(),
		AgentType:        userMemory.AgentID,
		Type:             userMemory.Type,
		Content:          userMemory.Content,
		Embedding:        userMemory.Embedding,
		ValidationScore:  validationScore,
		ValidationTrades: validationTrades,
		ValidatedAt:      &now,
		Importance:       userMemory.Importance,
		SourceUserID:     &userMemory.UserID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Copy optional fields
	if userMemory.Symbol != "" {
		collective.Symbol = &userMemory.Symbol
	}
	if userMemory.Timeframe != "" {
		collective.Timeframe = &userMemory.Timeframe
	}
	if userMemory.TradeID != nil {
		collective.SourceTradeID = userMemory.TradeID
	}

	if err := r.Store(ctx, collective); err != nil {
		return nil, err
	}

	return collective, nil
}

// DeleteExpired removes expired collective memories
func (r *CollectiveMemoryRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM collective_memories 
		WHERE expires_at IS NOT NULL AND expires_at < NOW()
	`)
	if err != nil {
		return 0, pkgerrors.Wrap(err, "failed to delete expired collective memories")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, pkgerrors.Wrap(err, "failed to get rows affected")
	}

	return rows, nil
}
