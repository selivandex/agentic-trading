package workflows

import (
	"google.golang.org/adk/agent"

	"prometheus/internal/agents"
	"prometheus/pkg/errors"
)

// CreatePortfolioInitializationWorkflow creates the onboarding workflow for new users
// This workflow is triggered when a user allocates capital and needs an initial portfolio strategy
//
// Flow: PortfolioArchitect only (fetches market data via tools, designs strategy, executes via execute_trade)
//
// Note: OpportunitySynthesizer removed - overkill for simple data fetching ($0.10 + 2min wasted on decision-making)
// Note: Executor agent removed - execution via execute_trade tool (algorithmic)
func (f *Factory) CreatePortfolioInitializationWorkflow() (agent.Agent, error) {
	f.log.Info("Creating portfolio initialization workflow")

	// Portfolio Architect agent
	// Responsibilities:
	// 1. Fetch market snapshot via analysis tools (technical, SMC, market)
	// 2. Design diversified portfolio based on user's capital and risk tolerance
	// 3. Calculate optimal allocations and position sizes
	// 4. Validate each position via pre_trade_check tool
	// 5. Execute portfolio via execute_trade tool for each allocation
	//
	// Tools: get_technical_analysis, get_smc_analysis, get_market_analysis (for snapshot)
	//        calc_asset_correlation (for diversification)
	//        pre_trade_check (for validation)
	//        execute_trade (for execution)
	architectAgent, err := f.createAgent(agents.AgentPortfolioArchitect)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create portfolio architect")
	}

	f.log.Info("Portfolio initialization workflow created successfully (single-agent)")
	return architectAgent, nil
}
