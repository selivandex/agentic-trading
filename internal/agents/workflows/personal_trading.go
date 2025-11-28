package workflows

import (
	"google.golang.org/adk/agent"

	"prometheus/internal/agents"
	"prometheus/pkg/errors"
)

// CreatePersonalTradingWorkflow creates a personal trading workflow for processing PRE-ANALYZED opportunity signals
// This workflow does NOT run market analysts - it assumes market research has already been completed
//
// Flow: PortfolioManager only (validates via pre_trade_check tool, executes via execute_trade tool)
//
// Note: RiskManager and Executor agents removed - validation and execution now done via tools:
// - pre_trade_check: Algorithmic risk validation via RiskEngine
// - execute_trade: Algorithmic execution via ExecutionService
func (f *Factory) CreatePersonalTradingWorkflow() (agent.Agent, error) {
	f.log.Info("Creating personal trading workflow (single-agent with validation/execution tools)")

	// PortfolioManager agent
	// Input: global market signal + user context (portfolio, risk profile)
	// Output: executes personal trading plan via tools
	// Tools: get_portfolio, get_balance, pre_trade_check, execute_trade, save_memory
	// Flow:
	//   1. Check user portfolio (tools)
	//   2. Calculate position size
	//   3. Validate via pre_trade_check tool (RiskEngine)
	//   4. Execute via execute_trade tool (ExecutionService)
	portfolioMgrAgent, err := f.createAgent(agents.AgentPortfolioManager)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create portfolio manager")
	}

	f.log.Info("Personal trading workflow created successfully (single-agent)")
	return portfolioMgrAgent, nil
}
