package seeds

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"prometheus/internal/adapters/ai"
	"prometheus/internal/domain/agent"
)

// AgentBuilder provides a fluent API for creating Agent entities
type AgentBuilder struct {
	db     DBTX
	ctx    context.Context
	entity *agent.Agent
}

// NewAgentBuilder creates a new AgentBuilder with sensible defaults
func NewAgentBuilder(db DBTX, ctx context.Context) *AgentBuilder {
	now := time.Now()
	tools := []string{"get_market_data", "calculate_indicators"}
	toolsJSON, _ := json.Marshal(tools)

	return &AgentBuilder{
		db:  db,
		ctx: ctx,
		entity: &agent.Agent{
			ID:             0, // Auto-increment
			Identifier:     "test_agent",
			Name:           "Test Agent",
			Description:    "Test agent description",
			Category:       agent.CategorySpecialist,
			SystemPrompt:   "You are a test agent.",
			Instructions:   "Follow test instructions.",
			ModelProvider:  ai.ProviderNameAnthropic,
			ModelName:      ai.ModelClaude45Sonnet,
			Temperature:    0.7,
			MaxTokens:      4000,
			AvailableTools: toolsJSON,
			MaxCostPerRun:  1.0,
			TimeoutSeconds: 300,
			IsActive:       true,
			Version:        1,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}
}

// WithIdentifier sets the agent identifier
func (b *AgentBuilder) WithIdentifier(identifier string) *AgentBuilder {
	b.entity.Identifier = identifier
	return b
}

// WithName sets the agent name
func (b *AgentBuilder) WithName(name string) *AgentBuilder {
	b.entity.Name = name
	return b
}

// WithDescription sets the description
func (b *AgentBuilder) WithDescription(desc string) *AgentBuilder {
	b.entity.Description = desc
	return b
}

// WithCategory sets the category
func (b *AgentBuilder) WithCategory(category string) *AgentBuilder {
	b.entity.Category = category
	return b
}

// WithCoordinator sets category to coordinator
func (b *AgentBuilder) WithCoordinator() *AgentBuilder {
	b.entity.Category = agent.CategoryCoordinator
	return b
}

// WithExpert sets category to expert
func (b *AgentBuilder) WithExpert() *AgentBuilder {
	b.entity.Category = agent.CategoryExpert
	return b
}

// WithSpecialist sets category to specialist
func (b *AgentBuilder) WithSpecialist() *AgentBuilder {
	b.entity.Category = agent.CategorySpecialist
	return b
}

// WithSystemPrompt sets the system prompt
func (b *AgentBuilder) WithSystemPrompt(prompt string) *AgentBuilder {
	b.entity.SystemPrompt = prompt
	return b
}

// WithInstructions sets the instructions
func (b *AgentBuilder) WithInstructions(instructions string) *AgentBuilder {
	b.entity.Instructions = instructions
	return b
}

// WithModel sets the model provider and name
func (b *AgentBuilder) WithModel(provider ai.ProviderName, model ai.ProviderModelName) *AgentBuilder {
	b.entity.ModelProvider = provider
	b.entity.ModelName = model
	return b
}

// WithClaude sets Claude as provider
func (b *AgentBuilder) WithClaude(model ai.ProviderModelName) *AgentBuilder {
	b.entity.ModelProvider = ai.ProviderNameAnthropic
	b.entity.ModelName = model
	return b
}

// WithOpenAI sets OpenAI as provider
func (b *AgentBuilder) WithOpenAI(model ai.ProviderModelName) *AgentBuilder {
	b.entity.ModelProvider = ai.ProviderNameOpenAI
	b.entity.ModelName = model
	return b
}

// WithDeepSeek sets DeepSeek as provider
func (b *AgentBuilder) WithDeepSeek(model ai.ProviderModelName) *AgentBuilder {
	b.entity.ModelProvider = ai.ProviderNameDeepSeek
	b.entity.ModelName = model
	return b
}

// WithGemini sets Gemini as provider
func (b *AgentBuilder) WithGemini(model ai.ProviderModelName) *AgentBuilder {
	b.entity.ModelProvider = ai.ProviderNameGoogle
	b.entity.ModelName = model
	return b
}

// WithTemperature sets the temperature
func (b *AgentBuilder) WithTemperature(temp float64) *AgentBuilder {
	b.entity.Temperature = temp
	return b
}

// WithMaxTokens sets max tokens
func (b *AgentBuilder) WithMaxTokens(tokens int) *AgentBuilder {
	b.entity.MaxTokens = tokens
	return b
}

// WithTools sets available tools
func (b *AgentBuilder) WithTools(tools []string) *AgentBuilder {
	toolsJSON, _ := json.Marshal(tools)
	b.entity.AvailableTools = toolsJSON
	return b
}

// WithMaxCostPerRun sets max cost per run
func (b *AgentBuilder) WithMaxCostPerRun(cost float64) *AgentBuilder {
	b.entity.MaxCostPerRun = cost
	return b
}

// WithTimeout sets timeout in seconds
func (b *AgentBuilder) WithTimeout(seconds int) *AgentBuilder {
	b.entity.TimeoutSeconds = seconds
	return b
}

// WithActive sets active status
func (b *AgentBuilder) WithActive(active bool) *AgentBuilder {
	b.entity.IsActive = active
	return b
}

// Build returns the built entity without inserting to DB
func (b *AgentBuilder) Build() *agent.Agent {
	return b.entity
}

// Insert inserts the agent into the database and returns the entity
func (b *AgentBuilder) Insert() (*agent.Agent, error) {
	query := `
		INSERT INTO agents (
			identifier, name, description, category,
			system_prompt, instructions,
			model_provider, model_name, temperature, max_tokens,
			available_tools, max_cost_per_run, timeout_seconds,
			is_active, version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id
	`

	err := b.db.QueryRowContext(
		b.ctx,
		query,
		b.entity.Identifier,
		b.entity.Name,
		b.entity.Description,
		b.entity.Category,
		b.entity.SystemPrompt,
		b.entity.Instructions,
		b.entity.ModelProvider,
		b.entity.ModelName,
		b.entity.Temperature,
		b.entity.MaxTokens,
		b.entity.AvailableTools,
		b.entity.MaxCostPerRun,
		b.entity.TimeoutSeconds,
		b.entity.IsActive,
		b.entity.Version,
		b.entity.CreatedAt,
		b.entity.UpdatedAt,
	).Scan(&b.entity.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to insert agent: %w", err)
	}

	return b.entity, nil
}

// MustInsert inserts the agent and panics on error (useful for tests)
func (b *AgentBuilder) MustInsert() *agent.Agent {
	entity, err := b.Insert()
	if err != nil {
		panic(err)
	}
	return entity
}
