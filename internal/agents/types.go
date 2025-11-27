package agents

// AgentType enumerates supported agent specializations.
type AgentType string

const (
	// Personal trading workflow agents (per-user)
	AgentStrategyPlanner AgentType = "strategy_planner"
	AgentRiskManager     AgentType = "risk_manager"
	AgentExecutor        AgentType = "executor"
	AgentPositionManager AgentType = "position_manager"
	AgentSelfEvaluator   AgentType = "self_evaluator"

	// Market research workflow agent (global)
	AgentOpportunitySynthesizer AgentType = "opportunity_synthesizer"

	// Portfolio management agent (onboarding)
	AgentPortfolioArchitect AgentType = "portfolio_architect"
)

// UserAgentConfig carries runtime user-specific context for building pipelines.
type UserAgentConfig struct {
	UserID     string
	AIProvider string
	Model      string
	Symbol     string
	MarketType string
	RiskLevel  string
}
