package workflows

import (
	"google.golang.org/adk/agent"

	"prometheus/internal/agents"
	"prometheus/pkg/errors"
)

// CreateMarketResearchWorkflow creates the global market research workflow
// This workflow runs ONCE per symbol/exchange and publishes opportunities for ALL users
//
// Architecture:
//   - Fast Path (default): OpportunitySynthesizer (single agent, 15-30s, $0.05-0.10)
//   - Committee Path (high-priority): ResearchCommittee (4 analysts + synthesizer, 60-90s, $0.30-0.50)
//
// Flow:
//
//	OpportunitySynthesizer → (calls tools) → publish_opportunity → Kafka
//	OR
//	ResearchCommittee → (4 analysts parallel + synthesis) → publish_opportunity → Kafka
func (f *Factory) CreateMarketResearchWorkflow() (agent.Agent, error) {
	f.log.Info("Creating market research workflow (fast-path: OpportunitySynthesizer)")

	// Single agent: OpportunitySynthesizer calls comprehensive analysis tools directly
	// No intermediate analyst agents needed - tools return algorithmic analysis
	// Synthesizer does the LLM reasoning and decision-making
	synthesizerAgent, err := f.createAgent(agents.AgentOpportunitySynthesizer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create opportunity synthesizer")
	}

	f.log.Info("Market research workflow created successfully (fast-path)")
	return synthesizerAgent, nil
}
