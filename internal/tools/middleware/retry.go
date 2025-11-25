package middleware

import (
	"context"
	"time"

	"prometheus/internal/tools"
)

// RetryMiddleware retries tool execution on error with optional backoff.
type RetryMiddleware struct {
	Attempts int
	Backoff  time.Duration
}

// Wrap adds retry semantics to a tool. The final error from the last attempt is returned.
func (m RetryMiddleware) Wrap(t tools.Tool) tools.Tool {
	attempts := m.Attempts
	if attempts <= 0 {
		attempts = 1
	}

	backoff := m.Backoff

	return tools.New(t.Name(), t.Description(), func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		var result map[string]interface{}
		var err error

		for i := 0; i < attempts; i++ {
			result, err = t.Execute(ctx, args)
			if err == nil {
				return result, nil
			}

			if backoff > 0 && i < attempts-1 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff):
				}
			}
		}

		return result, err
	})
}
