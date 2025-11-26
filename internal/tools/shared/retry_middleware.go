package shared
import (
	"time"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)
// RetryMiddleware retries tool execution on error with optional backoff
type RetryMiddleware struct {
	Attempts int
	Backoff  time.Duration
}
// WrapFunc wraps a tool function with retry logic
func (m RetryMiddleware) WrapFunc(name, description string, fn ToolFunc) tool.Tool {
	attempts := m.Attempts
	if attempts <= 0 {
		attempts = 1
	}
	backoff := m.Backoff
	t, _ := functiontool.New(
		functiontool.Config{
			Name:        name,
			Description: description,
		},
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			var result map[string]interface{}
			var err error
			for i := 0; i < attempts; i++ {
				result, err = fn(ctx, args)
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
	return t
}
