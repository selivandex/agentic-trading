package agents

import (
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/agenttool"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// CreateExpertTools creates agent-as-tool wrappers for specialized expert agents
// These can be used by higher-level agents (like Strategy Planner) to consult experts
func (f *Factory) CreateExpertTools(provider, model string) (map[string]tool.Tool, error) {
	log := logger.Get().With("component", "expert_tools")

	expertTools := make(map[string]tool.Tool)

	// Macro analyst expert
	macroAgent, err := f.CreateAgentForUser(AgentMacroAnalyst, provider, model)
	if err != nil {
		log.Warnf("Failed to create macro analyst: %v", err)
	} else {
		expertTools["consult_macro_expert"] = agenttool.New(macroAgent, &agenttool.Config{
			SkipSummarization: false, // Allow summarization for concise expert responses
		})
		log.Debug("Registered macro expert tool")
	}

	// On-chain analyst expert
	onchainAgent, err := f.CreateAgentForUser(AgentOnChainAnalyst, provider, model)
	if err != nil {
		log.Warnf("Failed to create onchain analyst: %v", err)
	} else {
		expertTools["consult_onchain_expert"] = agenttool.New(onchainAgent, nil)
		log.Debug("Registered onchain expert tool")
	}

	// Correlation analyst expert
	correlationAgent, err := f.CreateAgentForUser(AgentCorrelationAnalyst, provider, model)
	if err != nil {
		log.Warnf("Failed to create correlation analyst: %v", err)
	} else {
		expertTools["consult_correlation_expert"] = agenttool.New(correlationAgent, nil)
		log.Debug("Registered correlation expert tool")
	}

	// Derivatives analyst expert
	derivativesAgent, err := f.CreateAgentForUser(AgentDerivativesAnalyst, provider, model)
	if err != nil {
		log.Warnf("Failed to create derivatives analyst: %v", err)
	} else {
		expertTools["consult_derivatives_expert"] = agenttool.New(derivativesAgent, nil)
		log.Debug("Registered derivatives expert tool")
	}

	if len(expertTools) == 0 {
		return nil, errors.New("no expert tools could be created")
	}

	log.Infof("Created %d expert agent tools", len(expertTools))
	return expertTools, nil
}
