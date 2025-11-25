package functiontool

import (
	"context"

	"google.golang.org/adk/tool"
)

// Handler defines a function signature for function-backed tools.
type Handler func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error)

// FunctionTool is a simple tool backed by a handler function.
type FunctionTool struct {
	name        string
	description string
	handler     Handler
}

// New constructs a new function-backed tool.
func New(name, description string, handler Handler) tool.Tool {
	return &FunctionTool{name: name, description: description, handler: handler}
}

// Name returns the tool identifier.
func (t *FunctionTool) Name() string { return t.name }

// Description returns a human-readable description.
func (t *FunctionTool) Description() string { return t.description }

// Execute calls the wrapped handler.
func (t *FunctionTool) Execute(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	if t.handler == nil {
		return nil, nil
	}
	return t.handler(ctx, args)
}
