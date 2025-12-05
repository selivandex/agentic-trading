package agent

import (
	"context"

	"prometheus/internal/adapters/ai"
	"prometheus/internal/domain/agent"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"
)

// TemplateRenderer defines interface for rendering templates
type TemplateRenderer interface {
	Render(templatePath string, data any) (string, error)
}

// Service handles agent management (application layer)
// Orchestrates agent operations and template loading
type Service struct {
	domainService agent.DomainService // Domain service interface
	templates     TemplateRenderer
	log           *logger.Logger
}

// NewService creates a new agent application service
func NewService(domainService agent.DomainService, tmpl TemplateRenderer) *Service {
	if tmpl == nil {
		tmpl = templates.Get()
	}

	return &Service{
		domainService: domainService,
		templates:     tmpl,
		log:           logger.Get().With("component", "agent_app_service"),
	}
}

// EnsureSystemAgents ensures all system agents exist (FindOrCreate pattern)
// Loads prompts from templates and creates agents if they don't exist
func (s *Service) EnsureSystemAgents(ctx context.Context) error {
	s.log.Info("Ensuring system agents...")

	definitions := s.getSystemAgentDefinitions()
	created := 0
	existing := 0

	for _, def := range definitions {
		// Load system prompt from template
		systemPrompt, err := s.loadPromptFromTemplate(def.Identifier)
		if err != nil {
			s.log.Warnw("Failed to load prompt template, using placeholder",
				"agent", def.Identifier,
				"error", err,
			)
			systemPrompt = "System prompt for " + def.Name + " (template not found)"
		}

		// Create agent entity
		a := &agent.Agent{
			Identifier:     def.Identifier,
			Name:           def.Name,
			Description:    def.Description,
			Category:       def.Category,
			SystemPrompt:   systemPrompt,
			Instructions:   def.Instructions,
			ModelProvider:  def.ModelProvider,
			ModelName:      def.ModelName,
			Temperature:    def.Temperature,
			MaxTokens:      def.MaxTokens,
			MaxCostPerRun:  def.MaxCostPerRun,
			TimeoutSeconds: def.TimeoutSeconds,
			IsActive:       true,
		}

		// Set available tools
		if err := a.SetAvailableTools(def.AvailableTools); err != nil {
			s.log.Warnw("Failed to set available tools",
				"agent", def.Identifier,
				"error", err,
			)
		}

		// FindOrCreate via domain service
		resultAgent, wasCreated, err := s.domainService.FindOrCreate(ctx, a)
		if err != nil {
			s.log.Errorw("Failed to ensure agent",
				"agent", def.Identifier,
				"error", err,
			)
			continue
		}

		if wasCreated {
			created++
			s.log.Infow("✓ Agent created",
				"agent", def.Identifier,
				"id", resultAgent.ID,
				"version", resultAgent.Version,
			)
		} else {
			existing++
			s.log.Debugw("Agent already exists",
				"agent", def.Identifier,
				"id", resultAgent.ID,
				"version", resultAgent.Version,
			)
		}
	}

	s.log.Infow("✓ System agents initialized",
		"created", created,
		"existing", existing,
		"total", len(definitions),
	)

	return nil
}

// GetByIdentifier retrieves agent by identifier
func (s *Service) GetByIdentifier(ctx context.Context, identifier string) (*agent.Agent, error) {
	return s.domainService.GetByIdentifier(ctx, identifier)
}

// UpdatePrompt updates agent's system prompt (for tuning)
func (s *Service) UpdatePrompt(ctx context.Context, identifier, newPrompt string) error {
	a, err := s.domainService.GetByIdentifier(ctx, identifier)
	if err != nil {
		return errors.Wrap(err, "get agent")
	}

	a.SystemPrompt = newPrompt

	if err := s.domainService.Update(ctx, a); err != nil {
		return errors.Wrap(err, "update agent")
	}

	s.log.Infow("Agent prompt updated",
		"agent", identifier,
		"version", a.Version, // Auto-incremented by DB trigger
	)

	return nil
}

// ListActive retrieves all active agents
func (s *Service) ListActive(ctx context.Context) ([]*agent.Agent, error) {
	return s.domainService.ListActive(ctx)
}

// ListByCategory retrieves agents by category
func (s *Service) ListByCategory(ctx context.Context, category string) ([]*agent.Agent, error) {
	return s.domainService.ListByCategory(ctx, category)
}

