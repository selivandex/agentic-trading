package agents

// AgentType enumerates supported agent specializations.
type AgentType string

const (
	// Personal trading workflow agents (per-user)
	AgentPortfolioManager AgentType = "portfolio_manager" // Portfolio Manager responsible for client account
	AgentPositionManager  AgentType = "position_manager"

	// Market research workflow agent (global)
	AgentOpportunitySynthesizer AgentType = "opportunity_synthesizer"

	// Portfolio management agent (onboarding)
	AgentPortfolioArchitect AgentType = "portfolio_architect"

	// System agents (adaptive intelligence)
	AgentRegimeDetector      AgentType = "regime_detector"
	AgentPerformanceAnalyzer AgentType = "performance_analyzer"

	// Quality gates and review agents
	AgentPreTradeReviewer     AgentType = "pre_trade_reviewer"    // Pre-execution quality gate
	AgentPerformanceCommittee AgentType = "performance_committee" // Weekly performance review
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
