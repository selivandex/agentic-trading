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

// CreateMarketResearchWorkflowWithPathSelector creates a market research workflow that intelligently
// routes between fast-path (OpportunitySynthesizer) and committee-path (ResearchCommittee)
// based on request priority, position size, volatility, and cost control limits.
//
// This is the Phase 3 enhancement: multi-agent collaboration for high-stakes decisions.
//
// Usage:
//   - Fast-path (80% of requests): Routine scanning, low-medium priority, small positions
//   - Committee-path (20% of requests): High-priority, large positions, high volatility, conflicting signals
//
// Returns: PathSelector that dynamically routes to the appropriate workflow
func (f *Factory) CreateMarketResearchWorkflowWithPathSelector(config PathSelectorConfig) (*PathSelector, error) {
	f.log.Info("Creating market research workflow with intelligent path selection (Phase 3)")

	// Create fast-path workflow (OpportunitySynthesizer)
	fastPath, err := f.CreateMarketResearchWorkflow()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fast-path workflow")
	}

	// Create committee-path workflow (ResearchCommittee)
	committeePath, err := f.CreateResearchCommitteeWorkflow()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create research committee workflow")
	}

	// Configure and create PathSelector
	config.FastPath = fastPath
	config.CommitteePath = committeePath

	pathSelector := NewPathSelector(config)

	f.log.Infow("Market research workflow with path selector created successfully",
		"high_stakes_threshold", config.HighStakesThreshold,
		"volatility_threshold", config.VolatilityThreshold,
		"target_committee_usage_pct", config.CommitteeUsagePercent,
	)

	return pathSelector, nil
}
