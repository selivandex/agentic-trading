package middleware

import (
	"context"
	"time"

	"prometheus/internal/domain/stats"
	"prometheus/internal/tools"
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
func (m *StatsMiddleware) Wrap(t tools.Tool) tools.Tool {
	if m == nil || m.statsRepo == nil {
		return t
	}

	return tools.New(t.Name(), t.Description(), func(ctx context.Context, args interface{}) (interface{}, error) {
		start := time.Now()
		result, err := t.Execute(ctx, args)
		duration := time.Since(start)

		if meta, ok := tools.MetadataFromContext(ctx); ok {
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
