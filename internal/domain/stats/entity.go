package stats

import (
	"time"

	"github.com/google/uuid"
)

// ToolUsageEvent represents a single tool call event (for insertion)
type ToolUsageEvent struct {
	UserID    uuid.UUID `ch:"user_id"`
	AgentID   string    `ch:"agent_id"`
	ToolName  string    `ch:"tool_name"`
	Timestamp time.Time `ch:"timestamp"`

	DurationMs int  `ch:"duration_ms"`
	Success    bool `ch:"success"`

	SessionID string `ch:"session_id"`
	Symbol    string `ch:"symbol"`
}

// ToolUsageAggregated represents aggregated tool usage (from materialized view)
type ToolUsageAggregated struct {
	UserID   uuid.UUID `ch:"user_id"`
	AgentID  string    `ch:"agent_id"`
	ToolName string    `ch:"tool_name"`
	Hour     time.Time `ch:"hour"`

	CallCount       uint64  `ch:"call_count"`
	TotalDurationMs uint64  `ch:"total_duration_ms"`
	SuccessCount    uint64  `ch:"success_count"`
	ErrorCount      uint64  `ch:"error_count"`
	AvgDurationMs   float64 `ch:"avg_duration_ms"`
}
