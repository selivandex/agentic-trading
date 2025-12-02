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
// MVP: Only 3 agents - PortfolioArchitect, OpportunitySynthesizer, PortfolioManager
var DefaultAgentConfigs = map[AgentType]AgentConfig{
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
	AgentPortfolioManager: {
		Type:                 AgentPortfolioManager,
		Name:                 "PortfolioManager",
		Description:          "Portfolio Manager personalizing opportunities for individual clients",
		Tools:                ToolsForAgent(AgentPortfolioManager),
		OutputKey:            "trade_plan",
		SystemPromptTemplate: "agents/portfolio_manager",
		MaxToolCalls:         8,
		MaxThinkingTokens:    6000,
		TimeoutPerTool:       5 * time.Second,
		TotalTimeout:         time.Minute,
		MaxCostPerRun:        0.15,
	},
}
