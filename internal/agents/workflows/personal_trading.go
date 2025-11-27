package workflows

import (
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/workflowagents/sequentialagent"

	"prometheus/internal/agents"
	"prometheus/pkg/errors"
)

// CreatePersonalTradingWorkflow creates a personal trading workflow for processing PRE-ANALYZED opportunity signals
// This workflow does NOT run market analysts - it assumes market research has already been completed
//
// Key difference from CreateTradingPipeline:
// - NO ParallelAnalysts step (analysts already ran in MarketResearchWorkflow)
// - Receives PRE-ANALYZED market signals as input
// - Focuses on personal decision making: strategy synthesis → risk validation → execution
//
// Flow: StrategyPlanner → RiskManager → Executor
func (f *Factory) CreatePersonalTradingWorkflow() (agent.Agent, error) {
	f.log.Info("Creating personal trading workflow (analyst-free)")

	// Step 1: Strategy Planner
	// Input: global market signal + user context (portfolio, risk profile)
	// Output: personal trading plan (entry, SL, TP, size)
	// Tools: get_portfolio_summary, get_positions, get_balance, get_user_risk_profile
	strategyAgent, err := f.createAgent(agents.AgentStrategyPlanner)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create strategy planner")
	}

	// Step 2: Risk Manager
	// Input: trading plan from strategy planner
	// Output: validated/adjusted plan or rejection
	// Tools: validate_position, check_risk_limits, check_circuit_breaker
	riskAgent, err := f.createAgent(agents.AgentRiskManager)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create risk manager")
	}

	// Step 3: Executor
	// Input: approved trading plan from risk manager
	// Output: order placed (or skip if rejected)
	// Tools: place_order, cancel_order
	executorAgent, err := f.createAgent(agents.AgentExecutor)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create executor")
	}

	// Create sequential workflow: Strategy → Risk → Executor
	workflow, err := sequentialagent.New(sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        "PersonalTradingWorkflow",
			Description: "Personal trading workflow for processing opportunity signals: strategy synthesis (with user context) → risk validation → order execution",
			SubAgents: []agent.Agent{
				strategyAgent,
				riskAgent,
				executorAgent,
			},
		},
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to create personal trading workflow")
	}

	f.log.Info("Personal trading workflow created successfully")
	return workflow, nil
}
