package memory

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// Memory represents an agent's memory entry with vector embedding
type Memory struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	AgentID   string    `db:"agent_id"`
	SessionID string    `db:"session_id"`

	Type    MemoryType `db:"type"` // observation, decision, trade, lesson
	Content string     `db:"content"`

	// Embedding metadata (critical for search compatibility)
	Embedding           pgvector.Vector `db:"embedding"` // pgvector handles this automatically
	EmbeddingModel      string          `db:"embedding_model"`
	EmbeddingDimensions int             `db:"embedding_dimensions"`

	// Trading metadata (kept as columns for fast filtering)
	Symbol     string  `db:"symbol"`
	Timeframe  string  `db:"timeframe"`
	Importance float64 `db:"importance"` // 0-1, for retrieval ranking

	// Flexible metadata storage (tags, references, custom fields)
	// Examples: trade_id, position_id, tags, confidence, related_memories
	Metadata map[string]interface{} `db:"metadata"`

	CreatedAt time.Time  `db:"created_at"`
	ExpiresAt *time.Time `db:"expires_at"` // TTL for short-term
}

// MemoryType defines the type of memory
type MemoryType string

const (
	MemoryObservation MemoryType = "observation" // Market observation
	MemoryDecision    MemoryType = "decision"    // Trade decision reasoning
	MemoryTrade       MemoryType = "trade"       // Trade outcome
	MemoryLesson      MemoryType = "lesson"      // Learned pattern
	MemoryRegime      MemoryType = "regime"      // Market regime
	MemoryPattern     MemoryType = "pattern"     // Detected pattern
)

// Valid checks if memory type is valid
func (m MemoryType) Valid() bool {
	switch m {
	case MemoryObservation, MemoryDecision, MemoryTrade, MemoryLesson, MemoryRegime, MemoryPattern:
		return true
	}
	return false
}

// String returns string representation
func (m MemoryType) String() string {
	return string(m)
}

// CollectiveMemory represents shared knowledge validated across users
type CollectiveMemory struct {
	ID uuid.UUID `db:"id"`

	// Scope
	AgentType   string  `db:"agent_type"`  // "market_analyst", "risk_manager", etc.
	Personality *string `db:"personality"` // NULL for all personalities

	// Content
	Type      MemoryType      `db:"type"`
	Content   string          `db:"content"`
	Embedding pgvector.Vector `db:"embedding"`

	// Validation (promoted only when proven)
	ValidationScore  float64    `db:"validation_score"`  // 0-100, how well this lesson performed
	ValidationTrades int        `db:"validation_trades"` // Number of trades that validated this
	ValidatedAt      *time.Time `db:"validated_at"`

	// Metadata
	Symbol     *string  `db:"symbol"`     // NULL for general lessons
	Timeframe  *string  `db:"timeframe"`  // NULL for general lessons
	Importance float64  `db:"importance"` // 0-1, for retrieval ranking
	Tags       []string `db:"tags"`       // Additional categorization

	// Source (anonymized)
	SourceUserID  *uuid.UUID `db:"source_user_id"`  // Who contributed this
	SourceTradeID *uuid.UUID `db:"source_trade_id"` // Which trade led to this lesson

	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	ExpiresAt *time.Time `db:"expires_at"` // TTL for time-sensitive lessons
}
