package memory

import (
	"time"

	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"google.golang.org/adk/tool"

	memorydomain "prometheus/internal/domain/memory"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
	"prometheus/pkg/templates"
)

// searchMemoryArgs represents input parameters for memory search
type searchMemoryArgs struct {
	UserID uuid.UUID
	Query  string // Text query for semantic search
	Limit  int
}

// NewSearchMemoryTool performs semantic memory search for a user
func NewSearchMemoryTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"search_memory",
		"Search past memories using semantic search. Provide a text query describing what you're looking for.",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			if deps.MemoryRepo == nil {
				return nil, errors.Wrapf(errors.ErrInternal, "search_memory: memory repository not configured")
			}
			if deps.EmbeddingProvider == nil {
				return nil, errors.Wrapf(errors.ErrInternal, "search_memory: embedding provider not configured")
			}

			// Parse and validate input arguments
			searchArgs, err := parseSearchMemoryArgs(ctx, args)
			if err != nil {
				return nil, errors.Wrap(err, "search_memory")
			}

			// Execute search with text query (service generates embedding internally)
			service := memorydomain.NewService(deps.MemoryRepo, deps.EmbeddingProvider)
			results, err := service.SearchSimilar(ctx, searchArgs.UserID, searchArgs.Query, searchArgs.Limit)
			if err != nil {
				return nil, err
			}

			// Prepare data for template
			templateData := prepareTemplateData(results, searchArgs.Query)

			// Render using template
			tmpl := templates.Get()
			formatted, err := tmpl.Render("tools/search_memory", templateData)
			if err != nil {
				return nil, errors.Wrap(err, "failed to render search_memory template")
			}

			return map[string]interface{}{
				"results": formatted,
			}, nil
		},
		deps,
	).
		WithTimeout(15*time.Second). // Increased for embedding generation
		WithRetry(3, 500*time.Millisecond).
		Build()
}

// parseSearchMemoryArgs extracts and validates input arguments
func parseSearchMemoryArgs(ctx tool.Context, args map[string]interface{}) (*searchMemoryArgs, error) {
	result := &searchMemoryArgs{
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
		return nil, errors.Wrapf(errors.ErrInvalidInput, "user_id is required")
	}

	// Parse query text
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "query text is required")
	}
	result.Query = query

	// Parse limit
	if l, ok := args["limit"].(float64); ok && int(l) > 0 {
		result.Limit = int(l)
	}

	return result, nil
}

// prepareTemplateData converts domain memories to template-ready data
func prepareTemplateData(memories []*memorydomain.Memory, query string) map[string]interface{} {
	formattedMemories := make([]map[string]interface{}, 0, len(memories))

	for _, m := range memories {
		// Format importance as stars
		importanceStars := formatImportanceStars(m.Importance)
		importanceLabel := formatImportanceLabel(m.Importance)

		// Calculate time ago
		createdAgo := humanize.Time(m.CreatedAt)

		formattedMemories = append(formattedMemories, map[string]interface{}{
			"id":               m.ID.String(),
			"agent_id":         m.AgentID,
			"content":          m.Content,
			"symbol":           m.Symbol,
			"timeframe":        m.Timeframe,
			"importance":       m.Importance,
			"importance_stars": importanceStars,
			"importance_label": importanceLabel,
			"type":             string(m.Type),
			"created_at":       m.CreatedAt.Format("2006-01-02 15:04 MST"),
			"created_ago":      createdAgo,
		})
	}

	return map[string]interface{}{
		"Memories":    formattedMemories,
		"ResultCount": len(memories),
		"Query":       query,
		"Timestamp":   time.Now().UTC().Format(time.RFC3339),
	}
}

// formatImportanceStars returns star representation of importance (0.0 - 1.0)
func formatImportanceStars(importance float64) string {
	stars := int(importance * 5) // 0-5 stars
	if stars > 5 {
		stars = 5
	}
	result := ""
	for i := 0; i < stars; i++ {
		result += "⭐"
	}
	if stars == 0 {
		result = "☆"
	}
	return result
}

// formatImportanceLabel returns human-readable importance label
func formatImportanceLabel(importance float64) string {
	switch {
	case importance >= 0.8:
		return "Critical"
	case importance >= 0.6:
		return "High"
	case importance >= 0.4:
		return "Medium"
	case importance >= 0.2:
		return "Low"
	default:
		return "Minimal"
	}
}
