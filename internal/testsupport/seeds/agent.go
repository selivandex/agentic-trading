package seeds

import (
	"context"

	"prometheus/internal/adapters/ai"
	"prometheus/internal/domain/agent"
)

// AgentBuilder builds agent entities for testing
type AgentBuilder struct {
	db  DBTX
	ctx context.Context
	a   *agent.Agent
}

// NewAgentBuilder creates a new agent builder
func NewAgentBuilder(db DBTX, ctx context.Context) *AgentBuilder {
	return &AgentBuilder{
		db:  db,
		ctx: ctx,
		a: &agent.Agent{
			Identifier:     "test_agent",
			Name:           "Test Agent",
			Description:    "Test agent for testing",
			Category:       agent.CategoryExpert,
			SystemPrompt:   "You are a test agent",
			Instructions:   "Test instructions",
			ModelProvider:  ai.ProviderNameDeepSeek,
			ModelName:      ai.ModelDeepSeekReasoner,
			Temperature:    1.0,
			MaxTokens:      4000,
			MaxCostPerRun:  0.10,
			TimeoutSeconds: 60,
			IsActive:       true,
		},
	}
}

// WithIdentifier sets the identifier
func (b *AgentBuilder) WithIdentifier(identifier string) *AgentBuilder {
	b.a.Identifier = identifier
	return b
}

// WithName sets the name
func (b *AgentBuilder) WithName(name string) *AgentBuilder {
	b.a.Name = name
	return b
}

// WithDescription sets the description
func (b *AgentBuilder) WithDescription(description string) *AgentBuilder {
	b.a.Description = description
	return b
}

// WithCategory sets the category
func (b *AgentBuilder) WithCategory(category string) *AgentBuilder {
	b.a.Category = category
	return b
}

// WithSystemPrompt sets the system prompt
func (b *AgentBuilder) WithSystemPrompt(prompt string) *AgentBuilder {
	b.a.SystemPrompt = prompt
	return b
}

// WithInstructions sets the instructions
func (b *AgentBuilder) WithInstructions(instructions string) *AgentBuilder {
	b.a.Instructions = instructions
	return b
}

// WithModelProvider sets the model provider
func (b *AgentBuilder) WithModelProvider(provider ai.ProviderName) *AgentBuilder {
	b.a.ModelProvider = provider
	return b
}

// WithModelName sets the model name
func (b *AgentBuilder) WithModelName(model ai.ProviderModelName) *AgentBuilder {
	b.a.ModelName = model
	return b
}

// WithTemperature sets the temperature
func (b *AgentBuilder) WithTemperature(temp float64) *AgentBuilder {
	b.a.Temperature = temp
	return b
}

// WithMaxTokens sets the max tokens
func (b *AgentBuilder) WithMaxTokens(tokens int) *AgentBuilder {
	b.a.MaxTokens = tokens
	return b
}

// WithAvailableTools sets the available tools
func (b *AgentBuilder) WithAvailableTools(tools []string) *AgentBuilder {
	_ = b.a.SetAvailableTools(tools)
	return b
}

// WithMaxCostPerRun sets the max cost per run
func (b *AgentBuilder) WithMaxCostPerRun(cost float64) *AgentBuilder {
	b.a.MaxCostPerRun = cost
	return b
}

// WithTimeoutSeconds sets the timeout
func (b *AgentBuilder) WithTimeoutSeconds(timeout int) *AgentBuilder {
	b.a.TimeoutSeconds = timeout
	return b
}

// WithIsActive sets the active status
func (b *AgentBuilder) WithIsActive(active bool) *AgentBuilder {
	b.a.IsActive = active
	return b
}

// WithExpert sets category to expert
func (b *AgentBuilder) WithExpert() *AgentBuilder {
	b.a.Category = agent.CategoryExpert
	return b
}

// WithCoordinator sets category to coordinator
func (b *AgentBuilder) WithCoordinator() *AgentBuilder {
	b.a.Category = agent.CategoryCoordinator
	return b
}

// WithSpecialist sets category to specialist
func (b *AgentBuilder) WithSpecialist() *AgentBuilder {
	b.a.Category = agent.CategorySpecialist
	return b
}

// WithDeepSeek sets DeepSeek provider and model
func (b *AgentBuilder) WithDeepSeek(model ai.ProviderModelName) *AgentBuilder {
	b.a.ModelProvider = ai.ProviderNameDeepSeek
	b.a.ModelName = model
	return b
}

// WithClaude sets Claude provider and model
func (b *AgentBuilder) WithClaude(model ai.ProviderModelName) *AgentBuilder {
	b.a.ModelProvider = ai.ProviderNameAnthropic
	b.a.ModelName = model
	return b
}

// WithTools sets available tools
func (b *AgentBuilder) WithTools(tools []string) *AgentBuilder {
	_ = b.a.SetAvailableTools(tools)
	return b
}

// WithTimeout sets timeout in seconds
func (b *AgentBuilder) WithTimeout(seconds int) *AgentBuilder {
	b.a.TimeoutSeconds = seconds
	return b
}

// WithActive sets active status
func (b *AgentBuilder) WithActive(active bool) *AgentBuilder {
	b.a.IsActive = active
	return b
}

// MustInsert inserts the agent into the database
func (b *AgentBuilder) MustInsert() *agent.Agent {
	const query = `
		INSERT INTO agents (
			identifier, name, description, category,
			system_prompt, instructions,
			model_provider, model_name, temperature, max_tokens,
			available_tools, max_cost_per_run, timeout_seconds, is_active
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		) RETURNING id, version, created_at, updated_at
	`

	err := b.db.QueryRowContext(
		b.ctx,
		query,
		b.a.Identifier,
		b.a.Name,
		b.a.Description,
		b.a.Category,
		b.a.SystemPrompt,
		b.a.Instructions,
		b.a.ModelProvider,
		b.a.ModelName,
		b.a.Temperature,
		b.a.MaxTokens,
		b.a.AvailableTools,
		b.a.MaxCostPerRun,
		b.a.TimeoutSeconds,
		b.a.IsActive,
	).Scan(&b.a.ID, &b.a.Version, &b.a.CreatedAt, &b.a.UpdatedAt)

	if err != nil {
		panic("failed to insert agent: " + err.Error())
	}

	return b.a
}

// Insert is an alias for MustInsert for consistency with seed files
func (b *AgentBuilder) Insert() (*agent.Agent, error) {
	const query = `
		INSERT INTO agents (
			identifier, name, description, category,
			system_prompt, instructions,
			model_provider, model_name, temperature, max_tokens,
			available_tools, max_cost_per_run, timeout_seconds, is_active
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		) RETURNING id, version, created_at, updated_at
	`

	err := b.db.QueryRowContext(
		b.ctx,
		query,
		b.a.Identifier,
		b.a.Name,
		b.a.Description,
		b.a.Category,
		b.a.SystemPrompt,
		b.a.Instructions,
		b.a.ModelProvider,
		b.a.ModelName,
		b.a.Temperature,
		b.a.MaxTokens,
		b.a.AvailableTools,
		b.a.MaxCostPerRun,
		b.a.TimeoutSeconds,
		b.a.IsActive,
	).Scan(&b.a.ID, &b.a.Version, &b.a.CreatedAt, &b.a.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return b.a, nil
}
