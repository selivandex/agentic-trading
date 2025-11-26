package memory

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	memorydomain "prometheus/internal/domain/memory"
	"prometheus/internal/tools/shared"

	"google.golang.org/adk/tool/functiontool"
)

// NewSearchMemoryTool performs semantic memory search for a user.
func NewSearchMemoryTool(deps shared.Deps) *functiontool.Tool {
	return functiontool.New("search_memory", "Semantic memory search", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		if deps.MemoryRepo == nil {
			return nil, fmt.Errorf("search_memory: memory repository not configured")
		}

		userID := uuid.Nil
		if idVal, ok := args["user_id"].(string); ok {
			parsed, err := uuid.Parse(idVal)
			if err == nil {
				userID = parsed
			}
		}
		if userID == uuid.Nil {
			if meta, ok := shared.MetadataFromContext(ctx); ok {
				userID = meta.UserID
			}
		}
		if userID == uuid.Nil {
			return nil, fmt.Errorf("search_memory: user_id is required")
		}

		rawEmbedding, ok := args["embedding"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("search_memory: embedding is required")
		}
		vectorValues := make([]float32, 0, len(rawEmbedding))
		for _, v := range rawEmbedding {
			switch val := v.(type) {
			case float64:
				vectorValues = append(vectorValues, float32(val))
			case float32:
				vectorValues = append(vectorValues, val)
			default:
				return nil, fmt.Errorf("search_memory: embedding must be numeric array")
			}
		}
		vector := pgvector.NewVector(vectorValues)

		limit := 5
		if l, ok := args["limit"].(float64); ok && int(l) > 0 {
			limit = int(l)
		}

		service := memorydomain.NewService(deps.MemoryRepo)
		results, err := service.SearchSimilar(ctx, userID, vector, limit)
		if err != nil {
			return nil, err
		}

		data := make([]map[string]interface{}, 0, len(results))
		for _, m := range results {
			data = append(data, map[string]interface{}{
				"id":         m.ID.String(),
				"agent_id":   m.AgentID,
				"content":    m.Content,
				"metadata":   m.Metadata,
				"created_at": m.CreatedAt,
			})
		}

		return map[string]interface{}{"memories": data}, nil
	})
}
