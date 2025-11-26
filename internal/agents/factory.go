package agents

import (
	"context"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/tool"

	"prometheus/internal/adapters/adk"
	"prometheus/internal/adapters/ai"
	"prometheus/internal/tools"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"
)

// FactoryDeps gathers external dependencies needed to instantiate agents.
type FactoryDeps struct {
	AIRegistry   *ai.ProviderRegistry
	ToolRegistry *tools.Registry
	Templates    *templates.Registry
}

// Factory creates configured ADK agents.
type Factory struct {
	aiRegistry   *ai.ProviderRegistry
	toolRegistry *tools.Registry
	templates    *templates.Registry
	log          *logger.Logger
}

// NewFactory builds an agent factory with required dependencies.
func NewFactory(deps FactoryDeps) (*Factory, error) {
	if deps.ToolRegistry == nil {
		return nil, errors.ErrInvalidInput
	}

	if deps.AIRegistry == nil {
		return nil, errors.ErrInvalidInput
	}

	if deps.Templates == nil {
		deps.Templates = templates.Get()
	}

	return &Factory{
		aiRegistry:   deps.AIRegistry,
		toolRegistry: deps.ToolRegistry,
		templates:    deps.Templates,
		log:          logger.Get().With("component", "agent_factory"),
	}, nil
}

// CreateAgent constructs a pure ADK agent instance from a config.
func (f *Factory) CreateAgent(cfg AgentConfig) (agent.Agent, error) {
	// Resolve model info
	modelInfo, err := f.aiRegistry.ResolveModel(context.Background(), cfg.AIProvider, cfg.Model)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve model %s/%s", cfg.AIProvider, cfg.Model)
	}

	// Get AI provider (must support chat)
	provider, err := f.aiRegistry.Get(cfg.AIProvider)
	if err != nil {
		return nil, errors.Wrapf(err, "AI provider %s not found", cfg.AIProvider)
	}

	chatProvider, ok := provider.(ai.ChatProvider)
	if !ok {
		return nil, errors.Wrapf(errors.ErrInvalidInput,
			"provider %s does not support chat completions", cfg.AIProvider)
	}

	// Create ADK model adapter (bridges our AI provider to ADK model interface)
	adkModel := adk.NewModelAdapter(chatProvider, modelInfo.Name)

	// Collect tools for ADK agent
	adkTools := make([]tool.Tool, 0, len(cfg.Tools))
	toolInfo := make([]tools.Definition, 0, len(cfg.Tools))
	definitionByName := map[string]tools.Definition{}
	for _, def := range tools.Definitions() {
		definitionByName[def.Name] = def
	}

	for _, toolName := range cfg.Tools {
		t, ok := f.toolRegistry.Get(toolName)
		if !ok {
			return nil, errors.Wrapf(errors.ErrNotFound, "tool not found: %s", toolName)
		}

		// Our tools are already properly typed ADK tool.Tool instances
		// They were created via functiontool.New() with tool.Context signature
		adkTools = append(adkTools, t)

		if def, ok := definitionByName[toolName]; ok {
			toolInfo = append(toolInfo, def)
		} else {
			toolInfo = append(toolInfo, tools.Definition{Name: toolName, Description: ""})
		}
	}

	// Render system prompt from template
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
			return nil, errors.Wrapf(err, "render prompt for %s", cfg.Name)
		}
	}

	f.log.Debugf("Creating ADK agent: %s with model %s and %d tools",
		cfg.Name, modelInfo.Name, len(adkTools))

	// Create REAL Google ADK LLM agent
	return llmagent.New(llmagent.Config{
		Name:        cfg.Name,
		Description: cfg.Description,
		Model:       adkModel,
		Tools:       adkTools,
		Instruction: instruction,
	})
}

// CreateDefaultRegistry builds and registers agents using DefaultAgentConfigs.
func (f *Factory) CreateDefaultRegistry(provider, model string) (*Registry, error) {
	reg := NewRegistry()

	f.log.Infof("Creating default agent registry with provider=%s model=%s", provider, model)

	for agentType, cfg := range DefaultAgentConfigs {
		cfg.AIProvider = provider
		cfg.Model = model

		ag, err := f.CreateAgent(cfg)
		if err != nil {
			return nil, errors.Wrapf(err, "create agent %s", agentType)
		}

		reg.Register(cfg.Type, ag)
		f.log.Debugf("Registered agent: %s (%s)", cfg.Name, agentType)
	}

	f.log.Infof("Agent registry created with %d agents", len(reg.List()))
	return reg, nil
}

// CreateTradingPipeline creates a strategy planner agent.
// TODO: Extend to multi-agent workflow using ADK workflow agents when needed.
func (f *Factory) CreateTradingPipeline(userCfg UserAgentConfig) (agent.Agent, error) {
	// For now, create a strategy planner that can orchestrate via tools
	strategyCfg := DefaultAgentConfigs[AgentStrategyPlanner]
	strategyCfg.AIProvider = userCfg.AIProvider
	strategyCfg.Model = userCfg.Model

	return f.CreateAgent(strategyCfg)
}

// CreateAgentForUser creates a specific agent for a user with custom config.
func (f *Factory) CreateAgentForUser(
	agentType AgentType,
	provider string,
	model string,
) (agent.Agent, error) {
	cfg, ok := DefaultAgentConfigs[agentType]
	if !ok {
		return nil, errors.Wrapf(errors.ErrNotFound, "agent type %s not found", agentType)
	}

	cfg.AIProvider = provider
	cfg.Model = model

	return f.CreateAgent(cfg)
}
