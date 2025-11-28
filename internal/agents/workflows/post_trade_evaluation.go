package workflows

import (
	"google.golang.org/adk/agent"

	"prometheus/internal/agents"
	"prometheus/pkg/errors"
)

// CreatePostTradeEvaluationWorkflow creates workflow for analyzing closed trades.
// Triggered when a position closes (stop-loss, take-profit, or manual exit).
//
// Flow: PerformanceCommittee analyzes trade outcome → extracts lessons → saves to memory
func (f *Factory) CreatePostTradeEvaluationWorkflow() (agent.Agent, error) {
	f.log.Info("Creating post-trade evaluation workflow")

	// PerformanceCommittee agent
	// Input: Closed trade data (entry, exit, P&L, duration, original signal)
	// Output: Trade analysis, validated patterns, lessons learned, memory entries
	// Tools: get_trade_journal, get_strategy_stats, save_memory
	evaluatorAgent, err := f.createAgent(agents.AgentPerformanceCommittee)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create performance committee")
	}

	f.log.Info("Post-trade evaluation workflow created successfully")
	return evaluatorAgent, nil
}
