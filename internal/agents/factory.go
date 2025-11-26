package agents

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"

	"prometheus/internal/adapters/adk"
	"prometheus/internal/adapters/ai"
	"prometheus/internal/agents/callbacks"
	"prometheus/internal/agents/schemas"
	"prometheus/internal/tools"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"
)

// FactoryDeps gathers external dependencies needed to instantiate agents.
type FactoryDeps struct {
	AIRegistry     *ai.ProviderRegistry
	ToolRegistry   *tools.Registry
	Templates      *templates.Registry
	SessionService session.Service
	CallbackDeps   *CallbackDeps
}

// CallbackDeps contains dependencies for agent callbacks
type CallbackDeps struct {
	Redis       *redis.Client
	CostTracker callbacks.CostTracker
	RiskEngine  interface{}
	StatsRepo   interface{}
}

// Factory creates configured ADK agents.
type Factory struct {
	aiRegistry     *ai.ProviderRegistry
	toolRegistry   *tools.Registry
	templates      *templates.Registry
	sessionService session.Service
	callbackDeps   *CallbackDeps
	log            *logger.Logger
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

	// SessionService is optional for now (backwards compatibility)
	// Will become required after full migration

	// CallbackDeps is optional
	if deps.CallbackDeps == nil {
		deps.CallbackDeps = &CallbackDeps{}
	}

	return &Factory{
		aiRegistry:     deps.AIRegistry,
		toolRegistry:   deps.ToolRegistry,
		templates:      deps.Templates,
		sessionService: deps.SessionService,
		callbackDeps:   deps.CallbackDeps,
		log:            logger.Get().With("component", "agent_factory"),
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

	// Prepare callbacks
	agentCallbacks := f.buildAgentCallbacks()
	modelCallbacks := f.buildModelCallbacks()
	toolCallbacks := f.buildToolCallbacks()

	// Apply input/output schemas for critical agents
	inputSchema, outputSchema := getSchemaForAgent(cfg.Type)

	// Create REAL Google ADK LLM agent with callbacks and schemas
	return llmagent.New(llmagent.Config{
		Name:        cfg.Name,
		Description: cfg.Description,
		Model:       adkModel,
		Tools:       adkTools,
		Instruction: instruction,

		// Input/Output validation schemas
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,

		// Agent lifecycle callbacks
		BeforeAgentCallbacks: agentCallbacks.Before,
		AfterAgentCallbacks:  agentCallbacks.After,

		// Model callbacks
		BeforeModelCallbacks: modelCallbacks.Before,
		AfterModelCallbacks:  modelCallbacks.After,

		// Tool callbacks
		BeforeToolCallbacks: toolCallbacks.Before,
		AfterToolCallbacks:  toolCallbacks.After,
	})
}

// getSchemaForAgent returns input/output schemas for specific agent types
func getSchemaForAgent(agentType AgentType) (input, output *genai.Schema) {
	switch agentType {
	case AgentExecutor:
		return schemas.ExecutorInputSchema, schemas.ExecutorOutputSchema
	case AgentRiskManager:
		return nil, schemas.RiskManagerOutputSchema
	case AgentStrategyPlanner:
		return nil, schemas.StrategyPlannerOutputSchema
	case AgentMarketAnalyst:
		return nil, schemas.MarketAnalystOutputSchema
	default:
		return nil, nil
	}
}

// agentCallbacksSet contains before and after agent callbacks
type agentCallbacksSet struct {
	Before []agent.BeforeAgentCallback
	After  []agent.AfterAgentCallback
}

// modelCallbacksSet contains before and after model callbacks
type modelCallbacksSet struct {
	Before []llmagent.BeforeModelCallback
	After  []llmagent.AfterModelCallback
}

// toolCallbacksSet contains before and after tool callbacks
type toolCallbacksSet struct {
	Before []llmagent.BeforeToolCallback
	After  []llmagent.AfterToolCallback
}

// buildAgentCallbacks creates agent lifecycle callbacks
func (f *Factory) buildAgentCallbacks() agentCallbacksSet {
	var before []agent.BeforeAgentCallback
	var after []agent.AfterAgentCallback

	if f.callbackDeps != nil {
		// Cost tracking
		if f.callbackDeps.CostTracker != nil {
			before = append(before, callbacks.CostTrackingBeforeCallback(f.callbackDeps.CostTracker))
		}

		// Validation
		before = append(before, callbacks.ValidationBeforeCallback())

		// Metrics
		after = append(after, callbacks.MetricsAfterCallback(nil)) // TODO: pass stats repo

		// Reasoning log
		after = append(after, callbacks.ReasoningLogAfterCallback())
	}

	return agentCallbacksSet{Before: before, After: after}
}

// buildModelCallbacks creates model callbacks
func (f *Factory) buildModelCallbacks() modelCallbacksSet {
	var before []llmagent.BeforeModelCallback
	var after []llmagent.AfterModelCallback

	if f.callbackDeps != nil {
		// Response caching
		if f.callbackDeps.Redis != nil {
			before = append(before, callbacks.CachingBeforeModelCallback(f.callbackDeps.Redis))
			after = append(after, callbacks.SaveToCacheAfterModelCallback(f.callbackDeps.Redis, 1*time.Hour))
		}

		// Token counting
		if f.callbackDeps.CostTracker != nil {
			after = append(after, callbacks.TokenCountingCallback(f.callbackDeps.CostTracker))
		}

		// Rate limiting
		before = append(before, callbacks.RateLimitCallback())
	}

	return modelCallbacksSet{Before: before, After: after}
}

// buildToolCallbacks creates tool callbacks
func (f *Factory) buildToolCallbacks() toolCallbacksSet {
	var before []llmagent.BeforeToolCallback
	var after []llmagent.AfterToolCallback

	if f.callbackDeps != nil {
		// Risk validation - now uses Registry metadata instead of hardcoding tool names
		if f.toolRegistry != nil {
			before = append(before, callbacks.RiskValidationBeforeToolCallback(f.toolRegistry, f.callbackDeps.RiskEngine))
		}

		// Audit logging
		after = append(after, callbacks.AuditLogAfterToolCallback())

		// Error handling
		after = append(after, callbacks.ErrorHandlingCallback())
	}

	return toolCallbacksSet{Before: before, After: after}
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
