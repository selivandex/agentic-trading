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

// CreatePathSelector creates a PathSelector that routes between fast-path and committee
func (f *Factory) CreatePathSelector() (*PathSelector, error) {
	f.log.Info("Creating PathSelector for intelligent routing")

	// Get fast-path agent (OpportunitySynthesizer)
	fastPath, err := f.createAgent(agents.AgentOpportunitySynthesizer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fast-path agent")
	}

	// Get committee agent (ResearchCommittee workflow)
	committeeAgent, err := f.CreateResearchCommitteeWorkflow()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create committee workflow")
	}

	// Create path selector with default configuration
	selector := NewPathSelector(PathSelectorConfig{
		FastPath:              fastPath,
		CommitteePath:         committeeAgent,
		ForceCommitteeOnHigh:  true, // Always use committee for high/critical priority
		CommitteeUsagePercent: 20.0, // Target 20% of requests to committee (cost control)
	})

	f.log.Info("PathSelector created successfully")
	return selector, nil
}
