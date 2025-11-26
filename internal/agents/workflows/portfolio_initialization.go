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
// Flow: MarketAnalyst (quick snapshot) → PortfolioArchitect → RiskManager → Executor
func (f *Factory) CreatePortfolioInitializationWorkflow() (agent.Agent, error) {
	f.log.Info("Creating portfolio initialization workflow")

	// Step 1: Market Analyst (quick market assessment)
	// Get current market snapshot for portfolio construction
	// Shorter timeout, fewer tools - just need basic market state
	marketAgent, err := f.createAgent(agents.AgentMarketAnalyst)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create market analyst")
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
			Description: "Complete portfolio initialization for user onboarding: market snapshot → strategy design → risk validation → initial execution",
			SubAgents: []agent.Agent{
				marketAgent,
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

