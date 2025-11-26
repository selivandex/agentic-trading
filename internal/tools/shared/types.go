package shared
import "google.golang.org/adk/tool"
// ToolFunc is the function signature for tool execution
// Used by middleware to wrap tool functions
type ToolFunc func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error)
