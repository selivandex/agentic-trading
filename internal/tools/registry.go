package tools

import "sync"

// Registry stores tools by name for discovery and lookup.
type Registry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewRegistry constructs an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds or replaces a tool under the provided name.
func (r *Registry) Register(name string, t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[name] = t
}

// Get retrieves a tool by name if registered.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List returns the names of all registered tools.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}

	return names
}
