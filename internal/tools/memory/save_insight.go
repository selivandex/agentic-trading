package memory

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	memorydomain "prometheus/internal/domain/memory"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
)

// SaveInsightArgs represents input parameters for saving insights/learnings
type SaveInsightArgs struct {
	UserID     uuid.UUID `json:"user_id"`
	AgentType  string    `json:"agent_type"`
	Category   string    `json:"category"` // "market_pattern", "risk_lesson", "execution_tip", etc.
	Insight    string    `json:"insight"`
	Confidence float64   `json:"confidence"` // 0-1, how confident in this insight
	Symbol     string    `json:"symbol,omitempty"`
	Timeframe  string    `json:"timeframe,omitempty"`
	SessionID  string    `json:"session_id,omitempty"`
}

// SaveInsightResponse represents the response from save_insight
type SaveInsightResponse struct {
	Status    string `json:"status"`
	ID        string `json:"id"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// NewSaveInsightTool creates a tool for agents to record learnings and patterns.
// Insights are stored as "lesson" or "pattern" type memories for long-term learning.
func NewSaveInsightTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"save_insight",
		"Save a learning, pattern, or insight to long-term memory. Use when you discover something worth remembering for future decisions.",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			if deps.MemoryRepo == nil {
				return nil, errors.Wrapf(errors.ErrInternal, "save_insight: memory repository not configured")
			}

			// Parse and validate input arguments
			saveArgs, err := parseSaveInsightArgs(ctx, args)
			if err != nil {
				return nil, errors.Wrap(err, "save_insight")
			}

			// Determine memory type based on category
			memType := categorizeInsight(saveArgs.Category)

			// Generate embedding for semantic search
			var embedding pgvector.Vector
			if deps.EmbeddingService != nil {
				// Create searchable text combining category and insight
				searchText := fmt.Sprintf("Category: %s, Insight: %s", saveArgs.Category, saveArgs.Insight)

				embeddingVec, err := deps.EmbeddingService.GenerateEmbedding(ctx, searchText)
				if err != nil {
					return nil, errors.Wrap(err, "failed to generate embedding")
				}
				embedding = pgvector.NewVector(embeddingVec)
			}

			// Create memory entry
			memory := &memorydomain.Memory{
				ID:         uuid.New(),
				UserID:     saveArgs.UserID,
				AgentID:    saveArgs.AgentType,
				SessionID:  saveArgs.SessionID,
				Type:       memType,
				Content:    saveArgs.Insight,
				Embedding:  embedding,
				Symbol:     saveArgs.Symbol,
				Timeframe:  saveArgs.Timeframe,
				Importance: saveArgs.Confidence, // Confidence maps to importance
				CreatedAt:  time.Now(),
				// Insights don't expire by default (long-term memory)
				ExpiresAt: nil,
			}

			// Store memory
			service := memorydomain.NewService(deps.MemoryRepo)
			if err := service.Store(ctx, memory); err != nil {
				return nil, errors.Wrap(err, "failed to store insight")
			}

			// Build response
			response := SaveInsightResponse{
				Status:    "saved",
				ID:        memory.ID.String(),
				Message:   "Insight successfully saved to long-term memory",
				Timestamp: memory.CreatedAt.Format(time.RFC3339),
			}

			return map[string]interface{}{
				"status":    response.Status,
				"id":        response.ID,
				"message":   response.Message,
				"timestamp": response.Timestamp,
			}, nil
		},
		deps,
	).
		WithTimeout(5*time.Second).
		WithRetry(2, 500*time.Millisecond).
		WithStats().
		Build()
}

// parseSaveInsightArgs extracts and validates input arguments
func parseSaveInsightArgs(ctx tool.Context, args map[string]interface{}) (*SaveInsightArgs, error) {
	result := &SaveInsightArgs{
		Confidence: 0.5, // default confidence
	}

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

	// Parse agent_type
	if at, ok := args["agent_type"].(string); ok {
		result.AgentType = at
	}
	if result.AgentType == "" {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "agent_type is required")
	}

	// Parse category
	if cat, ok := args["category"].(string); ok {
		result.Category = cat
	}
	if result.Category == "" {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "category is required")
	}

	// Parse insight
	if ins, ok := args["insight"].(string); ok {
		result.Insight = ins
	}
	if result.Insight == "" {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "insight is required")
	}

	// Parse confidence
	if conf, ok := args["confidence"].(float64); ok {
		result.Confidence = conf
	}

	// Validate confidence range
	if result.Confidence < 0.0 || result.Confidence > 1.0 {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "confidence must be between 0 and 1")
	}

	// Parse optional fields
	if sym, ok := args["symbol"].(string); ok {
		result.Symbol = sym
	}
	if tf, ok := args["timeframe"].(string); ok {
		result.Timeframe = tf
	}
	if sid, ok := args["session_id"].(string); ok {
		result.SessionID = sid
	}

	return result, nil
}

// categorizeInsight maps insight categories to memory types
func categorizeInsight(category string) memorydomain.MemoryType {
	switch category {
	case "market_pattern", "price_pattern", "technical_pattern":
		return memorydomain.MemoryPattern
	case "risk_lesson", "execution_lesson", "strategy_lesson":
		return memorydomain.MemoryLesson
	case "market_regime", "volatility_regime":
		return memorydomain.MemoryRegime
	default:
		return memorydomain.MemoryLesson // Default to lesson
	}
}
