package seeds

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	"prometheus/internal/domain/memory"
)

// MemoryBuilder provides a fluent API for creating Memory entities
type MemoryBuilder struct {
	db     DBTX
	ctx    context.Context
	entity *memory.Memory
}

// NewMemoryBuilder creates a new MemoryBuilder with sensible defaults
func NewMemoryBuilder(db DBTX, ctx context.Context) *MemoryBuilder {
	now := time.Now()
	return &MemoryBuilder{
		db:  db,
		ctx: ctx,
		entity: &memory.Memory{
			ID:                  uuid.New(),
			Scope:               memory.MemoryScopeUser,
			UserID:              nil, // Can be nil for collective/working scope
			AgentID:             "test_agent",
			SessionID:           "",
			Type:                memory.MemoryObservation,
			Content:             "Test memory content",
			Embedding:           pgvector.Vector{},
			EmbeddingModel:      "text-embedding-3-small",
			EmbeddingDimensions: 1536,
			ValidationScore:     0.0,
			ValidationTrades:    0,
			ValidatedAt:         nil,
			Symbol:              "BTC/USDT",
			Timeframe:           "1h",
			Importance:          0.5,
			Tags:                []string{},
			Metadata:            map[string]interface{}{},
			Personality:         nil,
			SourceUserID:        nil,
			SourceTradeID:       nil,
			CreatedAt:           now,
			UpdatedAt:           now,
			ExpiresAt:           nil,
			Similarity:          0.0,
		},
	}
}

// WithID sets a specific ID
func (b *MemoryBuilder) WithID(id uuid.UUID) *MemoryBuilder {
	b.entity.ID = id
	return b
}

// WithScope sets the memory scope
func (b *MemoryBuilder) WithScope(scope memory.MemoryScope) *MemoryBuilder {
	b.entity.Scope = scope
	return b
}

// WithUserScope sets scope to user
func (b *MemoryBuilder) WithUserScope() *MemoryBuilder {
	b.entity.Scope = memory.MemoryScopeUser
	return b
}

// WithCollectiveScope sets scope to collective
func (b *MemoryBuilder) WithCollectiveScope() *MemoryBuilder {
	b.entity.Scope = memory.MemoryScopeCollective
	return b
}

// WithWorkingScope sets scope to working
func (b *MemoryBuilder) WithWorkingScope() *MemoryBuilder {
	b.entity.Scope = memory.MemoryScopeWorking
	return b
}

// WithUserID sets the user ID
func (b *MemoryBuilder) WithUserID(userID uuid.UUID) *MemoryBuilder {
	b.entity.UserID = &userID
	return b
}

// WithAgentID sets the agent ID
func (b *MemoryBuilder) WithAgentID(agentID string) *MemoryBuilder {
	b.entity.AgentID = agentID
	return b
}

// WithSessionID sets the session ID
func (b *MemoryBuilder) WithSessionID(sessionID string) *MemoryBuilder {
	b.entity.SessionID = sessionID
	return b
}

// WithType sets the memory type
func (b *MemoryBuilder) WithType(memType memory.MemoryType) *MemoryBuilder {
	b.entity.Type = memType
	return b
}

// WithObservation sets type to observation
func (b *MemoryBuilder) WithObservation() *MemoryBuilder {
	b.entity.Type = memory.MemoryObservation
	return b
}

// WithDecision sets type to decision
func (b *MemoryBuilder) WithDecision() *MemoryBuilder {
	b.entity.Type = memory.MemoryDecision
	return b
}

// WithTrade sets type to trade
func (b *MemoryBuilder) WithTrade() *MemoryBuilder {
	b.entity.Type = memory.MemoryTrade
	return b
}

// WithLesson sets type to lesson
func (b *MemoryBuilder) WithLesson() *MemoryBuilder {
	b.entity.Type = memory.MemoryLesson
	return b
}

// WithRegime sets type to regime
func (b *MemoryBuilder) WithRegime() *MemoryBuilder {
	b.entity.Type = memory.MemoryRegime
	return b
}

// WithPattern sets type to pattern
func (b *MemoryBuilder) WithPattern() *MemoryBuilder {
	b.entity.Type = memory.MemoryPattern
	return b
}

