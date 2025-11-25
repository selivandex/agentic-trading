package agents

import "sync"

// Registry stores agents by their type for quick lookup.
type Registry struct {
	agents map[AgentType]Agent
	mu     sync.RWMutex
}

// NewRegistry constructs an empty agent registry.
func NewRegistry() *Registry {
	return &Registry{agents: make(map[AgentType]Agent)}
}

// Register adds or replaces an agent entry.
func (r *Registry) Register(agentType AgentType, ag Agent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[agentType] = ag
}

// Get retrieves an agent by type.
func (r *Registry) Get(agentType AgentType) (Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ag, ok := r.agents[agentType]
	return ag, ok
}

// List returns registered agent types.
func (r *Registry) List() []AgentType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]AgentType, 0, len(r.agents))
	for t := range r.agents {
		res = append(res, t)
	}

	return res
}
