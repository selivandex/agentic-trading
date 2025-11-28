package agents

import "prometheus/internal/tools"

// ToolRequirement defines whether a tool is required or optional for an agent
type ToolRequirement string

const (
	ToolRequired ToolRequirement = "required"
	ToolOptional ToolRequirement = "optional"
)

// AgentToolRequirements defines which tools are required vs optional for each agent.
// This is used for prompt generation and validation.
var AgentToolRequirements = map[AgentType]map[string]ToolRequirement{
	AgentHeadOfResearch: {
		"publish_opportunity": ToolRequired, // Must publish decision
		"save_memory":         ToolOptional, // Can save reasoning
		"search_memory":       ToolOptional, // Can reference past patterns
	},
	AgentPortfolioManager: {
		"get_account_status": ToolRequired, // Must check account
		"pre_trade_check":    ToolRequired, // Must validate before trading
		"place_order":        ToolRequired, // Core function
		"save_memory":        ToolOptional, // Can save decisions
		"search_memory":      ToolOptional, // Can reference past trades
	},
	AgentTechnicalAnalyst: {
		"get_technical_analysis": ToolRequired, // Core analysis
		"get_market_analysis":    ToolRequired, // Market context
		"save_memory":            ToolOptional, // Can save patterns
		"search_memory":          ToolOptional, // Can reference history
	},
	AgentStructuralAnalyst: {
		"get_smc_analysis":    ToolRequired, // Core SMC analysis
		"get_market_analysis": ToolRequired, // Price context
		"save_memory":         ToolOptional, // Can save patterns
		"search_memory":       ToolOptional, // Can reference history
	},
	AgentFlowAnalyst: {
		"get_market_analysis": ToolRequired, // Order flow data
		"save_memory":         ToolOptional, // Can save patterns
		"search_memory":       ToolOptional, // Can reference history
	},
	AgentPortfolioArchitect: {
		"get_account_status": ToolRequired, // Must check account
		"pre_trade_check":    ToolRequired, // Must validate
		"place_order":        ToolRequired, // Execute initial allocation
		"save_memory":        ToolRequired, // Must save portfolio design
	},
	// TODO: Add requirements for other agents as needed
}

// AgentToolCategories defines the tool categories each agent can access.
// Phase 2 refactoring: Removed 8 analyst agents - analysis now done via algorithmic tools
var AgentToolCategories = map[AgentType][]string{
	// Personal trading workflow agents (per-user decision making)
	// Portfolio Manager (per-client decision making)
	AgentPortfolioManager: {"market_data", "account", "risk", "execution", "memory"},
	AgentPositionManager:  {"account", "execution", "memory"},

	// Market research agent (global opportunity identification)
	AgentOpportunitySynthesizer: {"market_data", "smc", "memory"}, // Direct access to technical, SMC, and market analysis tools

	// Research Committee agents (Phase 3: multi-agent collaboration)
	AgentTechnicalAnalyst:  {"market_data", "memory"}, // Technical analysis tools
	AgentStructuralAnalyst: {"smc", "memory"},         // SMC analysis tools
	AgentFlowAnalyst:       {"market_data", "memory"}, // Market flow analysis tools
	AgentMacroAnalyst:      {"correlation", "memory"}, // Correlation analysis tools
	AgentHeadOfResearch:    {"memory", "market"},      // Memory + publish_opportunity tool

	// Portfolio management agent (onboarding)
	// Added smc category for market snapshot, execution for execute_trade tool, risk for pre_trade_check
	AgentPortfolioArchitect: {"market_data", "smc", "correlation", "account", "execution", "risk", "memory"},

	// System agents (adaptive intelligence)
	AgentRegimeDetector:      {"market_data", "derivatives", "memory"},
	AgentPerformanceAnalyzer: {"evaluation", "memory"},

	// Quality gates and review agents
	AgentPreTradeReviewer:     {"risk", "memory"},
	AgentPerformanceCommittee: {"evaluation", "memory"},
}

// AgentToolMap resolves tool names per agent by filtering the global catalog by category.
var AgentToolMap = buildAgentToolMap()

func buildAgentToolMap() map[AgentType][]string {
	result := make(map[AgentType][]string, len(AgentToolCategories))
	defs := tools.Definitions()

	for agentType, categories := range AgentToolCategories {
		categorySet := make(map[string]struct{}, len(categories))
		for _, category := range categories {
			categorySet[category] = struct{}{}
		}

		for _, def := range defs {
			if _, ok := categorySet[def.Category]; ok {
				result[agentType] = append(result[agentType], def.Name)
			}
		}
	}

	return result
}

// ToolsForAgent returns a copy of the allowed tools for a given agent type.
func ToolsForAgent(agentType AgentType) []string {
	tools := AgentToolMap[agentType]
	res := make([]string, len(tools))
	copy(res, tools)
	return res
}

// ValidateToolAccess checks if an agent has access to a specific tool
func ValidateToolAccess(agentType AgentType, toolName string) bool {
	allowedTools := AgentToolMap[agentType]

	for _, allowed := range allowedTools {
		if allowed == toolName {
			return true
		}
	}

	return false
}

// GetToolRequirement returns whether a tool is required or optional for a given agent.
// Returns empty string if the tool requirement is not explicitly defined.
func GetToolRequirement(agentType AgentType, toolName string) ToolRequirement {
	if requirements, ok := AgentToolRequirements[agentType]; ok {
		if req, found := requirements[toolName]; found {
			return req
		}
	}
	return "" // Not explicitly defined - defaults to optional
}

// IsToolRequired returns true if the tool is marked as required for the agent
func IsToolRequired(agentType AgentType, toolName string) bool {
	return GetToolRequirement(agentType, toolName) == ToolRequired
}
