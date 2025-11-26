package agents

import "prometheus/internal/tools"

// AgentToolCategories defines the tool categories each agent can access.
var AgentToolCategories = map[AgentType][]string{
	// Analysts - get memory tools to save their analysis via CoT
	AgentMarketAnalyst:      {"market_data", "momentum", "volatility", "trend", "volume", "smc", "memory"},
	AgentSMCAnalyst:         {"market_data", "smc", "memory"},
	AgentSentimentAnalyst:   {"sentiment", "memory"},
	AgentOnChainAnalyst:     {"onchain", "memory"},
	AgentMacroAnalyst:       {"macro", "memory"},
	AgentOrderFlowAnalyst:   {"market_data", "order_flow", "memory"},
	AgentDerivativesAnalyst: {"market_data", "derivatives", "memory"},
	AgentCorrelationAnalyst: {"market_data", "correlation", "memory"},
	
	// Decision makers - also need memory for saving plans and decisions
	AgentStrategyPlanner:    {"memory"},
	AgentRiskManager:        {"account", "risk", "memory"},
	AgentExecutor:           {"account", "execution", "memory"},
	AgentPositionManager:    {"account", "execution", "memory"},
	AgentSelfEvaluator:      {"evaluation", "memory"},
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
