package tools

import (
	"context"
	"errors"
)

// Tool represents a callable capability exposed to agents.
type Tool interface {
	// Name returns the unique tool identifier.
	Name() string
	// Description returns a short human-readable summary.
	Description() string
	// Execute performs the tool's action using the provided arguments.
	Execute(ctx context.Context, args interface{}) (interface{}, error)
}

// HandlerFunc is the function signature for tool handlers.
type HandlerFunc func(ctx context.Context, args interface{}) (interface{}, error)

// FunctionTool is a simple Tool implementation backed by a handler function.
type FunctionTool struct {
	name        string
	description string
	handler     HandlerFunc
}

// New creates a new function-backed Tool.
func New(name, description string, handler HandlerFunc) Tool {
	return &FunctionTool{
		name:        name,
		description: description,
		handler:     handler,
	}
}

// Name returns the tool identifier.
func (t *FunctionTool) Name() string { return t.name }

// Description returns a human description of the tool.
func (t *FunctionTool) Description() string { return t.description }

// Execute runs the underlying handler.
func (t *FunctionTool) Execute(ctx context.Context, args interface{}) (interface{}, error) {
	if t.handler == nil {
		return nil, errors.New("tool handler is not defined")
	}

	return t.handler(ctx, args)
}
