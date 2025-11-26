package agents

import "prometheus/internal/tools"

// AgentToolCategories defines the tool categories each agent can access.
var AgentToolCategories = map[AgentType][]string{
	AgentMarketAnalyst:      {"market_data", "momentum", "volatility", "trend", "volume", "smc"},
	AgentSMCAnalyst:         {"market_data", "smc"},
	AgentSentimentAnalyst:   {"sentiment"},
	AgentOnChainAnalyst:     {"onchain"},
	AgentMacroAnalyst:       {"macro"},
	AgentOrderFlowAnalyst:   {"market_data", "order_flow"},
	AgentDerivativesAnalyst: {"market_data", "derivatives"},
	AgentCorrelationAnalyst: {"market_data", "correlation"},
	AgentStrategyPlanner:    {"memory"},
	AgentRiskManager:        {"account", "risk"},
	AgentExecutor:           {"account", "execution"},
	AgentPositionManager:    {"account", "execution"},
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
