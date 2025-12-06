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

// List retrieves all agents
func (s *Service) List(ctx context.Context) ([]*agent.Agent, error) {
	return s.domainService.List(ctx)
}

// GetByID retrieves agent by ID
func (s *Service) GetByID(ctx context.Context, id int) (*agent.Agent, error) {
	return s.domainService.GetByID(ctx, id)
}

// CreateAgentParams contains parameters for creating a new agent
type CreateAgentParams struct {
	Identifier     string
	Name           string
	Description    string
	Category       string
	SystemPrompt   string
	Instructions   string
	ModelProvider  ai.ProviderName
	ModelName      ai.ProviderModelName
	Temperature    float64
	MaxTokens      int
	AvailableTools []string
	MaxCostPerRun  float64
	TimeoutSeconds int
	IsActive       bool
}

// CreateAgent creates a new agent (admin function)
func (s *Service) CreateAgent(ctx context.Context, params CreateAgentParams) (*agent.Agent, error) {
	s.log.Infow("Creating new agent",
		"identifier", params.Identifier,
		"name", params.Name,
	)

	// Validate
	if params.Identifier == "" {
		return nil, errors.New("identifier is required")
	}
	if params.Name == "" {
		return nil, errors.New("name is required")
	}

	// Create agent entity
	a := &agent.Agent{
		Identifier:     params.Identifier,
		Name:           params.Name,
		Description:    params.Description,
		Category:       params.Category,
		SystemPrompt:   params.SystemPrompt,
		Instructions:   params.Instructions,
		ModelProvider:  params.ModelProvider,
		ModelName:      params.ModelName,
		Temperature:    params.Temperature,
		MaxTokens:      params.MaxTokens,
		MaxCostPerRun:  params.MaxCostPerRun,
		TimeoutSeconds: params.TimeoutSeconds,
		IsActive:       params.IsActive,
	}

	// Set available tools
	if err := a.SetAvailableTools(params.AvailableTools); err != nil {
		return nil, errors.Wrap(err, "failed to set available tools")
	}

	// Create via domain service
	if err := s.domainService.Create(ctx, a); err != nil {
		return nil, errors.Wrap(err, "failed to create agent")
	}

	s.log.Infow("Agent created successfully",
		"agent_id", a.ID,
		"identifier", a.Identifier,
	)

	return a, nil
}

// UpdateAgentParams contains parameters for updating an agent
type UpdateAgentParams struct {
	Name           *string
	Description    *string
	Category       *string
	SystemPrompt   *string
	Instructions   *string
	ModelProvider  *ai.ProviderName
	ModelName      *ai.ProviderModelName
	Temperature    *float64
	MaxTokens      *int
	AvailableTools []string // nil means no update, empty array clears tools
	MaxCostPerRun  *float64
	TimeoutSeconds *int
}

// UpdateAgent updates agent configuration (admin function)
func (s *Service) UpdateAgent(ctx context.Context, id int, params UpdateAgentParams) (*agent.Agent, error) {
	s.log.Infow("Updating agent", "agent_id", id)

	// Get existing agent via domain service
	a, err := s.domainService.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get agent")
	}

	// Apply updates (only non-nil fields)
	if params.Name != nil {
		a.Name = *params.Name
	}
	if params.Description != nil {
		a.Description = *params.Description
	}
	if params.Category != nil {
		a.Category = *params.Category
	}
	if params.SystemPrompt != nil {
		a.SystemPrompt = *params.SystemPrompt
	}
	if params.Instructions != nil {
		a.Instructions = *params.Instructions
	}
	if params.ModelProvider != nil {
		a.ModelProvider = *params.ModelProvider
	}
	if params.ModelName != nil {
		a.ModelName = *params.ModelName
	}
	if params.Temperature != nil {
		a.Temperature = *params.Temperature
	}
	if params.MaxTokens != nil {
		a.MaxTokens = *params.MaxTokens
	}
	if params.AvailableTools != nil {
		if err := a.SetAvailableTools(params.AvailableTools); err != nil {
			return nil, errors.Wrap(err, "failed to set available tools")
		}
	}
	if params.MaxCostPerRun != nil {
		a.MaxCostPerRun = *params.MaxCostPerRun
	}
	if params.TimeoutSeconds != nil {
		a.TimeoutSeconds = *params.TimeoutSeconds
	}

	// Update via domain service
	if err := s.domainService.Update(ctx, a); err != nil {
		return nil, errors.Wrap(err, "failed to update agent")
	}

	s.log.Infow("Agent updated successfully",
		"agent_id", id,
		"identifier", a.Identifier,
		"version", a.Version,
	)

	return a, nil
}

// DeleteAgent deletes an agent (admin function, hard delete)
func (s *Service) DeleteAgent(ctx context.Context, id int) error {
	s.log.Warnw("Deleting agent", "agent_id", id)

	// Get agent to check if it exists
	a, err := s.domainService.GetByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "failed to get agent")
	}

	// Use domain service to delete
	if err := s.domainService.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "failed to delete agent")
	}

	s.log.Infow("Agent deleted successfully",
		"agent_id", id,
		"identifier", a.Identifier,
	)

	return nil
}

// BatchDeleteAgents deletes multiple agents in batch (admin function)
// Returns the number of successfully deleted agents
func (s *Service) BatchDeleteAgents(ctx context.Context, ids []int) (int, error) {
	s.log.Infow("Batch deleting agents",
		"count", len(ids),
	)

	if len(ids) == 0 {
		return 0, errors.ErrInvalidInput
	}

	deleted := 0
	var lastErr error

	for _, id := range ids {
		if err := s.DeleteAgent(ctx, id); err != nil {
			s.log.Warnw("Failed to delete agent in batch",
				"agent_id", id,
				"error", err,
			)
			lastErr = err
			continue
		}
		deleted++
	}

	s.log.Infow("Batch delete completed",
		"deleted", deleted,
		"total", len(ids),
	)

	// Return error only if all deletions failed
	if deleted == 0 && lastErr != nil {
		return 0, lastErr
	}

	return deleted, nil
}

// SetActive activates or deactivates an agent
func (s *Service) SetActive(ctx context.Context, id int, isActive bool) (*agent.Agent, error) {
	s.log.Infow("Setting agent active status",
		"agent_id", id,
		"is_active", isActive,
	)

	// Get agent via domain service
	a, err := s.domainService.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get agent")
	}

	a.IsActive = isActive

	// Update via domain service
	if err := s.domainService.Update(ctx, a); err != nil {
		return nil, errors.Wrap(err, "failed to update agent active status")
	}

	s.log.Infow("Agent active status updated",
		"agent_id", id,
		"identifier", a.Identifier,
		"is_active", isActive,
	)

	return a, nil
}

// UpdatePromptByID updates agent's system prompt by ID
func (s *Service) UpdatePromptByID(ctx context.Context, id int, newPrompt string) (*agent.Agent, error) {
	s.log.Infow("Updating agent prompt", "agent_id", id)

	// Get agent via domain service
	a, err := s.domainService.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get agent")
	}

	a.SystemPrompt = newPrompt

	// Update via domain service
	if err := s.domainService.Update(ctx, a); err != nil {
		return nil, errors.Wrap(err, "failed to update agent prompt")
	}

	s.log.Infow("Agent prompt updated",
		"agent_id", id,
		"identifier", a.Identifier,
		"version", a.Version,
	)

	return a, nil
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
