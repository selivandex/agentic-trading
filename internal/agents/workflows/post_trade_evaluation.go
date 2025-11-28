package workflows

import (
	"google.golang.org/adk/agent"

	"prometheus/internal/agents"
	"prometheus/pkg/errors"
)

// CreatePostTradeEvaluationWorkflow creates workflow for analyzing closed trades.
// Triggered when a position closes (stop-loss, take-profit, or manual exit).
//
// Flow: SelfEvaluator analyzes trade outcome → extracts lessons → saves to memory
func (f *Factory) CreatePostTradeEvaluationWorkflow() (agent.Agent, error) {
	f.log.Info("Creating post-trade evaluation workflow")

	// SelfEvaluator agent
	// Input: Closed trade data (entry, exit, P&L, duration, original signal)
	// Output: Trade analysis, lessons learned, memory entries
	// Tools: get_trade_journal, get_strategy_stats, save_memory
	evaluatorAgent, err := f.createAgent(agents.AgentSelfEvaluator)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create self evaluator")
	}

	f.log.Info("Post-trade evaluation workflow created successfully")
	return evaluatorAgent, nil
}
