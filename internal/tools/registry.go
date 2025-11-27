package tools

import (
	"sync"

	"google.golang.org/adk/tool"
)

// Registry stores tools by name for discovery and lookup
type Registry struct {
	tools    map[string]tool.Tool
	metadata map[string]*ToolDefinition
	mu       sync.RWMutex
}

// NewRegistry constructs an empty tool registry
func NewRegistry() *Registry {
	registry := &Registry{
		tools:    make(map[string]tool.Tool),
		metadata: make(map[string]*ToolDefinition),
	}
	// Pre-populate metadata from catalog
	for _, def := range Definitions() {
		defCopy := def // Create a copy to avoid pointer issues
		registry.metadata[def.Name] = &defCopy
	}
	return registry
}

// Register adds or replaces a tool under the provided name
func (r *Registry) Register(name string, t tool.Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[name] = t
}

// Get retrieves a tool by name if registered
func (r *Registry) Get(name string) (tool.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List returns the names of all registered tools
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// GetMetadata retrieves tool metadata by name
func (r *Registry) GetMetadata(name string) (*ToolDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	meta, ok := r.metadata[name]
	return meta, ok
}

// GetRiskLevel returns the risk level for a tool, or RiskLevelNone if not found
func (r *Registry) GetRiskLevel(name string) RiskLevel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if meta, ok := r.metadata[name]; ok {
		return meta.RiskLevel
	}
	return RiskLevelNone
}

// IsHighRisk returns true if the tool has High or Critical risk level
func (r *Registry) IsHighRisk(name string) bool {
	level := r.GetRiskLevel(name)
	return level == RiskLevelHigh || level == RiskLevelCritical
}

// ListByRiskLevel returns all tool names matching the specified risk level
func (r *Registry) ListByRiskLevel(level RiskLevel) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []string
	for name, meta := range r.metadata {
		if meta.RiskLevel == level {
			result = append(result, name)
		}
	}
	return result
}
