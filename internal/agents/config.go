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
	// Research Committee agents (Phase 3: multi-agent collaboration)
	AgentTechnicalAnalyst: {
		Type:                 AgentTechnicalAnalyst,
		Name:                 "TechnicalAnalyst",
		Description:          "Technical indicators specialist analyzing momentum, trend, and volatility patterns",
		Tools:                ToolsForAgent(AgentTechnicalAnalyst),
		OutputKey:            "analyst_report",
		SystemPromptTemplate: "agents/technical_analyst",
		MaxToolCalls:         3,
		MaxThinkingTokens:    2000,
		TimeoutPerTool:       5 * time.Second,
		TotalTimeout:         30 * time.Second,
		MaxCostPerRun:        0.03,
	},
	AgentStructuralAnalyst: {
		Type:                 AgentStructuralAnalyst,
		Name:                 "StructuralAnalyst",
		Description:          "Smart Money Concepts specialist analyzing market structure, order blocks, and liquidity zones",
		Tools:                ToolsForAgent(AgentStructuralAnalyst),
		OutputKey:            "analyst_report",
		SystemPromptTemplate: "agents/structural_analyst",
		MaxToolCalls:         3,
		MaxThinkingTokens:    2000,
		TimeoutPerTool:       5 * time.Second,
		TotalTimeout:         30 * time.Second,
		MaxCostPerRun:        0.03,
	},
	AgentFlowAnalyst: {
		Type:                 AgentFlowAnalyst,
		Name:                 "FlowAnalyst",
		Description:          "Order flow specialist analyzing real-time market activity, whale behavior, and positioning",
		Tools:                ToolsForAgent(AgentFlowAnalyst),
		OutputKey:            "analyst_report",
		SystemPromptTemplate: "agents/flow_analyst",
		MaxToolCalls:         3,
		MaxThinkingTokens:    2000,
		TimeoutPerTool:       5 * time.Second,
		TotalTimeout:         30 * time.Second,
		MaxCostPerRun:        0.03,
	},
	AgentMacroAnalyst: {
		Type:                 AgentMacroAnalyst,
		Name:                 "MacroAnalyst",
		Description:          "Economic context specialist analyzing correlations, risk events, and broader market conditions",
		Tools:                ToolsForAgent(AgentMacroAnalyst),
		OutputKey:            "analyst_report",
		SystemPromptTemplate: "agents/macro_analyst",
		MaxToolCalls:         3,
		MaxThinkingTokens:    2000,
		TimeoutPerTool:       5 * time.Second,
		TotalTimeout:         30 * time.Second,
		MaxCostPerRun:        0.03,
	},
	AgentHeadOfResearch: {
		Type:                 AgentHeadOfResearch,
		Name:                 "HeadOfResearch",
		Description:          "Research synthesizer leading debate, resolving conflicts, and making final publish/skip decisions",
		Tools:                ToolsForAgent(AgentHeadOfResearch),
		OutputKey:            "research_decision",
		SystemPromptTemplate: "agents/head_of_research",
		MaxToolCalls:         8,
		MaxThinkingTokens:    6000,
		TimeoutPerTool:       10 * time.Second,
		TotalTimeout:         60 * time.Second,
		MaxCostPerRun:        0.08,
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
	AgentRegimeDetector: {
		Type:                 AgentRegimeDetector,
		Name:                 "RegimeDetector",
		Description:          "Detects market regime and adjusts system parameters for adaptive trading",
		Tools:                ToolsForAgent(AgentRegimeDetector),
		OutputKey:            "regime_classification",
		SystemPromptTemplate: "agents/regime_detector",
		MaxToolCalls:         8,
		MaxThinkingTokens:    3000,
		TimeoutPerTool:       10 * time.Second,
		TotalTimeout:         90 * time.Second,
		MaxCostPerRun:        0.05,
	},
	AgentRegimeInterpreter: {
		Type:                 AgentRegimeInterpreter,
		Name:                 "RegimeInterpreter",
		Description:          "Interprets ML regime classification and provides strategic recommendations",
		Tools:                []string{}, // No tools needed, pure interpretation
		OutputKey:            "interpretation",
		SystemPromptTemplate: "agents/regime_interpreter",
		MaxToolCalls:         0,
		MaxThinkingTokens:    3000,
		TimeoutPerTool:       10 * time.Second,
		TotalTimeout:         30 * time.Second,
		MaxCostPerRun:        0.05,
	},
	AgentPerformanceAnalyzer: {
		Type:                 AgentPerformanceAnalyzer,
		Name:                 "PerformanceAnalyzer",
		Description:          "Analyzes trade outcomes and extracts patterns for system improvement",
		Tools:                ToolsForAgent(AgentPerformanceAnalyzer),
		OutputKey:            "performance_analysis",
		SystemPromptTemplate: "agents/performance_analyzer",
		MaxToolCalls:         15,
		MaxThinkingTokens:    5000,
		TimeoutPerTool:       10 * time.Second,
		TotalTimeout:         3 * time.Minute,
		MaxCostPerRun:        0.08,
	},
	AgentPreTradeReviewer: {
		Type:                 AgentPreTradeReviewer,
		Name:                 "PreTradeReviewer",
		Description:          "Quality gate challenging trade plans before execution",
		Tools:                ToolsForAgent(AgentPreTradeReviewer),
		OutputKey:            "review_decision",
		SystemPromptTemplate: "agents/pre_trade_reviewer",
		MaxToolCalls:         5,
		MaxThinkingTokens:    3000,
		TimeoutPerTool:       5 * time.Second,
		TotalTimeout:         30 * time.Second,
		MaxCostPerRun:        0.03,
	},
	AgentPerformanceCommittee: {
		Type:                 AgentPerformanceCommittee,
		Name:                 "PerformanceCommittee",
		Description:          "Weekly performance review extracting patterns and lessons",
		Tools:                ToolsForAgent(AgentPerformanceCommittee),
		OutputKey:            "performance_report",
		SystemPromptTemplate: "agents/performance_committee",
		MaxToolCalls:         15,
		MaxThinkingTokens:    5000,
		TimeoutPerTool:       10 * time.Second,
		TotalTimeout:         3 * time.Minute,
		MaxCostPerRun:        0.08,
	},
}
