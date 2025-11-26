package shared

import (
	"context"
	"time"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"prometheus/internal/domain/stats"
)

// StatsMiddleware records tool usage metrics into the stats repository.
type StatsMiddleware struct {
	statsRepo stats.Repository
}

// NewStatsMiddleware constructs a middleware with the provided repository.
func NewStatsMiddleware(repo stats.Repository) *StatsMiddleware {
	return &StatsMiddleware{statsRepo: repo}
}

// WrapFunc adds asynchronous stats tracking around a tool function
func (m *StatsMiddleware) WrapFunc(name, description string, fn ToolFunc) tool.Tool {
	if m == nil || m.statsRepo == nil {
		// No stats repo configured, create tool without middleware
		t, _ := functiontool.New(
			functiontool.Config{
				Name:        name,
				Description: description,
			},
			func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
				return fn(ctx, args)
			})
		return t
	}

	t, _ := functiontool.New(
		functiontool.Config{
			Name:        name,
			Description: description,
		},
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			start := time.Now()
			result, err := fn(ctx, args)
			duration := time.Since(start)

			if meta, ok := MetadataFromContext(ctx); ok {
				usage := &stats.ToolUsageEvent{
					UserID:     meta.UserID,
					AgentID:    meta.AgentID,
					ToolName:   name,
					Timestamp:  time.Now(),
					DurationMs: int(duration.Milliseconds()),
					Success:    err == nil,
					SessionID:  meta.SessionID,
					Symbol:     meta.Symbol,
				}

				go m.statsRepo.InsertToolUsage(context.Background(), usage)
			}

			return result, err
		})
	return t
}
