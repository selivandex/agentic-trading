package agents

import "time"

// AgentConfig captures runtime settings for an ADK agent instance.
type AgentConfig struct {
	Type                 AgentType
	Name                 string
	Description          string
	AIProvider           string
	Model                string
	Tools                []string
	OutputKey            string
	SystemPromptTemplate string

	MaxToolCalls      int
	MaxThinkingTokens int
	TimeoutPerTool    time.Duration
	TotalTimeout      time.Duration
	MaxCostPerRun     float64
}

// DefaultAgentConfigs defines configuration for all active agents
// Phase 2 refactoring: Removed 8 analyst agents (now replaced by algorithmic tools)
var DefaultAgentConfigs = map[AgentType]AgentConfig{
	AgentStrategyPlanner: {
		Type:                 AgentStrategyPlanner,
		Name:                 "StrategyPlanner",
		Description:          "Planner synthesizing agent outputs into a trade plan",
		Tools:                ToolsForAgent(AgentStrategyPlanner),
		OutputKey:            "trade_plan",
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
		Description:          "Risk manager validating trades and sizing positions",
		Tools:                ToolsForAgent(AgentRiskManager),
		OutputKey:            "risk_assessment",
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
		Description:          "Executor responsible for placing and managing orders",
		Tools:                ToolsForAgent(AgentExecutor),
		OutputKey:            "execution_result",
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
		Description:          "Position manager monitoring live trades",
		Tools:                ToolsForAgent(AgentPositionManager),
		OutputKey:            "position_update",
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
		Description:          "Self evaluator reviewing performance",
		Tools:                ToolsForAgent(AgentSelfEvaluator),
		OutputKey:            "evaluation_report",
		SystemPromptTemplate: "agents/self_evaluator",
		MaxToolCalls:         10,
		MaxThinkingTokens:    4000,
		TimeoutPerTool:       10 * time.Second,
		TotalTimeout:         90 * time.Second,
		MaxCostPerRun:        0.05,
	},
	AgentOpportunitySynthesizer: {
		Type:                 AgentOpportunitySynthesizer,
		Name:                 "OpportunitySynthesizer",
		Description:          "Autonomous market analyzer calling technical, SMC, and market analysis tools directly for opportunity identification",
		Tools:                ToolsForAgent(AgentOpportunitySynthesizer),
		OutputKey:            "opportunity_decision",
		SystemPromptTemplate: "agents/opportunity_synthesizer",
		MaxToolCalls:         10,               // Increased: needs 3 analysis tools + publish_opportunity + save_memory
		MaxThinkingTokens:    8000,             // Increased: more complex reasoning (tool synthesis + decision)
		TimeoutPerTool:       15 * time.Second, // Increased: analysis tools may take longer
		TotalTimeout:         2 * time.Minute,  // Increased: 3 tools + LLM reasoning
		MaxCostPerRun:        0.10,             // Increased: longer reasoning, but still cheaper than 9 agents
	},
	AgentPortfolioArchitect: {
		Type:                 AgentPortfolioArchitect,
		Name:                 "PortfolioArchitect",
		Description:          "Designs diversified portfolio strategies based on capital, risk tolerance, and market conditions",
		Tools:                ToolsForAgent(AgentPortfolioArchitect),
		OutputKey:            "portfolio_strategy",
		SystemPromptTemplate: "agents/portfolio_architect",
		MaxToolCalls:         15,
		MaxThinkingTokens:    5000,
		TimeoutPerTool:       10 * time.Second,
		TotalTimeout:         2 * time.Minute,
		MaxCostPerRun:        0.10,
	},
}
