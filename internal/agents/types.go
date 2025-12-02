package agents

// AgentType enumerates supported agent specializations.
type AgentType string

const (
	// MVP agents (only 3 for simple operation)
	AgentPortfolioArchitect     AgentType = "portfolio_architect"     // Designs initial portfolio during onboarding
	AgentOpportunitySynthesizer AgentType = "opportunity_synthesizer" // Global market research
	AgentPortfolioManager       AgentType = "portfolio_manager"       // Personal trading decisions
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
