package memory

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	memorydomain "prometheus/internal/domain/memory"
	reasoningdomain "prometheus/internal/domain/reasoning"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
)

// RecordReasoningArgs represents input parameters for recording a reasoning step
type RecordReasoningArgs struct {
	UserID    uuid.UUID              `json:"user_id"`
	AgentID   string                 `json:"agent_id"`
	SessionID string                 `json:"session_id"`
	Step      int                    `json:"step"`
	Action    string                 `json:"action"` // "thinking", "tool_call", "decision"
	Content   string                 `json:"content,omitempty"`
	Tool      string                 `json:"tool,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Output    map[string]interface{} `json:"output,omitempty"`
}

// RecordReasoningResponse represents the response from record_reasoning
type RecordReasoningResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Step      int    `json:"step"`
	Timestamp string `json:"timestamp"`
}

// NewRecordReasoningTool creates a tool for agents to explicitly record reasoning steps.
// Note: This may be redundant with CoTWrapper, but provides explicit control for agents.
func NewRecordReasoningTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"record_reasoning",
		"Record a step in your reasoning process. Use to explicitly log important decision points or insights during analysis.",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			if deps.ReasoningRepo == nil {
				return nil, errors.Wrapf(errors.ErrInternal, "record_reasoning: reasoning repository not configured")
			}

			// Parse and validate input arguments
			recArgs, err := parseRecordReasoningArgs(ctx, args)
			if err != nil {
				return nil, errors.Wrap(err, "record_reasoning")
			}

			// Create reasoning step
			step := reasoningdomain.Step{
				Step:      recArgs.Step,
				Action:    recArgs.Action,
				Content:   recArgs.Content,
				Tool:      recArgs.Tool,
				Input:     recArgs.Input,
				Output:    recArgs.Output,
				Timestamp: time.Now(),
			}

			// For now, we'll store this as an observation in memory
			// The full reasoning log will be assembled by CoTWrapper
			// This tool allows agents to explicitly mark important steps

			// Convert step to JSON for storage
			stepJSON, err := json.Marshal(step)
			if err != nil {
				return nil, errors.Wrap(err, "failed to marshal reasoning step")
			}

			// Generate embedding for semantic search
			var embedding pgvector.Vector
			if deps.EmbeddingService != nil {
				// Create searchable text from reasoning step
				searchText := fmt.Sprintf("Action: %s, Content: %s", recArgs.Action, recArgs.Content)
				if recArgs.Tool != "" {
					searchText = fmt.Sprintf("%s, Tool: %s", searchText, recArgs.Tool)
				}

				embeddingVec, err := deps.EmbeddingService.GenerateEmbedding(ctx, searchText)
				if err != nil {
					return nil, errors.Wrap(err, "failed to generate embedding")
				}
				embedding = pgvector.NewVector(embeddingVec)
			}

			// Store in memory repo as observation (meta-reasoning)
			// This creates a searchable record of important reasoning steps
			memory := &memorydomain.Memory{
				ID:         uuid.New(),
				UserID:     recArgs.UserID,
				AgentID:    recArgs.AgentID,
				SessionID:  recArgs.SessionID,
				Type:       memorydomain.MemoryObservation,
				Content:    string(stepJSON),
				Embedding:  embedding,
				Importance: 0.6, // Reasoning steps have moderate importance
				CreatedAt:  time.Now(),
			}

			service := memorydomain.NewService(deps.MemoryRepo)
			if err := service.Store(ctx, memory); err != nil {
				return nil, errors.Wrap(err, "failed to store reasoning step")
			}

			// Build response
			response := RecordReasoningResponse{
				Status:    "recorded",
				Message:   "Reasoning step successfully recorded",
				Step:      recArgs.Step,
				Timestamp: step.Timestamp.Format(time.RFC3339),
			}

			return map[string]interface{}{
				"status":    response.Status,
				"message":   response.Message,
				"step":      response.Step,
				"timestamp": response.Timestamp,
			}, nil
		},
		deps,
	).
		WithTimeout(3*time.Second).
		WithRetry(2, 300*time.Millisecond).
		WithStats().
		Build()
}

// parseRecordReasoningArgs extracts and validates input arguments
func parseRecordReasoningArgs(ctx tool.Context, args map[string]interface{}) (*RecordReasoningArgs, error) {
	result := &RecordReasoningArgs{}

	// Parse user_id
	if idVal, ok := args["user_id"].(string); ok {
		parsed, err := uuid.Parse(idVal)
		if err != nil {
			return nil, errors.Wrap(err, "invalid user_id format")
		}
		result.UserID = parsed
	}

	// Try to get user_id from context if not provided
	if result.UserID == uuid.Nil {
		if meta, ok := shared.MetadataFromContext(ctx); ok {
			result.UserID = meta.UserID
		}
	}

	if result.UserID == uuid.Nil {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "user_id is required")
	}

	// Parse agent_id
	if aid, ok := args["agent_id"].(string); ok {
		result.AgentID = aid
	}
	if result.AgentID == "" {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "agent_id is required")
	}

	// Parse session_id
	if sid, ok := args["session_id"].(string); ok {
		result.SessionID = sid
	}
	if result.SessionID == "" {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "session_id is required")
	}

	// Parse step number
	if s, ok := args["step"].(float64); ok {
		result.Step = int(s)
	}
	if result.Step <= 0 {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "step must be a positive integer")
	}

	// Parse action
	if act, ok := args["action"].(string); ok {
		result.Action = act
	}
	if result.Action == "" {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "action is required")
	}

	// Validate action type
	switch result.Action {
	case "thinking", "tool_call", "decision":
		// valid
	default:
		return nil, errors.Wrapf(errors.ErrInvalidInput, "action must be one of: thinking, tool_call, decision")
	}

	// Parse optional fields
	if content, ok := args["content"].(string); ok {
		result.Content = content
	}
	if tool, ok := args["tool"].(string); ok {
		result.Tool = tool
	}
	if input, ok := args["input"].(map[string]interface{}); ok {
		result.Input = input
	}
	if output, ok := args["output"].(map[string]interface{}); ok {
		result.Output = output
	}

	return result, nil
}
