package memory

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/adk/tool"

	memorydomain "prometheus/internal/domain/memory"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
)

// NewSaveMemoryTool creates a universal tool for saving any type of memory.
// Replaces: save_analysis, save_insight.
// Type parameter determines what kind of memory to save:
// - "analysis": structured analysis results (JSON)
// - "insight": learning/pattern (text)
// - "observation": market observation (text)
func NewSaveMemoryTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"save_memory",
		"Save information to long-term memory. Types: 'analysis' (structured results), 'insight' (learnings/patterns), 'observation' (market notes).",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			if deps.MemoryRepo == nil {
				return nil, errors.Wrapf(errors.ErrInternal, "memory repository not configured")
			}

			// Get user ID
			userID := uuid.Nil
			if idVal, ok := args["user_id"].(string); ok {
				if parsed, err := uuid.Parse(idVal); err == nil {
					userID = parsed
				}
			}
			if userID == uuid.Nil {
				if meta, ok := shared.MetadataFromContext(ctx); ok {
					userID = meta.UserID
				}
			}
			if userID == uuid.Nil {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "user_id is required")
			}

			// Get memory type
			memoryType, _ := args["type"].(string)
			if memoryType == "" {
				memoryType = "observation" // default
			}

			// Get agent ID
			agentID, _ := args["agent_id"].(string)
			if agentID == "" {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "agent_id is required")
			}

			// Get content (text or structured)
			content := ""

			switch memoryType {
			case "analysis":
				// Structured JSON content
				if analysis, ok := args["content"].(map[string]interface{}); ok {
					analysisJSON, err := json.Marshal(analysis)
					if err != nil {
						return nil, errors.Wrap(err, "failed to marshal analysis")
					}
					content = string(analysisJSON)
				} else {
					return nil, errors.Wrapf(errors.ErrInvalidInput, "content must be object for type=analysis")
				}

			case "insight", "observation":
				// Text content
				if text, ok := args["content"].(string); ok {
					content = text
				} else {
					return nil, errors.Wrapf(errors.ErrInvalidInput, "content must be string for type=%s", memoryType)
				}

			default:
				return nil, errors.Wrapf(errors.ErrInvalidInput, "invalid type: must be 'analysis', 'insight', or 'observation'")
			}

			if content == "" {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "content is required")
			}

			// Optional fields
			symbol, _ := args["symbol"].(string)
			timeframe, _ := args["timeframe"].(string)
			sessionID, _ := args["session_id"].(string)
			category, _ := args["category"].(string)
			importance := 0.5
			if imp, ok := args["importance"].(float64); ok && imp >= 0 && imp <= 1 {
				importance = imp
			}

			// Determine memory domain type
			domainType := categorizeMemoryType(memoryType, category)

			// Prepare search text for embedding (add context for better retrieval)
			searchText := content
			if memoryType == "analysis" && symbol != "" {
				searchText = fmt.Sprintf("Symbol: %s, Analysis: %s", symbol, content)
			} else if category != "" {
				searchText = fmt.Sprintf("Category: %s, Content: %s", category, content)
			}

			// Build metadata map for flexible storage
			metadata := make(map[string]interface{})
			if category != "" {
				metadata["category"] = category
			}
			// Extract optional references from args
			if tradeID, ok := args["trade_id"].(string); ok && tradeID != "" {
				metadata["trade_id"] = tradeID
			}
			if positionID, ok := args["position_id"].(string); ok && positionID != "" {
				metadata["position_id"] = positionID
			}
			if tags, ok := args["tags"].([]interface{}); ok && len(tags) > 0 {
				metadata["tags"] = tags
			}

			// Create memory entry (embedding will be generated in service)
			memory := &memorydomain.Memory{
				ID:         uuid.New(),
				UserID:     userID,
				AgentID:    agentID,
				SessionID:  sessionID,
				Type:       domainType,
				Content:    content,
				Symbol:     symbol,
				Timeframe:  timeframe,
				Importance: importance,
				Metadata:   metadata,
				CreatedAt:  time.Now(),
				ExpiresAt:  nil, // Long-term memory
			}

			// Store memory with automatic embedding generation
			service := memorydomain.NewService(deps.MemoryRepo, deps.EmbeddingProvider)
			if err := service.StoreWithText(ctx, memory, searchText); err != nil {
				return nil, errors.Wrap(err, "failed to store memory")
			}

			return map[string]interface{}{
				"status":    "saved",
				"id":        memory.ID.String(),
				"type":      memoryType,
				"message":   fmt.Sprintf("%s successfully saved to memory", memoryType),
				"timestamp": memory.CreatedAt.Format(time.RFC3339),
			}, nil
		},
		deps,
	).
		WithTimeout(5*time.Second).
		WithRetry(2, 500*time.Millisecond).
		Build()
}

// categorizeMemoryType maps memory type and category to domain type
func categorizeMemoryType(memType, category string) memorydomain.MemoryType {
	switch memType {
	case "analysis":
		return memorydomain.MemoryDecision

	case "insight":
		// Map by category
		switch category {
		case "market_pattern", "price_pattern", "technical_pattern":
			return memorydomain.MemoryPattern
		case "risk_lesson", "execution_lesson", "strategy_lesson":
			return memorydomain.MemoryLesson
		default:
			return memorydomain.MemoryLesson
		}

	case "observation":
		return memorydomain.MemoryObservation

	default:
		return memorydomain.MemoryObservation
	}
}
