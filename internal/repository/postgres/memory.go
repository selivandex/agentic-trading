package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"

	"prometheus/internal/domain/memory"
	pkgerrors "prometheus/pkg/errors"
)

// Compile-time check
var _ memory.Repository = (*MemoryRepository)(nil)

// MemoryRepository implements unified memory.Repository using sqlx and pgvector
type MemoryRepository struct {
	db DBTX
}

// NewMemoryRepository creates a new unified memory repository
func NewMemoryRepository(db DBTX) *MemoryRepository {
	return &MemoryRepository{db: db}
}

// scanMemory scans a single memory from database row
func scanMemory(row interface {
	Scan(dest ...interface{}) error
}) (*memory.Memory, error) {
	m := &memory.Memory{}
	var metadataJSON []byte

	err := row.Scan(
		&m.ID, &m.Scope, &m.UserID, &m.AgentID, &m.SessionID, &m.Type, &m.Content,
		&m.Embedding, &m.EmbeddingModel, &m.EmbeddingDimensions,
		&m.ValidationScore, &m.ValidationTrades, &m.ValidatedAt,
		&m.Symbol, &m.Timeframe, &m.Importance, pq.Array(&m.Tags), &metadataJSON,
		&m.Personality, &m.SourceUserID, &m.SourceTradeID,
		&m.CreatedAt, &m.UpdatedAt, &m.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	// Unmarshal metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &m.Metadata); err != nil {
			return nil, pkgerrors.Wrap(err, "failed to unmarshal metadata")
		}
	}

	return m, nil
}

// scanMemoryWithSimilarity scans a memory with similarity score
func scanMemoryWithSimilarity(row interface {
	Scan(dest ...interface{}) error
}) (*memory.Memory, error) {
	m := &memory.Memory{}
	var metadataJSON []byte

	err := row.Scan(
		&m.ID, &m.Scope, &m.UserID, &m.AgentID, &m.SessionID, &m.Type, &m.Content,
		&m.Embedding, &m.EmbeddingModel, &m.EmbeddingDimensions,
		&m.ValidationScore, &m.ValidationTrades, &m.ValidatedAt,
		&m.Symbol, &m.Timeframe, &m.Importance, pq.Array(&m.Tags), &metadataJSON,
		&m.Personality, &m.SourceUserID, &m.SourceTradeID,
		&m.CreatedAt, &m.UpdatedAt, &m.ExpiresAt,
		&m.Similarity,
	)
	if err != nil {
		return nil, err
	}

	// Unmarshal metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &m.Metadata); err != nil {
			return nil, pkgerrors.Wrap(err, "failed to unmarshal metadata")
		}
	}

	return m, nil
}

// Store inserts a new memory (user, collective, or working)
func (r *MemoryRepository) Store(ctx context.Context, m *memory.Memory) error {
	query := `
		INSERT INTO memories (
			id, scope, user_id, agent_id, session_id, type, content,
			embedding, embedding_model, embedding_dimensions,
			validation_score, validation_trades, validated_at,
			symbol, timeframe, importance, tags, metadata,
			personality, source_user_id, source_trade_id,
			created_at, updated_at, expires_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10,
			$11, $12, $13,
			$14, $15, $16, $17, $18,
			$19, $20, $21,
			$22, $23, $24
		)`

	// Convert metadata to JSON
	var metadataJSON []byte
	var err error
	if m.Metadata != nil {
		metadataJSON, err = json.Marshal(m.Metadata)
		if err != nil {
			return pkgerrors.Wrap(err, "failed to marshal metadata")
		}
	} else {
		metadataJSON = []byte("{}")
	}

	_, err = r.db.ExecContext(ctx, query,
		m.ID, m.Scope, m.UserID, m.AgentID, m.SessionID, m.Type, m.Content,
		m.Embedding, m.EmbeddingModel, m.EmbeddingDimensions,
		m.ValidationScore, m.ValidationTrades, m.ValidatedAt,
		m.Symbol, m.Timeframe, m.Importance, pq.Array(m.Tags), metadataJSON,
		m.Personality, m.SourceUserID, m.SourceTradeID,
		m.CreatedAt, m.UpdatedAt, m.ExpiresAt,
	)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to store memory")
	}

	return nil
}

