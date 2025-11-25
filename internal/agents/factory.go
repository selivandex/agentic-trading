package agents

import (
	"context"
	"fmt"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	parallelagent "google.golang.org/adk/agent/workflowagents/parallelagent"
	sequentialagent "google.golang.org/adk/agent/workflowagents/sequentialagent"
	adkmodel "google.golang.org/adk/model"
	adktool "google.golang.org/adk/tool"

	"prometheus/internal/adapters/ai"
	"prometheus/internal/tools"
	"prometheus/pkg/templates"
)

// FactoryDeps gathers external dependencies needed to instantiate agents.
type FactoryDeps struct {
	AIRegistry   *ai.ProviderRegistry
	ToolRegistry *tools.Registry
	Templates    *templates.Registry
}

// Factory creates configured agents and registries.
type Factory struct {
	aiRegistry   *ai.ProviderRegistry
	toolRegistry *tools.Registry
	templates    *templates.Registry
}

// NewFactory builds an agent factory with required dependencies.
func NewFactory(deps FactoryDeps) (*Factory, error) {
	if deps.ToolRegistry == nil {
		return nil, fmt.Errorf("tool registry is required")
	}

	if deps.AIRegistry == nil {
		return nil, fmt.Errorf("AI provider registry is required")
	}

	if deps.Templates == nil {
		deps.Templates = templates.Get()
	}

	return &Factory{aiRegistry: deps.AIRegistry, toolRegistry: deps.ToolRegistry, templates: deps.Templates}, nil
}

// CreateAgent constructs a single ADK agent instance from a config.
func (f *Factory) CreateAgent(cfg AgentConfig) (agent.Agent, error) {
	modelInfo, err := f.aiRegistry.ResolveModel(context.Background(), cfg.AIProvider, cfg.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve model %s/%s: %w", cfg.AIProvider, cfg.Model, err)
	}

	llmModel := adkmodel.BasicModel{ID: modelInfo.Name, ProviderID: cfg.AIProvider, Tokens: modelInfo.MaxTokens}

	agentTools := make([]adktool.Tool, 0, len(cfg.Tools))
	toolInfo := make([]tools.Definition, 0, len(cfg.Tools))
	definitionByName := map[string]tools.Definition{}
	for _, def := range tools.Definitions() {
		definitionByName[def.Name] = def
	}

	for _, toolName := range cfg.Tools {
		t, ok := f.toolRegistry.Get(toolName)
		if !ok {
			return nil, fmt.Errorf("tool not found: %s", toolName)
		}
		agentTools = append(agentTools, t)
		if def, ok := definitionByName[toolName]; ok {
			toolInfo = append(toolInfo, def)
		} else {
			toolInfo = append(toolInfo, tools.Definition{Name: toolName, Description: ""})
		}
	}

	instruction := ""
	if cfg.SystemPromptTemplate != "" {
		data := map[string]interface{}{
			"Tools":        toolInfo,
			"MaxToolCalls": cfg.MaxToolCalls,
			"AgentName":    cfg.Name,
			"AgentType":    cfg.Type,
		}
		instruction, err = f.templates.Render(cfg.SystemPromptTemplate, data)
		if err != nil {
			return nil, fmt.Errorf("render prompt for %s: %w", cfg.Name, err)
		}
	}

	return llmagent.New(llmagent.Config{
		Name:        cfg.Name,
		Description: cfg.Description,
		Model:       llmModel,
		Tools:       agentTools,
		Instruction: instruction,
		OutputKey:   cfg.OutputKey,
	})
}

// CreateDefaultRegistry builds and registers agents using DefaultAgentConfigs.
func (f *Factory) CreateDefaultRegistry(provider, model string) (*Registry, error) {
	reg := NewRegistry()

	for _, cfg := range DefaultAgentConfigs {
		cfg.AIProvider = provider
		cfg.Model = model
		ag, err := f.CreateAgent(cfg)
		if err != nil {
			return nil, err
		}
		reg.Register(cfg.Type, ag)
	}

	return reg, nil
}

// CreateTradingPipeline creates the full trading workflow using default configs.
func (f *Factory) CreateTradingPipeline(userCfg UserAgentConfig) (agent.Agent, error) {
	marketCfg := DefaultAgentConfigs[AgentMarketAnalyst]
	marketCfg.AIProvider, marketCfg.Model = userCfg.AIProvider, userCfg.Model
	marketAnalyst, err := f.CreateAgent(marketCfg)
	if err != nil {
		return nil, err
	}

	sentimentCfg := DefaultAgentConfigs[AgentSentimentAnalyst]
	sentimentCfg.AIProvider, sentimentCfg.Model = userCfg.AIProvider, userCfg.Model
	sentimentAnalyst, err := f.CreateAgent(sentimentCfg)
	if err != nil {
		return nil, err
	}

	onchainCfg := DefaultAgentConfigs[AgentOnChainAnalyst]
	onchainCfg.AIProvider, onchainCfg.Model = userCfg.AIProvider, userCfg.Model
	onchainAnalyst, err := f.CreateAgent(onchainCfg)
	if err != nil {
		return nil, err
	}

	parallelAnalysis, err := parallelagent.New(parallelagent.Config{Config: agent.Config{
		Name:        "ParallelAnalysis",
		Description: "Concurrent market, sentiment, and on-chain analysis",
		SubAgents:   []agent.Agent{marketAnalyst, sentimentAnalyst, onchainAnalyst},
	}})
	if err != nil {
		return nil, err
	}

	correlationCfg := DefaultAgentConfigs[AgentCorrelationAnalyst]
	correlationCfg.AIProvider, correlationCfg.Model = userCfg.AIProvider, userCfg.Model
	correlationAnalyst, err := f.CreateAgent(correlationCfg)
	if err != nil {
		return nil, err
	}

	strategyCfg := DefaultAgentConfigs[AgentStrategyPlanner]
	strategyCfg.AIProvider, strategyCfg.Model = userCfg.AIProvider, userCfg.Model
	strategyPlanner, err := f.CreateAgent(strategyCfg)
	if err != nil {
		return nil, err
	}

	riskCfg := DefaultAgentConfigs[AgentRiskManager]
	riskCfg.AIProvider, riskCfg.Model = userCfg.AIProvider, userCfg.Model
	riskManager, err := f.CreateAgent(riskCfg)
	if err != nil {
		return nil, err
	}

	executorCfg := DefaultAgentConfigs[AgentExecutor]
	executorCfg.AIProvider, executorCfg.Model = userCfg.AIProvider, userCfg.Model
	executor, err := f.CreateAgent(executorCfg)
	if err != nil {
		return nil, err
	}

	return sequentialagent.New(sequentialagent.Config{Config: agent.Config{
		Name:        "TradingPipeline",
		Description: "Complete trading analysis and execution workflow",
		SubAgents: []agent.Agent{
			parallelAnalysis,
			correlationAnalyst,
			strategyPlanner,
			riskManager,
			executor,
		},
	}})
}
