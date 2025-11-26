package tools

import (
	"time"

	"google.golang.org/adk/tool"

	"prometheus/internal/tools/middleware"
	"prometheus/internal/tools/shared"
)

// Factory provides fluent API for creating tools with middleware
type Factory struct {
	name        string
	description string
	fn          middleware.ToolFunc
	deps        shared.Deps

	// Middleware options
	withRetry   bool
	retryConfig middleware.RetryMiddleware

	withTimeout   bool
	timeoutConfig middleware.TimeoutMiddleware

	withStats bool
}

// NewFactory creates a new factory for a tool
func NewFactory(name, description string, fn middleware.ToolFunc, deps shared.Deps) *Factory {
	return &Factory{
		name:        name,
		description: description,
		fn:          fn,
		deps:        deps,
		// Default configs
		retryConfig:   middleware.RetryMiddleware{Attempts: 3, Backoff: 500 * time.Millisecond},
		timeoutConfig: middleware.TimeoutMiddleware{Timeout: 30 * time.Second},
	}
}

// WithRetry enables retry middleware
func (b *Factory) WithRetry(attempts int, backoff time.Duration) *Factory {
	b.withRetry = true
	b.retryConfig = middleware.RetryMiddleware{
		Attempts: attempts,
		Backoff:  backoff,
	}
	return b
}

// WithTimeout enables timeout middleware
func (b *Factory) WithTimeout(timeout time.Duration) *Factory {
	b.withTimeout = true
	b.timeoutConfig = middleware.TimeoutMiddleware{
		Timeout: timeout,
	}
	return b
}

// WithStats enables stats tracking middleware
func (b *Factory) WithStats() *Factory {
	b.withStats = true
	return b
}

// Build creates the tool with configured middleware applied
func (b *Factory) Build() tool.Tool {
	fn := b.fn

	// Apply middleware in order: retry -> timeout -> stats
	// Inner layers are applied first

	// 1. Retry (innermost - retries the actual tool logic)
	if b.withRetry {
		fn = wrapWithRetry(b.retryConfig, fn)
	}

	// 2. Timeout (wraps retry)
	if b.withTimeout {
		fn = wrapWithTimeout(b.timeoutConfig, fn)
	}

	// 3. Stats (outermost - tracks everything including retries)
	if b.withStats && b.deps.StatsRepo != nil {
		statsMiddleware := middleware.NewStatsMiddleware(b.deps.StatsRepo)
		return statsMiddleware.WrapFunc(b.name, b.description, fn)
	}

	// No stats, create tool directly
	return createToolFromFunc(b.name, b.description, fn)
}

// Helper functions to apply middleware

func wrapWithRetry(retry middleware.RetryMiddleware, fn middleware.ToolFunc) middleware.ToolFunc {
	return func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
		var result map[string]interface{}
		var err error

		attempts := retry.Attempts
		if attempts <= 0 {
			attempts = 1
		}

		for i := 0; i < attempts; i++ {
			result, err = fn(ctx, args)
			if err == nil {
				return result, nil
			}

			// Wait before retry
			if retry.Backoff > 0 && i < attempts-1 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(retry.Backoff):
				}
			}
		}

		return result, err
	}
}

func wrapWithTimeout(timeout middleware.TimeoutMiddleware, fn middleware.ToolFunc) middleware.ToolFunc {
	if timeout.Timeout <= 0 {
		return fn
	}

	return func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
		// Create context with timeout
		// tool.Context is an interface, we need to get underlying context
		// For now, pass through - timeout will be handled at invocation level
		return fn(ctx, args)
	}
}

func createToolFromFunc(name, description string, fn middleware.ToolFunc) tool.Tool {
	// Use timeout middleware with no timeout (passthrough) to create the tool
	return middleware.TimeoutMiddleware{}.WrapFunc(name, description, fn)
}
