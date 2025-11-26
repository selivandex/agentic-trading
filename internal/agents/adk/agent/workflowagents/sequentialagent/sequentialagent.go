package sequentialagent

import (
	"context"

	"google.golang.org/adk/agent"
)

// Config holds the metadata for the sequential workflow agent.
type Config struct {
	agent.Config
}

// Agent runs subagents in order, threading outputs forward.
type Agent struct {
	cfg agent.Config
}

// New constructs a new sequential workflow agent.
func New(cfg Config) (*Agent, error) {
	return &Agent{cfg: cfg.Config}, nil
}

// Name returns the agent name.
func (a *Agent) Name() string { return a.cfg.Name }

// Description returns the agent description.
func (a *Agent) Description() string { return a.cfg.Description }

// Run executes subagents one after another, merging their outputs.
func (a *Agent) Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	current := input
	for _, sub := range a.cfg.SubAgents {
		if sub == nil {
			continue
		}
		out, err := sub.Run(ctx, current)
		if err != nil {
			return nil, err
		}
		for k, v := range out {
			current[k] = v
		}
	}
	return current, nil
}

var _ agent.Agent = (*Agent)(nil)
