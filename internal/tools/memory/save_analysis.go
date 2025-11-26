package memory

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	memorydomain "prometheus/internal/domain/memory"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
)

// SaveAnalysisArgs represents input parameters for saving analysis results
type SaveAnalysisArgs struct {
	UserID    uuid.UUID              `json:"user_id"`
	AgentType string                 `json:"agent_type"`
	Symbol    string                 `json:"symbol"`
	Timeframe string                 `json:"timeframe,omitempty"`
	Analysis  map[string]interface{} `json:"analysis"`
	SessionID string                 `json:"session_id,omitempty"`
}

// SaveAnalysisResponse represents the response from save_analysis
type SaveAnalysisResponse struct {
	Status    string `json:"status"`
	ID        string `json:"id"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// NewSaveAnalysisTool creates a tool for agents to persist their analysis results.
// This enables real Chain-of-Thought by allowing agents to save conclusions.
func NewSaveAnalysisTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"save_analysis",
		"Save analysis results to memory. Use this when you've completed your analysis and want to persist findings.",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			if deps.MemoryRepo == nil {
				return nil, errors.Wrapf(errors.ErrInternal, "save_analysis: memory repository not configured")
			}

			// Parse and validate input arguments
			saveArgs, err := parseSaveAnalysisArgs(ctx, args)
			if err != nil {
				return nil, errors.Wrap(err, "save_analysis")
			}

			// Convert analysis to JSON string for storage
			analysisJSON, err := json.Marshal(saveArgs.Analysis)
			if err != nil {
				return nil, errors.Wrap(err, "failed to marshal analysis")
			}

			// Generate embedding for semantic search
			var embedding pgvector.Vector
			if deps.EmbeddingService != nil {
				// Create searchable text from analysis
				searchText := fmt.Sprintf("Agent: %s, Symbol: %s, Analysis: %s",
					saveArgs.AgentType, saveArgs.Symbol, string(analysisJSON))

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
				Type:       memorydomain.MemoryDecision, // Analysis is a decision/conclusion
				Content:    string(analysisJSON),
				Embedding:  embedding,
				Symbol:     saveArgs.Symbol,
				Timeframe:  saveArgs.Timeframe,
				Importance: calculateImportance(saveArgs.Analysis),
				CreatedAt:  time.Now(),
			}

			// Store memory
			service := memorydomain.NewService(deps.MemoryRepo)
			if err := service.Store(ctx, memory); err != nil {
				return nil, errors.Wrap(err, "failed to store analysis")
			}

			// Build response
			response := SaveAnalysisResponse{
				Status:    "saved",
				ID:        memory.ID.String(),
				Message:   "Analysis successfully saved to memory",
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

// parseSaveAnalysisArgs extracts and validates input arguments
func parseSaveAnalysisArgs(ctx tool.Context, args map[string]interface{}) (*SaveAnalysisArgs, error) {
	result := &SaveAnalysisArgs{}

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

	// Parse symbol
	if sym, ok := args["symbol"].(string); ok {
		result.Symbol = sym
	}
	if result.Symbol == "" {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "symbol is required")
	}

	// Parse optional timeframe
	if tf, ok := args["timeframe"].(string); ok {
		result.Timeframe = tf
	}

	// Parse analysis (required, can be any structure)
	if analysis, ok := args["analysis"].(map[string]interface{}); ok {
		result.Analysis = analysis
	} else {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "analysis is required and must be an object")
	}

	// Parse optional session_id
	if sid, ok := args["session_id"].(string); ok {
		result.SessionID = sid
	}

	return result, nil
}

// calculateImportance estimates the importance of an analysis based on its content
func calculateImportance(analysis map[string]interface{}) float64 {
	// Base importance
	importance := 0.5

	// Higher importance for confident analysis
	if conf, ok := analysis["confidence"].(float64); ok {
		importance = conf
	}

	// Cap at 1.0
	if importance > 1.0 {
		importance = 1.0
	}

	return importance
}
