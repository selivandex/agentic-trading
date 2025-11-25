package agents

import "time"

// AgentConfig captures runtime settings for an agent instance.
type AgentConfig struct {
	Type                 AgentType
	Name                 string
	Tools                []string
	SystemPromptTemplate string

	MaxToolCalls      int
	MaxThinkingTokens int
	TimeoutPerTool    time.Duration
	TotalTimeout      time.Duration
	MaxCostPerRun     float64
}

// DefaultAgentConfigs aligns with the phase 7 plan for agent limits and prompts.
var DefaultAgentConfigs = map[AgentType]AgentConfig{
	AgentMarketAnalyst: {
		Type:                 AgentMarketAnalyst,
		Name:                 "MarketAnalyst",
		Tools:                AgentToolMap[AgentMarketAnalyst],
		SystemPromptTemplate: "agents/market_analyst",
		MaxToolCalls:         25,
		MaxThinkingTokens:    4000,
		TimeoutPerTool:       10 * time.Second,
		TotalTimeout:         2 * time.Minute,
		MaxCostPerRun:        0.10,
	},
	AgentSMCAnalyst: {
		Type:                 AgentSMCAnalyst,
		Name:                 "SMCAnalyst",
		Tools:                AgentToolMap[AgentSMCAnalyst],
		SystemPromptTemplate: "agents/smc_analyst",
		MaxToolCalls:         15,
		MaxThinkingTokens:    4000,
		TimeoutPerTool:       10 * time.Second,
		TotalTimeout:         time.Minute,
		MaxCostPerRun:        0.10,
	},
	AgentSentimentAnalyst: {
		Type:                 AgentSentimentAnalyst,
		Name:                 "SentimentAnalyst",
		Tools:                AgentToolMap[AgentSentimentAnalyst],
		SystemPromptTemplate: "agents/sentiment_analyst",
		MaxToolCalls:         10,
		MaxThinkingTokens:    2000,
		TimeoutPerTool:       15 * time.Second,
		TotalTimeout:         time.Minute,
		MaxCostPerRun:        0.05,
	},
	AgentOnChainAnalyst: {
		Type:                 AgentOnChainAnalyst,
		Name:                 "OnChainAnalyst",
		Tools:                AgentToolMap[AgentOnChainAnalyst],
		SystemPromptTemplate: "agents/onchain_analyst",
		MaxToolCalls:         10,
		MaxThinkingTokens:    2000,
		TimeoutPerTool:       15 * time.Second,
		TotalTimeout:         time.Minute,
		MaxCostPerRun:        0.05,
	},
	AgentCorrelationAnalyst: {
		Type:                 AgentCorrelationAnalyst,
		Name:                 "CorrelationAnalyst",
		Tools:                AgentToolMap[AgentCorrelationAnalyst],
		SystemPromptTemplate: "agents/correlation_analyst",
		MaxToolCalls:         8,
		MaxThinkingTokens:    3000,
		TimeoutPerTool:       10 * time.Second,
		TotalTimeout:         time.Minute,
		MaxCostPerRun:        0.05,
	},
	AgentMacroAnalyst: {
		Type:                 AgentMacroAnalyst,
		Name:                 "MacroAnalyst",
		Tools:                AgentToolMap[AgentMacroAnalyst],
		SystemPromptTemplate: "agents/macro_analyst",
		MaxToolCalls:         8,
		MaxThinkingTokens:    3000,
		TimeoutPerTool:       10 * time.Second,
		TotalTimeout:         time.Minute,
		MaxCostPerRun:        0.05,
	},
	AgentOrderFlowAnalyst: {
		Type:                 AgentOrderFlowAnalyst,
		Name:                 "OrderFlowAnalyst",
		Tools:                AgentToolMap[AgentOrderFlowAnalyst],
		SystemPromptTemplate: "agents/order_flow_analyst",
		MaxToolCalls:         12,
		MaxThinkingTokens:    3000,
		TimeoutPerTool:       5 * time.Second,
		TotalTimeout:         time.Minute,
		MaxCostPerRun:        0.05,
	},
	AgentDerivativesAnalyst: {
		Type:                 AgentDerivativesAnalyst,
		Name:                 "DerivativesAnalyst",
		Tools:                AgentToolMap[AgentDerivativesAnalyst],
		SystemPromptTemplate: "agents/derivatives_analyst",
		MaxToolCalls:         10,
		MaxThinkingTokens:    3000,
		TimeoutPerTool:       10 * time.Second,
		TotalTimeout:         time.Minute,
		MaxCostPerRun:        0.05,
	},
	AgentStrategyPlanner: {
		Type:                 AgentStrategyPlanner,
		Name:                 "StrategyPlanner",
		Tools:                AgentToolMap[AgentStrategyPlanner],
		SystemPromptTemplate: "agents/strategy_planner",
		MaxToolCalls:         8,
		MaxThinkingTokens:    6000,
		TimeoutPerTool:       5 * time.Second,
		TotalTimeout:         time.Minute,
		MaxCostPerRun:        0.15,
	},
	AgentRiskManager: {
		Type:                 AgentRiskManager,
		Name:                 "RiskManager",
		Tools:                AgentToolMap[AgentRiskManager],
		SystemPromptTemplate: "agents/risk_manager",
		MaxToolCalls:         5,
		MaxThinkingTokens:    2000,
		TimeoutPerTool:       3 * time.Second,
		TotalTimeout:         30 * time.Second,
		MaxCostPerRun:        0.03,
	},
	AgentExecutor: {
		Type:                 AgentExecutor,
		Name:                 "Executor",
		Tools:                AgentToolMap[AgentExecutor],
		SystemPromptTemplate: "agents/executor",
		MaxToolCalls:         3,
		MaxThinkingTokens:    1500,
		TimeoutPerTool:       3 * time.Second,
		TotalTimeout:         30 * time.Second,
		MaxCostPerRun:        0.03,
	},
	AgentPositionManager: {
		Type:                 AgentPositionManager,
		Name:                 "PositionManager",
		Tools:                AgentToolMap[AgentPositionManager],
		SystemPromptTemplate: "agents/position_manager",
		MaxToolCalls:         5,
		MaxThinkingTokens:    2000,
		TimeoutPerTool:       5 * time.Second,
		TotalTimeout:         45 * time.Second,
		MaxCostPerRun:        0.03,
	},
	AgentSelfEvaluator: {
		Type:                 AgentSelfEvaluator,
		Name:                 "SelfEvaluator",
		Tools:                AgentToolMap[AgentSelfEvaluator],
		SystemPromptTemplate: "agents/self_evaluator",
		MaxToolCalls:         10,
		MaxThinkingTokens:    4000,
		TimeoutPerTool:       10 * time.Second,
		TotalTimeout:         90 * time.Second,
		MaxCostPerRun:        0.05,
	},
}
