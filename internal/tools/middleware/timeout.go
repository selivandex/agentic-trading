package middleware

import (
	"context"
	"time"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// TimeoutMiddleware enforces per-call deadlines for tool execution.
type TimeoutMiddleware struct {
	Timeout time.Duration
}

// Wrap sets a timeout on tool execution if configured.
func (m TimeoutMiddleware) Wrap(t tool.Tool) tool.Tool {
	if m.Timeout <= 0 {
		return t
	}

	return functiontool.New(t.Name(), t.Description(), func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, m.Timeout)
		defer cancel()
		return t.Execute(ctxWithTimeout, args)
	})
}