// WithContent sets the memory content
func (b *MemoryBuilder) WithContent(content string) *MemoryBuilder {
	b.entity.Content = content
	return b
}

// WithSymbol sets the symbol
func (b *MemoryBuilder) WithSymbol(symbol string) *MemoryBuilder {
	b.entity.Symbol = symbol
	return b
}

// WithTimeframe sets the timeframe
func (b *MemoryBuilder) WithTimeframe(timeframe string) *MemoryBuilder {
	b.entity.Timeframe = timeframe
	return b
}

// WithImportance sets the importance score
func (b *MemoryBuilder) WithImportance(importance float64) *MemoryBuilder {
	b.entity.Importance = importance
	return b
}

// WithEmbedding sets the embedding vector
func (b *MemoryBuilder) WithEmbedding(embedding pgvector.Vector) *MemoryBuilder {
	b.entity.Embedding = embedding
	return b
}

// WithEmbeddingModel sets the embedding model name
func (b *MemoryBuilder) WithEmbeddingModel(model string) *MemoryBuilder {
	b.entity.EmbeddingModel = model
	return b
}

// WithValidationScore sets the validation score
func (b *MemoryBuilder) WithValidationScore(score float64) *MemoryBuilder {
	b.entity.ValidationScore = score
	return b
}

// WithTags sets the tags
func (b *MemoryBuilder) WithTags(tags []string) *MemoryBuilder {
	b.entity.Tags = tags
	return b
}

// WithMetadata sets the metadata
func (b *MemoryBuilder) WithMetadata(metadata map[string]interface{}) *MemoryBuilder {
	b.entity.Metadata = metadata
	return b
}

// WithPersonality sets the personality filter
func (b *MemoryBuilder) WithPersonality(personality string) *MemoryBuilder {
	b.entity.Personality = &personality
	return b
}

// WithSourceUser sets the source user ID
func (b *MemoryBuilder) WithSourceUser(userID uuid.UUID) *MemoryBuilder {
	b.entity.SourceUserID = &userID
	return b
}

// WithSourceTrade sets the source trade ID
func (b *MemoryBuilder) WithSourceTrade(tradeID uuid.UUID) *MemoryBuilder {
	b.entity.SourceTradeID = &tradeID
	return b
}

// WithExpiresAt sets the expiration time
func (b *MemoryBuilder) WithExpiresAt(expiresAt time.Time) *MemoryBuilder {
	b.entity.ExpiresAt = &expiresAt
	return b
}

// Build returns the built entity without inserting to DB
func (b *MemoryBuilder) Build() *memory.Memory {
	return b.entity
}

// Insert inserts the memory into the database and returns the entity
func (b *MemoryBuilder) Insert() (*memory.Memory, error) {
	query := `
		INSERT INTO memories (
			id, scope, user_id, agent_id, session_id, type, content,
			embedding, embedding_model, embedding_dimensions,
			validation_score, validation_trades, validated_at,
			symbol, timeframe, importance, tags, metadata,
			personality, source_user_id, source_trade_id,
			created_at, updated_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
	`

	_, err := b.db.ExecContext(
		b.ctx,
		query,
		b.entity.ID,
		b.entity.Scope,
		b.entity.UserID,
		b.entity.AgentID,
		b.entity.SessionID,
		b.entity.Type,
		b.entity.Content,
		b.entity.Embedding,
		b.entity.EmbeddingModel,
		b.entity.EmbeddingDimensions,
		b.entity.ValidationScore,
		b.entity.ValidationTrades,
		b.entity.ValidatedAt,
		b.entity.Symbol,
		b.entity.Timeframe,
		b.entity.Importance,
		b.entity.Tags,
		b.entity.Metadata,
		b.entity.Personality,
		b.entity.SourceUserID,
		b.entity.SourceTradeID,
		b.entity.CreatedAt,
		b.entity.UpdatedAt,
		b.entity.ExpiresAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to insert memory: %w", err)
	}

	return b.entity, nil
}

// MustInsert inserts the memory and panics on error (useful for tests)
func (b *MemoryBuilder) MustInsert() *memory.Memory {
	entity, err := b.Insert()
	if err != nil {
		panic(err)
	}
	return entity
}
