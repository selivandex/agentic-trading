package middleware

import (
	"context"
	"time"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"prometheus/internal/domain/stats"
	toolctx "prometheus/internal/tools"
)

// StatsMiddleware records tool usage metrics into the stats repository.
type StatsMiddleware struct {
	statsRepo stats.Repository
}

// NewStatsMiddleware constructs a middleware with the provided repository.
func NewStatsMiddleware(repo stats.Repository) *StatsMiddleware {
	return &StatsMiddleware{statsRepo: repo}
}

// Wrap adds asynchronous stats tracking around a tool.
func (m *StatsMiddleware) Wrap(t tool.Tool) tool.Tool {
	if m == nil || m.statsRepo == nil {
		return t
	}

	return functiontool.New(t.Name(), t.Description(), func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		start := time.Now()
		result, err := t.Execute(ctx, args)
		duration := time.Since(start)

		if meta, ok := toolctx.MetadataFromContext(ctx); ok {
			usage := &stats.ToolUsageEvent{
				UserID:     meta.UserID,
				AgentID:    meta.AgentID,
				ToolName:   t.Name(),
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
}