// GetByID retrieves a memory by ID
func (r *MemoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*memory.Memory, error) {
	query := `
		SELECT 
			id, scope, user_id, agent_id, session_id, type, content,
			embedding, embedding_model, embedding_dimensions,
			validation_score, validation_trades, validated_at,
			symbol, timeframe, importance, tags, metadata,
			personality, source_user_id, source_trade_id,
			created_at, updated_at, expires_at
		FROM memories
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)
	m, err := scanMemory(row)
	if err == sql.ErrNoRows {
		return nil, pkgerrors.Wrap(err, "memory not found")
	}
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to get memory")
	}

	return m, nil
}

// SearchSimilar performs semantic search for user memories
func (r *MemoryRepository) SearchSimilar(ctx context.Context, userID uuid.UUID, agentID string, embedding pgvector.Vector, limit int) ([]*memory.Memory, error) {
	query := `
		SELECT 
			id, scope, user_id, agent_id, session_id, type, content,
			embedding, embedding_model, embedding_dimensions,
			validation_score, validation_trades, validated_at,
			symbol, timeframe, importance, tags, metadata,
			personality, source_user_id, source_trade_id,
			created_at, updated_at, expires_at,
			1 - (embedding <=> $3) as similarity
		FROM memories
		WHERE scope = 'user'
			AND user_id = $1
			AND agent_id = $2
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY embedding <=> $3
		LIMIT $4`

	rows, err := r.db.QueryContext(ctx, query, userID, agentID, embedding, limit)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to search memories")
	}
	defer rows.Close()

	var memories []*memory.Memory
	for rows.Next() {
		m, err := scanMemoryWithSimilarity(rows)
		if err != nil {
			return nil, pkgerrors.Wrap(err, "failed to scan memory")
		}
		memories = append(memories, m)
	}

	if err := rows.Err(); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to iterate memories")
	}

	return memories, nil
}

// GetByAgent retrieves user memories for a specific agent
func (r *MemoryRepository) GetByAgent(ctx context.Context, userID uuid.UUID, agentID string, limit int) ([]*memory.Memory, error) {
	query := `
		SELECT 
			id, scope, user_id, agent_id, session_id, type, content,
			embedding, embedding_model, embedding_dimensions,
			validation_score, validation_trades, validated_at,
			symbol, timeframe, importance, tags, metadata,
			personality, source_user_id, source_trade_id,
			created_at, updated_at, expires_at
		FROM memories
		WHERE scope = 'user'
			AND user_id = $1
			AND agent_id = $2
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, userID, agentID, limit)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to get memories by agent")
	}
	defer rows.Close()

	var memories []*memory.Memory
	for rows.Next() {
		m, err := scanMemory(rows)
		if err != nil {
			return nil, pkgerrors.Wrap(err, "failed to scan memory")
		}
		memories = append(memories, m)
	}

	if err := rows.Err(); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to iterate memories")
	}

	return memories, nil
}

