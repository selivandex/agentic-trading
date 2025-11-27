package workflows

import (
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/workflowagents/sequentialagent"

	"prometheus/internal/agents"
	"prometheus/pkg/errors"
)

// CreatePortfolioInitializationWorkflow creates the onboarding workflow for new users
// This workflow is triggered when a user allocates capital and needs an initial portfolio strategy
//
// Flow: OpportunitySynthesizer (market snapshot via analysis tools) → PortfolioArchitect → RiskManager → Executor
func (f *Factory) CreatePortfolioInitializationWorkflow() (agent.Agent, error) {
	f.log.Info("Creating portfolio initialization workflow")

	// Step 1: Market Snapshot (using OpportunitySynthesizer tools)
	// OpportunitySynthesizer calls 3 analysis tools to get comprehensive market state
	// Portfolio Architect will use this data to design initial strategy
	// Note: We use OpportunitySynthesizer because it already has access to all analysis tools
	// For onboarding, we just need the market analysis, not the publish decision
	marketSnapshotAgent, err := f.createAgent(agents.AgentOpportunitySynthesizer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create market snapshot agent")
	}

	// Step 2: Portfolio Architect (strategy design)
	// Design diversified portfolio based on:
	// - User's capital and risk tolerance
	// - Current market conditions (from Step 1)
	// - Asset correlations and diversification principles
	architectAgent, err := f.createAgent(agents.AgentPortfolioArchitect)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create portfolio architect")
	}

	// Step 3: Risk Manager (validate strategy)
	// Ensure portfolio meets risk requirements:
	// - Position sizes within limits
	// - Diversification adequate
	// - No over-concentration
	riskAgent, err := f.createAgent(agents.AgentRiskManager)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create risk manager")
	}

	// Step 4: Executor (place initial orders)
	// Execute the portfolio strategy:
	// - Place orders for each allocation
	// - Save strategy to database
	// - Link positions to strategy
	executorAgent, err := f.createAgent(agents.AgentExecutor)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create executor")
	}

	// Create sequential workflow
	workflow, err := sequentialagent.New(sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        "PortfolioInitializationWorkflow",
			Description: "Complete portfolio initialization for user onboarding: market snapshot (via opportunity analyzer tools) → strategy design → risk validation → initial execution",
			SubAgents: []agent.Agent{
				marketSnapshotAgent,
				architectAgent,
				riskAgent,
				executorAgent,
			},
		},
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to create portfolio initialization workflow")
	}

	f.log.Info("Portfolio initialization workflow created successfully")
	return workflow, nil
}
