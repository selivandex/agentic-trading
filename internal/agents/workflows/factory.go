package workflows

import (
	"google.golang.org/adk/agent"

	"prometheus/internal/agents"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Factory creates workflow agents (parallel, sequential, loop)
type Factory struct {
	baseFactory *agents.Factory
	provider    string
	model       string
	log         *logger.Logger
}

// NewFactory creates a new workflow agent factory
func NewFactory(baseFactory *agents.Factory, provider, model string) *Factory {
	return &Factory{
		baseFactory: baseFactory,
		provider:    provider,
		model:       model,
		log:         logger.Get().With("component", "workflow_factory"),
	}
}

// createAgent is a helper to create individual agents
func (f *Factory) createAgent(agentType agents.AgentType) (agent.Agent, error) {
	ag, err := f.baseFactory.CreateAgentForUser(agentType, f.provider, f.model)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create agent %s", agentType)
	}
	return ag, nil
}