// GetByType retrieves user memories by type
func (r *MemoryRepository) GetByType(ctx context.Context, userID uuid.UUID, memType memory.MemoryType, limit int) ([]*memory.Memory, error) {
	query := `
		SELECT 
			id, scope, user_id, agent_id, session_id, type, content,
			embedding, embedding_model, embedding_dimensions,
			validation_score, validation_trades, validated_at,
			symbol, timeframe, importance, tags, metadata,
			personality, source_user_id, source_trade_id,
			created_at, updated_at, expires_at
		FROM memories
		WHERE scope = 'user'
			AND user_id = $1
			AND type = $2
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY importance DESC, created_at DESC
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, userID, memType, limit)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to get memories by type")
	}
	defer rows.Close()

	var memories []*memory.Memory
	for rows.Next() {
		m, err := scanMemory(rows)
		if err != nil {
			return nil, pkgerrors.Wrap(err, "failed to scan memory")
		}
		memories = append(memories, m)
	}

	if err := rows.Err(); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to iterate memories")
	}

	return memories, nil
}

// DeleteExpired removes expired memories
func (r *MemoryRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM memories 
		WHERE expires_at IS NOT NULL AND expires_at < NOW()
	`)
	if err != nil {
		return 0, pkgerrors.Wrap(err, "failed to delete expired memories")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, pkgerrors.Wrap(err, "failed to get rows affected")
	}

	return rows, nil
}

// SearchCollectiveSimilar performs semantic search for collective memories
func (r *MemoryRepository) SearchCollectiveSimilar(ctx context.Context, embedding pgvector.Vector, agentID string, limit int) ([]*memory.Memory, error) {
	query := `
		SELECT 
			id, scope, user_id, agent_id, session_id, type, content,
			embedding, embedding_model, embedding_dimensions,
			validation_score, validation_trades, validated_at,
			symbol, timeframe, importance, tags, metadata,
			personality, source_user_id, source_trade_id,
			created_at, updated_at, expires_at,
			1 - (embedding <=> $2) as similarity
		FROM memories
		WHERE scope = 'collective'
			AND agent_id = $1
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY embedding <=> $2
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, agentID, embedding, limit)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to search collective memories")
	}
	defer rows.Close()

	var memories []*memory.Memory
	for rows.Next() {
		m, err := scanMemoryWithSimilarity(rows)
		if err != nil {
			return nil, pkgerrors.Wrap(err, "failed to scan memory")
		}
		memories = append(memories, m)
	}

	if err := rows.Err(); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to iterate memories")
	}

	return memories, nil
}

// SearchCollectiveByPersonality searches collective memories for a specific personality type
func (r *MemoryRepository) SearchCollectiveByPersonality(ctx context.Context, embedding pgvector.Vector, agentID string, personality string, limit int) ([]*memory.Memory, error) {
	query := `
		SELECT 
			id, scope, user_id, agent_id, session_id, type, content,
			embedding, embedding_model, embedding_dimensions,
			validation_score, validation_trades, validated_at,
			symbol, timeframe, importance, tags, metadata,
			personality, source_user_id, source_trade_id,
			created_at, updated_at, expires_at,
			1 - (embedding <=> $3) as similarity
		FROM memories
		WHERE scope = 'collective'
			AND agent_id = $1
			AND (personality = $2 OR personality IS NULL)
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY embedding <=> $3
		LIMIT $4`

	rows, err := r.db.QueryContext(ctx, query, agentID, personality, embedding, limit)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to search collective memories by personality")
	}
	defer rows.Close()

	var memories []*memory.Memory
	for rows.Next() {
		m, err := scanMemoryWithSimilarity(rows)
		if err != nil {
			return nil, pkgerrors.Wrap(err, "failed to scan memory")
		}
		memories = append(memories, m)
	}

	if err := rows.Err(); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to iterate memories")
	}

	return memories, nil
}

// GetValidated returns only validated collective memories with high confidence
func (r *MemoryRepository) GetValidated(ctx context.Context, agentID string, minScore float64, limit int) ([]*memory.Memory, error) {
	query := `
		SELECT 
			id, scope, user_id, agent_id, session_id, type, content,
			embedding, embedding_model, embedding_dimensions,
			validation_score, validation_trades, validated_at,
			symbol, timeframe, importance, tags, metadata,
			personality, source_user_id, source_trade_id,
			created_at, updated_at, expires_at
		FROM memories
		WHERE scope = 'collective'
			AND agent_id = $1
			AND validated_at IS NOT NULL
			AND validation_score >= $2
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY validation_score DESC, validation_trades DESC, importance DESC
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, agentID, minScore, limit)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to get validated memories")
	}
	defer rows.Close()

	var memories []*memory.Memory
	for rows.Next() {
		m, err := scanMemory(rows)
		if err != nil {
			return nil, pkgerrors.Wrap(err, "failed to scan memory")
		}
		memories = append(memories, m)
	}

	if err := rows.Err(); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to iterate memories")
	}

	return memories, nil
}

