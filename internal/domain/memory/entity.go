package memory

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// MemoryScope defines the level of memory
type MemoryScope string

const (
	MemoryScopeUser       MemoryScope = "user"       // Agent-level memory for specific user (episodic)
	MemoryScopeCollective MemoryScope = "collective" // Agent-level shared knowledge across users (semantic)
	MemoryScopeWorking    MemoryScope = "working"    // Temporary working memory
)

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

// Memory represents unified memory entry (user, collective, or working)
type Memory struct {
	ID uuid.UUID `db:"id"`

	// Scope determines memory level
	Scope MemoryScope `db:"scope"`

	// User-specific (NULL for collective/working)
	UserID *uuid.UUID `db:"user_id"`

	// Agent that created memory
	AgentID   string `db:"agent_id"`
	SessionID string `db:"session_id"`

	// Memory classification
	Type    MemoryType `db:"type"`
	Content string     `db:"content"`

	// Vector embedding for semantic search
	Embedding           pgvector.Vector `db:"embedding"`
	EmbeddingModel      string          `db:"embedding_model"`
	EmbeddingDimensions int             `db:"embedding_dimensions"`

	// Validation metrics (for collective memories)
	ValidationScore  float64    `db:"validation_score"`
	ValidationTrades int        `db:"validation_trades"`
	ValidatedAt      *time.Time `db:"validated_at"`

	// Context metadata
	Symbol     string   `db:"symbol"`
	Timeframe  string   `db:"timeframe"`
	Importance float64  `db:"importance"`
	Tags       []string `db:"tags"`

	// Additional context (flexible JSON)
	Metadata map[string]interface{} `db:"metadata"`

	// Personality filter (for collective memories)
	Personality *string `db:"personality"`

	// Source tracking (for collective memories)
	SourceUserID  *uuid.UUID `db:"source_user_id"`
	SourceTradeID *uuid.UUID `db:"source_trade_id"`

	// Lifecycle
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	ExpiresAt *time.Time `db:"expires_at"`

	// Search result metadata (not stored in DB)
	Similarity float64 `db:"similarity"` // Cosine similarity score for search results
}

// IsUserMemory checks if this is agent-level memory for specific user
func (m *Memory) IsUserMemory() bool {
	return m.Scope == MemoryScopeUser
}

// IsCollectiveMemory checks if this is agent-level shared knowledge
func (m *Memory) IsCollectiveMemory() bool {
	return m.Scope == MemoryScopeCollective
}

// IsWorkingMemory checks if this is temporary working memory
func (m *Memory) IsWorkingMemory() bool {
	return m.Scope == MemoryScopeWorking
}
