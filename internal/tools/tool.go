package tools

import (
	"google.golang.org/adk/tool"
	functiontool "google.golang.org/adk/tool/functiontool"
)

// Tool is an alias to the ADK tool interface.
type Tool = tool.Tool

// HandlerFunc is the function signature for tool handlers.
type HandlerFunc = functiontool.Handler

// New creates a new function-backed Tool compatible with ADK.
func New(name, description string, handler HandlerFunc) Tool {
	return functiontool.New(name, description, handler)
}
