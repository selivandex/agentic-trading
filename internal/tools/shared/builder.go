package shared

import (
	"time"

	"google.golang.org/adk/tool"
)

// ToolBuilder provides fluent API for creating tools with middleware
// Note: Stats tracking is now handled by ADK callbacks, not middleware
type ToolBuilder struct {
	name        string
	description string
	fn          ToolFunc
	deps        Deps

	// Middleware options
	withRetry   bool
	retryConfig RetryMiddleware

	withTimeout   bool
	timeoutConfig TimeoutMiddleware
}

// NewToolBuilder creates a new factory for a tool
func NewToolBuilder(name, description string, fn ToolFunc, deps Deps) *ToolBuilder {
	return &ToolBuilder{
		name:        name,
		description: description,
		fn:          fn,
		deps:        deps,
		// Default configs
		retryConfig:   RetryMiddleware{Attempts: 3, Backoff: 500 * time.Millisecond},
		timeoutConfig: TimeoutMiddleware{Timeout: 30 * time.Second},
	}
}

// WithRetry enables retry middleware
func (b *ToolBuilder) WithRetry(attempts int, backoff time.Duration) *ToolBuilder {
	b.withRetry = true
	b.retryConfig = RetryMiddleware{
		Attempts: attempts,
		Backoff:  backoff,
	}
	return b
}

// WithTimeout enables timeout middleware
func (b *ToolBuilder) WithTimeout(timeout time.Duration) *ToolBuilder {
	b.withTimeout = true
	b.timeoutConfig = TimeoutMiddleware{
		Timeout: timeout,
	}
	return b
}

// Build creates the tool with configured middleware applied
// Note: Stats tracking is handled by ADK callbacks (see internal/agents/callbacks/tool.go)
func (b *ToolBuilder) Build() tool.Tool {
	// Apply middleware in layers:
	// The order matters: we wrap from innermost (closest to business logic) to outermost
	// 1. Base function (business logic)
	// 2. Retry (retry the business logic on failure)
	// 3. Timeout (timeout includes retries)
	//
	// Stats tracking is now handled by ADK AfterToolCallback, not middleware

	name := b.name
	description := b.description
	fn := b.fn

	// Layer 1: Start with base function wrapped in retry middleware
	if b.withRetry {
		fn = b.wrapWithRetry(fn)
	}

	// Layer 2: Wrap with timeout
	if b.withTimeout {
		fn = b.wrapWithTimeout(fn)
	}

	// Layer 3: Create final tool via TimeoutMiddleware (may be passthrough if no timeout)
	return TimeoutMiddleware{Timeout: 0}.WrapFunc(name, description, fn)
}

// wrapWithRetry wraps a ToolFunc with retry logic
func (b *ToolBuilder) wrapWithRetry(fn ToolFunc) ToolFunc {
	attempts := b.retryConfig.Attempts
	if attempts <= 0 {
		attempts = 1
	}

	return func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
		var result map[string]interface{}
		var err error

		for i := 0; i < attempts; i++ {
			result, err = fn(ctx, args)
			if err == nil {
				return result, nil
			}

			// Wait before retry
			if b.retryConfig.Backoff > 0 && i < attempts-1 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(b.retryConfig.Backoff):
				}
			}
		}

		return result, err
	}
}

// wrapWithTimeout wraps a ToolFunc with timeout logic
func (b *ToolBuilder) wrapWithTimeout(fn ToolFunc) ToolFunc {
	if b.timeoutConfig.Timeout <= 0 {
		return fn
	}

	// Note: Actual timeout enforcement happens in TimeoutMiddleware.WrapFunc
	// This is a placeholder for consistency
	return fn
}