// GetCollectiveByType returns collective memories filtered by type
func (r *MemoryRepository) GetCollectiveByType(ctx context.Context, agentID string, memType memory.MemoryType, limit int) ([]*memory.Memory, error) {
	query := `
		SELECT 
			id, scope, user_id, agent_id, session_id, type, content,
			embedding, embedding_model, embedding_dimensions,
			validation_score, validation_trades, validated_at,
			symbol, timeframe, importance, tags, metadata,
			personality, source_user_id, source_trade_id,
			created_at, updated_at, expires_at
		FROM memories
		WHERE scope = 'collective'
			AND agent_id = $1
			AND type = $2
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY validation_score DESC, created_at DESC
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, agentID, memType, limit)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to get collective memories by type")
	}
	defer rows.Close()

	var memories []*memory.Memory
	for rows.Next() {
		m, err := scanMemory(rows)
		if err != nil {
			return nil, pkgerrors.Wrap(err, "failed to scan memory")
		}
		memories = append(memories, m)
	}

	if err := rows.Err(); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to iterate memories")
	}

	return memories, nil
}

// UpdateValidation updates validation metrics for a collective memory
func (r *MemoryRepository) UpdateValidation(ctx context.Context, id uuid.UUID, score float64, tradeCount int) error {
	query := `
		UPDATE memories
		SET 
			validation_score = $2,
			validation_trades = $3,
			validated_at = CASE WHEN validated_at IS NULL THEN NOW() ELSE validated_at END,
			updated_at = NOW()
		WHERE id = $1 AND scope = 'collective'`

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
func (r *MemoryRepository) PromoteToCollective(ctx context.Context, userMemory *memory.Memory, validationScore float64, validationTrades int) (*memory.Memory, error) {
	now := time.Now()

	// Create collective memory from user memory
	collective := &memory.Memory{
		ID:                  uuid.New(),
		Scope:               memory.MemoryScopeCollective,
		UserID:              nil, // Collective memories have no user_id
		AgentID:             userMemory.AgentID,
		SessionID:           "",
		Type:                userMemory.Type,
		Content:             userMemory.Content,
		Embedding:           userMemory.Embedding,
		EmbeddingModel:      userMemory.EmbeddingModel,
		EmbeddingDimensions: userMemory.EmbeddingDimensions,
		ValidationScore:     validationScore,
		ValidationTrades:    validationTrades,
		ValidatedAt:         &now,
		Symbol:              userMemory.Symbol,
		Timeframe:           userMemory.Timeframe,
		Importance:          userMemory.Importance,
		Tags:                userMemory.Tags,
		Metadata:            userMemory.Metadata,
		Personality:         nil,
		SourceUserID:        userMemory.UserID,
		SourceTradeID:       nil,
		CreatedAt:           now,
		UpdatedAt:           now,
		ExpiresAt:           nil,
	}

	// Extract trade_id from metadata if present
	if userMemory.Metadata != nil {
		if tradeIDStr, ok := userMemory.Metadata["trade_id"].(string); ok && tradeIDStr != "" {
			if tradeID, err := uuid.Parse(tradeIDStr); err == nil {
				collective.SourceTradeID = &tradeID
			}
		}
	}

	// Store collective memory
	if err := r.Store(ctx, collective); err != nil {
		return nil, err
	}

	return collective, nil
}
