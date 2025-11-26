package tool

import "context"

// Tool represents a callable capability available to an agent.
type Tool interface {
	Name() string
	Description() string
	// Execute performs the tool action with the provided arguments.
	Execute(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error)
}
