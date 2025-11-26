package workflows

import (
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/workflowagents/sequentialagent"

	"prometheus/internal/agents"
	"prometheus/pkg/errors"
)

// CreateMarketResearchWorkflow creates the global market research workflow
// This workflow runs ONCE per symbol/exchange and publishes opportunities for ALL users
//
// Flow: ParallelAnalysts (8 agents) → OpportunitySynthesizer → publish_opportunity → Kafka
func (f *Factory) CreateMarketResearchWorkflow() (agent.Agent, error) {
	f.log.Info("Creating market research workflow")

	// Step 1: Parallel Analysts (existing implementation)
	// Runs 8 specialist analysts simultaneously
	analystsAgent, err := f.CreateParallelAnalysts()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create parallel analysts")
	}

	// Step 2: Opportunity Synthesizer
	// Receives outputs from all 8 analysts and decides whether to publish
	synthesizerAgent, err := f.createAgent(agents.AgentOpportunitySynthesizer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create opportunity synthesizer")
	}

	// Create sequential workflow: Analysts → Synthesizer
	workflow, err := sequentialagent.New(sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        "MarketResearchWorkflow",
			Description: "Global market research pipeline: parallel multi-dimensional analysis → opportunity synthesis → signal publication",
			SubAgents: []agent.Agent{
				analystsAgent,    // 8 analysts in parallel
				synthesizerAgent, // 1 synthesizer (decides to publish or skip)
			},
		},
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to create market research workflow")
	}

	f.log.Info("Market research workflow created successfully")
	return workflow, nil
}
