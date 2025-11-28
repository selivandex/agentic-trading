package agents

// AgentType enumerates supported agent specializations.
type AgentType string

const (
	// Personal trading workflow agents (per-user)
	AgentPortfolioManager AgentType = "portfolio_manager" // Portfolio Manager responsible for client account
	AgentPositionManager  AgentType = "position_manager"

	// Market research workflow agent (global)
	AgentOpportunitySynthesizer AgentType = "opportunity_synthesizer"

	// Research Committee agents (global, multi-agent collaboration)
	AgentTechnicalAnalyst  AgentType = "technical_analyst"  // Technical indicators specialist
	AgentStructuralAnalyst AgentType = "structural_analyst" // SMC/ICT specialist
	AgentFlowAnalyst       AgentType = "flow_analyst"       // Order flow & whale activity
	AgentMacroAnalyst      AgentType = "macro_analyst"      // Economic context & correlations
	AgentHeadOfResearch    AgentType = "head_of_research"   // Synthesizer & debate leader

	// Portfolio management agent (onboarding)
	AgentPortfolioArchitect AgentType = "portfolio_architect"

	// System agents (adaptive intelligence)
	AgentRegimeDetector      AgentType = "regime_detector"
	AgentRegimeInterpreter   AgentType = "regime_interpreter"
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
