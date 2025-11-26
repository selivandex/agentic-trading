package ai_usage

import "time"

// UsageLog represents a single AI model usage event
type UsageLog struct {
	Timestamp time.Time `ch:"timestamp"`
	EventID   string    `ch:"event_id"`

	// User context
	UserID    string `ch:"user_id"`
	SessionID string `ch:"session_id"`

	// Agent context
	AgentName string `ch:"agent_name"`
	AgentType string `ch:"agent_type"`

	// Model details
	Provider    string `ch:"provider"`     // anthropic, openai, google, deepseek
	ModelID     string `ch:"model_id"`     // claude-sonnet-4, gpt-4o, etc.
	ModelFamily string `ch:"model_family"` // claude-3.5, gpt-4o, etc.

	// Token usage
	PromptTokens     uint32 `ch:"prompt_tokens"`
	CompletionTokens uint32 `ch:"completion_tokens"`
	TotalTokens      uint32 `ch:"total_tokens"`

	// Cost
	InputCostUSD  float64 `ch:"input_cost_usd"`
	OutputCostUSD float64 `ch:"output_cost_usd"`
	TotalCostUSD  float64 `ch:"total_cost_usd"`

	// Request metadata
	ToolCallsCount uint16 `ch:"tool_calls_count"`
	IsCached       bool   `ch:"is_cached"`
	CacheHit       bool   `ch:"cache_hit"`

	// Performance
	LatencyMs uint32 `ch:"latency_ms"`

	// Additional context
	ReasoningStep uint16 `ch:"reasoning_step"` // Which step in reasoning chain
	WorkflowName  string `ch:"workflow_name"`  // MarketResearchWorkflow, etc.

	CreatedAt time.Time `ch:"created_at"`
}
