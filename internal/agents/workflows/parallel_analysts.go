package workflows

import (
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/workflowagents/parallelagent"

	"prometheus/internal/agents"
	"prometheus/pkg/errors"
)

// CreateParallelAnalysts creates a parallel workflow that runs all analysts simultaneously
func (f *Factory) CreateParallelAnalysts() (agent.Agent, error) {
	f.log.Info("Creating parallel analysts workflow")

	// Create all analyst agents
	analystTypes := []agents.AgentType{
		agents.AgentMarketAnalyst,
		agents.AgentSMCAnalyst,
		agents.AgentSentimentAnalyst,
		agents.AgentOrderFlowAnalyst,
		agents.AgentDerivativesAnalyst,
		agents.AgentMacroAnalyst,
		agents.AgentOnChainAnalyst,
		agents.AgentCorrelationAnalyst,
	}

	analystAgents := make([]agent.Agent, 0, len(analystTypes))
	for _, agentType := range analystTypes {
		ag, err := f.createAgent(agentType)
		if err != nil {
			f.log.Warnf("Failed to create %s: %v", agentType, err)
			continue // Skip failed agents
		}
		analystAgents = append(analystAgents, ag)
	}

	if len(analystAgents) == 0 {
		return nil, errors.New("no analysts could be created")
	}

	f.log.Infof("Created %d analyst agents for parallel execution", len(analystAgents))

	// Create parallel workflow
	parallelWorkflow, err := parallelagent.New(parallelagent.Config{
		AgentConfig: agent.Config{
			Name:        "ParallelAnalysts",
			Description: "Runs all market analysts in parallel for comprehensive multi-dimensional analysis",
			SubAgents:   analystAgents,
		},
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to create parallel agent")
	}

	return parallelWorkflow, nil
}
