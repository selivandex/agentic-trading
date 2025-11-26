package memory

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	memorydomain "prometheus/internal/domain/memory"
	"prometheus/internal/tools/shared"

	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// SearchMemoryArgs represents input parameters for memory search
type SearchMemoryArgs struct {
	UserID    uuid.UUID `json:"user_id"`
	Embedding []float32 `json:"embedding"`
	Limit     int       `json:"limit"`
}

// MemoryMetadata represents metadata associated with a memory
type MemoryMetadata struct {
	Symbol     string                  `json:"symbol"`
	Timeframe  string                  `json:"timeframe"`
	Importance float64                 `json:"importance"`
	Type       memorydomain.MemoryType `json:"type"`
}

// MemorySearchResult represents a single memory search result
type MemorySearchResult struct {
	ID        string         `json:"id"`
	AgentID   string         `json:"agent_id"`
	Content   string         `json:"content"`
	Metadata  MemoryMetadata `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
}

// MemorySearchResponse represents the response from memory search
type MemorySearchResponse struct {
	Memories []MemorySearchResult `json:"memories"`
}

// NewSearchMemoryTool performs semantic memory search for a user.
func NewSearchMemoryTool(deps shared.Deps) tool.Tool {
	return functiontool.New("search_memory", "Semantic memory search", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		if deps.MemoryRepo == nil {
			return nil, errors.Wrapf(errors.ErrInternal, "search_memory: memory repository not configured")
		}

		// Parse and validate input arguments with strong typing
		searchArgs, err := parseSearchMemoryArgs(ctx, args)
		if err != nil {
			return nil, errors.Wrap(err, "search_memory")
		}

		// Create vector from embedding
		vector := pgvector.NewVector(searchArgs.Embedding)

		// Execute search with typed parameters
		service := memorydomain.NewService(deps.MemoryRepo)
		results, err := service.SearchSimilar(ctx, searchArgs.UserID, vector, searchArgs.Limit)
		if err != nil {
			return nil, err
		}

		// Build strongly-typed response
		response := buildMemorySearchResponse(results)

		// Convert to map[string]interface{} for ADK compatibility
		// This is the only place where we lose type safety
		return map[string]interface{}{
			"memories": response.Memories,
		}, nil
	})
}

// parseSearchMemoryArgs extracts and validates input arguments
func parseSearchMemoryArgs(ctx context.Context, args map[string]interface{}) (*SearchMemoryArgs, error) {
	result := &SearchMemoryArgs{
		Limit: 5, // default
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
		return nil, errors.Wrapf(errors.ErrInternal, "user_id is required")
	}

	// Parse embedding
	rawEmbedding, ok := args["embedding"].([]interface{})
	if !ok {
		return nil, errors.Wrapf(errors.ErrInternal, "embedding is required")
	}

	result.Embedding = make([]float32, 0, len(rawEmbedding))
	for i, v := range rawEmbedding {
		switch val := v.(type) {
		case float64:
			result.Embedding = append(result.Embedding, float32(val))
		case float32:
			result.Embedding = append(result.Embedding, val)
		default:
			return nil, errors.Wrapf(errors.ErrInternal, "embedding[%d] must be numeric, got %T", i, v)
		}
	}

	// Parse limit
	if l, ok := args["limit"].(float64); ok && int(l) > 0 {
		result.Limit = int(l)
	}

	return result, nil
}

// buildMemorySearchResponse converts domain memories to typed response
func buildMemorySearchResponse(memories []*memorydomain.Memory) MemorySearchResponse {
	results := make([]MemorySearchResult, 0, len(memories))

	for _, m := range memories {
		results = append(results, MemorySearchResult{
			ID:      m.ID.String(),
			AgentID: m.AgentID,
			Content: m.Content,
			Metadata: MemoryMetadata{
				Symbol:     m.Symbol,
				Timeframe:  m.Timeframe,
				Importance: m.Importance,
				Type:       m.Type,
			},
			CreatedAt: m.CreatedAt,
		})
	}

	return MemorySearchResponse{
		Memories: results,
	}
}
