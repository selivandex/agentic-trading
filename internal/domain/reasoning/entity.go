package reasoning

import (
	"time"

	"github.com/google/uuid"
)

// LogEntry represents an agent's reasoning log with Chain-of-Thought steps
type LogEntry struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	AgentID   string    `db:"agent_id"`
	SessionID string    `db:"session_id"`

	// Context
	Symbol        string     `db:"symbol"`
	TradingPairID *uuid.UUID `db:"trading_pair_id"`

	// Reasoning steps (stored as JSONB)
	ReasoningSteps []byte `db:"reasoning_steps"` // JSON array of steps

	// Final decision
	Decision   []byte  `db:"decision"`   // JSON
	Confidence float64 `db:"confidence"` // 0-100

	// Performance metrics
	TokensUsed     int     `db:"tokens_used"`
	CostUSD        float64 `db:"cost_usd"`
	DurationMs     int     `db:"duration_ms"`
	ToolCallsCount int     `db:"tool_calls_count"`

	CreatedAt time.Time `db:"created_at"`
}

// Step represents a single reasoning step
type Step struct {
	Step      int                    `json:"step"`
	Action    string                 `json:"action"` // "thinking", "tool_call", "decision"
	Tool      string                 `json:"tool,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Output    map[string]interface{} `json:"output,omitempty"`
	Content   string                 `json:"content,omitempty"` // For "thinking" and "decision"
	Timestamp time.Time              `json:"timestamp"`
}
