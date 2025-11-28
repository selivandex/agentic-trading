package agents

// AgentType enumerates supported agent specializations.
type AgentType string

const (
	// Personal trading workflow agents (per-user)
	AgentStrategyPlanner AgentType = "strategy_planner"
	AgentPositionManager AgentType = "position_manager"
	AgentSelfEvaluator   AgentType = "self_evaluator"

	// Market research workflow agent (global)
	AgentOpportunitySynthesizer AgentType = "opportunity_synthesizer"

	// Portfolio management agent (onboarding)
	AgentPortfolioArchitect AgentType = "portfolio_architect"

	// System agents (adaptive intelligence)
	AgentRegimeDetector      AgentType = "regime_detector"
	AgentPerformanceAnalyzer AgentType = "performance_analyzer"
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
