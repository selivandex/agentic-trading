package workflows

import (
	"google.golang.org/adk/agent"

	"prometheus/internal/agents"
	"prometheus/pkg/errors"
)

// CreateMarketResearchWorkflow creates the global market research workflow
// This workflow runs ONCE per symbol/exchange and publishes opportunities for ALL users
// Flow: OpportunitySynthesizer → (calls get_technical_analysis, get_smc_analysis, get_market_analysis) → publish_opportunity → Kafka
func (f *Factory) CreateMarketResearchWorkflow() (agent.Agent, error) {
	f.log.Info("Creating market research workflow (simplified architecture)")

	// Single agent: OpportunitySynthesizer calls comprehensive analysis tools directly
	// No intermediate analyst agents needed - tools return algorithmic analysis
	// Synthesizer does the LLM reasoning and decision-making
	synthesizerAgent, err := f.createAgent(agents.AgentOpportunitySynthesizer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create opportunity synthesizer")
	}

	f.log.Info("Market research workflow created successfully")
	return synthesizerAgent, nil
}