// loadPromptFromTemplate loads agent prompt from template file
func (s *Service) loadPromptFromTemplate(identifier string) (string, error) {
	templatePath := "prompts/agents/" + identifier
	prompt, err := s.templates.Render(templatePath, map[string]interface{}{
		// Base template without dynamic data
		// Dynamic data will be injected when agent runs
	})
	if err != nil {
		return "", errors.Wrap(err, "render template")
	}
	return prompt, nil
}

// agentDefinition holds agent metadata for initialization
type agentDefinition struct {
	Identifier     string
	Name           string
	Description    string
	Category       string
	Instructions   string
	ModelProvider  ai.ProviderName
	ModelName      ai.ProviderModelName
	Temperature    float64
	MaxTokens      int
	AvailableTools []string
	MaxCostPerRun  float64
	TimeoutSeconds int
}

// getSystemAgentDefinitions returns definitions for all system agents
func (s *Service) getSystemAgentDefinitions() []agentDefinition {
	return []agentDefinition{
		{
			Identifier:     agent.IdentifierPortfolioArchitect,
			Name:           "Portfolio Architect",
			Description:    "Designs initial portfolio allocation for new users",
			Category:       agent.CategoryCoordinator,
			ModelProvider:  ai.ProviderNameDeepSeek,
			ModelName:      ai.ModelDeepSeekReasoner,
			Temperature:    1.0,
			MaxTokens:      8000,
			AvailableTools: []string{"get_technical_analysis", "get_smc_analysis", "get_market_analysis", "calc_asset_correlation", "pre_trade_check", "place_order", "save_memory"},
			MaxCostPerRun:  0.50,
			TimeoutSeconds: 180,
		},
		{
			Identifier:     agent.IdentifierPortfolioManager,
			Name:           "Portfolio Manager",
			Description:    "Reviews portfolio and makes rebalancing decisions",
			Category:       agent.CategoryCoordinator,
			ModelProvider:  ai.ProviderNameDeepSeek,
			ModelName:      ai.ModelDeepSeekReasoner,
			Temperature:    1.0,
			MaxTokens:      8000,
			AvailableTools: []string{"get_account_status", "call_technical_analyzer", "call_smc_analyzer", "call_correlation_analyzer", "pre_trade_check", "place_order", "save_memory"},
			MaxCostPerRun:  0.50,
			TimeoutSeconds: 180,
		},
		{
			Identifier:     agent.IdentifierTechnicalAnalyzer,
			Name:           "Technical Analyzer",
			Description:    "Expert in price action, indicators, momentum",
			Category:       agent.CategoryExpert,
			ModelProvider:  ai.ProviderNameDeepSeek,
			ModelName:      ai.ModelDeepSeekReasoner,
			Temperature:    1.0,
			MaxTokens:      4000,
			AvailableTools: []string{"get_ohlcv", "get_price", "calc_indicators"},
			MaxCostPerRun:  0.10,
			TimeoutSeconds: 60,
		},
		{
			Identifier:     agent.IdentifierSMCAnalyzer,
			Name:           "SMC Analyzer",
			Description:    "Smart Money Concepts - market structure, liquidity, order flow",
			Category:       agent.CategoryExpert,
			ModelProvider:  ai.ProviderNameDeepSeek,
			ModelName:      ai.ModelDeepSeekReasoner,
			Temperature:    1.0,
			MaxTokens:      4000,
			AvailableTools: []string{"get_ohlcv", "get_orderbook", "get_market_flow"},
			MaxCostPerRun:  0.10,
			TimeoutSeconds: 60,
		},
		{
			Identifier:     agent.IdentifierCorrelationAnalyzer,
			Name:           "Correlation Analyzer",
			Description:    "Analyzes correlation between assets for diversification",
			Category:       agent.CategoryExpert,
			ModelProvider:  ai.ProviderNameDeepSeek,
			ModelName:      ai.ModelDeepSeekReasoner,
			Temperature:    1.0,
			MaxTokens:      4000,
			AvailableTools: []string{"get_ohlcv", "calc_asset_correlation"},
			MaxCostPerRun:  0.05,
			TimeoutSeconds: 30,
		},
		{
			Identifier:     agent.IdentifierOpportunitySynthesizer,
			Name:           "Opportunity Synthesizer",
			Description:    "Synthesizes market research from expert analysts",
			Category:       agent.CategoryCoordinator,
			ModelProvider:  ai.ProviderNameDeepSeek,
			ModelName:      ai.ModelDeepSeekReasoner,
			Temperature:    1.0,
			MaxTokens:      6000,
			AvailableTools: []string{"publish_opportunity", "save_memory"},
			MaxCostPerRun:  0.30,
			TimeoutSeconds: 120,
		},
	}
}
