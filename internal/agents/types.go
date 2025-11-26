package agents

// AgentType enumerates supported agent specializations.
type AgentType string

const (
	AgentMarketAnalyst      AgentType = "market_analyst"
	AgentSMCAnalyst         AgentType = "smc_analyst"
	AgentSentimentAnalyst   AgentType = "sentiment_analyst"
	AgentOnChainAnalyst     AgentType = "onchain_analyst"
	AgentCorrelationAnalyst AgentType = "correlation_analyst"
	AgentMacroAnalyst       AgentType = "macro_analyst"
	AgentOrderFlowAnalyst   AgentType = "order_flow_analyst"
	AgentDerivativesAnalyst AgentType = "derivatives_analyst"

	AgentStrategyPlanner        AgentType = "strategy_planner"
	AgentRiskManager            AgentType = "risk_manager"
	AgentExecutor               AgentType = "executor"
	AgentPositionManager        AgentType = "position_manager"
	AgentSelfEvaluator          AgentType = "self_evaluator"
	AgentOpportunitySynthesizer AgentType = "opportunity_synthesizer"
	AgentPortfolioArchitect     AgentType = "portfolio_architect"
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
