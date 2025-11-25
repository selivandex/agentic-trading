package middleware

import (
	"context"
	"time"

	"prometheus/internal/tools"
)

// TimeoutMiddleware enforces per-call deadlines for tool execution.
type TimeoutMiddleware struct {
	Timeout time.Duration
}

// Wrap sets a timeout on tool execution if configured.
func (m TimeoutMiddleware) Wrap(t tools.Tool) tools.Tool {
	if m.Timeout <= 0 {
		return t
	}

	return tools.New(t.Name(), t.Description(), func(ctx context.Context, args interface{}) (interface{}, error) {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, m.Timeout)
		defer cancel()
		return t.Execute(ctxWithTimeout, args)
	})
}
