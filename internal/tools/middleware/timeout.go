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

// WrapFunc sets a timeout on tool execution if configured
func (m TimeoutMiddleware) WrapFunc(name, description string, fn ToolFunc) tool.Tool {
	if m.Timeout <= 0 {
		// No timeout configured, create tool without middleware
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
			ctxWithTimeout, cancel := context.WithTimeout(ctx, m.Timeout)
			defer cancel()

			// Create a new tool.Context with timeout
			// Note: tool.Context wraps context.Context, so we need to pass the timeout context
			return fn(ctxWithTimeout.(tool.Context), args)
		})
	return t
}
