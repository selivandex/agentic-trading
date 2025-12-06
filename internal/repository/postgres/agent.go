package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"prometheus/internal/domain/agent"
	"prometheus/pkg/errors"
)

// AgentRepository implements agent.Repository
type AgentRepository struct {
	db DBTX
}

// NewAgentRepository creates a new agent repository
func NewAgentRepository(db DBTX) *AgentRepository {
	return &AgentRepository{db: db}
}

// Create creates a new agent
func (r *AgentRepository) Create(ctx context.Context, a *agent.Agent) error {
	query := `
		INSERT INTO agents (
			identifier, name, description, category,
			system_prompt, instructions,
			model_provider, model_name, temperature, max_tokens,
			available_tools, max_cost_per_run, timeout_seconds, is_active
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
		RETURNING id, version, created_at, updated_at
	`

	return r.db.QueryRowContext(ctx, query,
		a.Identifier, a.Name, a.Description, a.Category,
		a.SystemPrompt, a.Instructions,
		a.ModelProvider, a.ModelName, a.Temperature, a.MaxTokens,
		a.AvailableTools, a.MaxCostPerRun, a.TimeoutSeconds, a.IsActive,
	).Scan(&a.ID, &a.Version, &a.CreatedAt, &a.UpdatedAt)
}

// GetByID retrieves agent by ID
func (r *AgentRepository) GetByID(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {
	query := `
		SELECT id, identifier, name, description, category,
		       system_prompt, instructions,
		       model_provider, model_name, temperature, max_tokens,
		       available_tools, max_cost_per_run, timeout_seconds,
		       is_active, version, created_at, updated_at
		FROM agents
		WHERE id = $1
	`

	a := &agent.Agent{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&a.ID, &a.Identifier, &a.Name, &a.Description, &a.Category,
		&a.SystemPrompt, &a.Instructions,
		&a.ModelProvider, &a.ModelName, &a.Temperature, &a.MaxTokens,
		&a.AvailableTools, &a.MaxCostPerRun, &a.TimeoutSeconds,
		&a.IsActive, &a.Version, &a.CreatedAt, &a.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.ErrNotFound
	}
	if err != nil {
		return nil, errors.Wrap(err, "get agent by id")
	}
	return a, nil
}

// GetByIdentifier retrieves agent by unique identifier
func (r *AgentRepository) GetByIdentifier(ctx context.Context, identifier string) (*agent.Agent, error) {
	query := `
		SELECT id, identifier, name, description, category,
		       system_prompt, instructions,
		       model_provider, model_name, temperature, max_tokens,
		       available_tools, max_cost_per_run, timeout_seconds,
		       is_active, version, created_at, updated_at
		FROM agents
		WHERE identifier = $1
	`

	a := &agent.Agent{}
	err := r.db.QueryRowContext(ctx, query, identifier).Scan(
		&a.ID, &a.Identifier, &a.Name, &a.Description, &a.Category,
		&a.SystemPrompt, &a.Instructions,
		&a.ModelProvider, &a.ModelName, &a.Temperature, &a.MaxTokens,
		&a.AvailableTools, &a.MaxCostPerRun, &a.TimeoutSeconds,
		&a.IsActive, &a.Version, &a.CreatedAt, &a.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.ErrNotFound
	}
	if err != nil {
		return nil, errors.Wrap(err, "get agent by identifier")
	}
	return a, nil
}

// FindOrCreate gets existing agent or creates new one
func (r *AgentRepository) FindOrCreate(ctx context.Context, a *agent.Agent) (*agent.Agent, bool, error) {
	// Try to find existing
	existing, err := r.GetByIdentifier(ctx, a.Identifier)
	if err == nil {
		// Found - return existing
		return existing, false, nil
	}
	if err != errors.ErrNotFound {
		// Unexpected error
		return nil, false, errors.Wrap(err, "find agent")
	}

	// Not found - create new
	if err := r.Create(ctx, a); err != nil {
		return nil, false, errors.Wrap(err, "create agent")
	}

	return a, true, nil
}

// Update updates agent configuration
func (r *AgentRepository) Update(ctx context.Context, a *agent.Agent) error {
	query := `
		UPDATE agents SET
			name = $1,
			description = $2,
			category = $3,
			system_prompt = $4,
			instructions = $5,
			model_provider = $6,
			model_name = $7,
			temperature = $8,
			max_tokens = $9,
			available_tools = $10,
			max_cost_per_run = $11,
			timeout_seconds = $12,
			is_active = $13
		WHERE id = $14
		RETURNING version, updated_at
	`

	return r.db.QueryRowContext(ctx, query,
		a.Name, a.Description, a.Category,
		a.SystemPrompt, a.Instructions,
		a.ModelProvider, a.ModelName, a.Temperature, a.MaxTokens,
		a.AvailableTools, a.MaxCostPerRun, a.TimeoutSeconds, a.IsActive,
		a.ID,
	).Scan(&a.Version, &a.UpdatedAt)
}

// List retrieves all agents
func (r *AgentRepository) List(ctx context.Context) ([]*agent.Agent, error) {
	query := `
		SELECT id, identifier, name, description, category,
		       system_prompt, instructions,
		       model_provider, model_name, temperature, max_tokens,
		       available_tools, max_cost_per_run, timeout_seconds,
		       is_active, version, created_at, updated_at
		FROM agents
		ORDER BY category, name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "list agents")
	}
	defer rows.Close()

	var agents []*agent.Agent
	for rows.Next() {
		a := &agent.Agent{}
		if err := rows.Scan(
			&a.ID, &a.Identifier, &a.Name, &a.Description, &a.Category,
			&a.SystemPrompt, &a.Instructions,
			&a.ModelProvider, &a.ModelName, &a.Temperature, &a.MaxTokens,
			&a.AvailableTools, &a.MaxCostPerRun, &a.TimeoutSeconds,
			&a.IsActive, &a.Version, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, errors.Wrap(err, "scan agent")
		}
		agents = append(agents, a)
	}

	return agents, rows.Err()
}

// ListActive retrieves only active agents
func (r *AgentRepository) ListActive(ctx context.Context) ([]*agent.Agent, error) {
	query := `
		SELECT id, identifier, name, description, category,
		       system_prompt, instructions,
		       model_provider, model_name, temperature, max_tokens,
		       available_tools, max_cost_per_run, timeout_seconds,
		       is_active, version, created_at, updated_at
		FROM agents
		WHERE is_active = true
		ORDER BY category, name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "list active agents")
	}
	defer rows.Close()

	var agents []*agent.Agent
	for rows.Next() {
		a := &agent.Agent{}
		if err := rows.Scan(
			&a.ID, &a.Identifier, &a.Name, &a.Description, &a.Category,
			&a.SystemPrompt, &a.Instructions,
			&a.ModelProvider, &a.ModelName, &a.Temperature, &a.MaxTokens,
			&a.AvailableTools, &a.MaxCostPerRun, &a.TimeoutSeconds,
			&a.IsActive, &a.Version, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, errors.Wrap(err, "scan agent")
		}
		agents = append(agents, a)
	}

	return agents, rows.Err()
}

// ListByCategory retrieves agents by category
func (r *AgentRepository) ListByCategory(ctx context.Context, category string) ([]*agent.Agent, error) {
	query := `
		SELECT id, identifier, name, description, category,
		       system_prompt, instructions,
		       model_provider, model_name, temperature, max_tokens,
		       available_tools, max_cost_per_run, timeout_seconds,
		       is_active, version, created_at, updated_at
		FROM agents
		WHERE category = $1 AND is_active = true
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, category)
	if err != nil {
		return nil, errors.Wrap(err, "list agents by category")
	}
	defer rows.Close()

	var agents []*agent.Agent
	for rows.Next() {
		a := &agent.Agent{}
		if err := rows.Scan(
			&a.ID, &a.Identifier, &a.Name, &a.Description, &a.Category,
			&a.SystemPrompt, &a.Instructions,
			&a.ModelProvider, &a.ModelName, &a.Temperature, &a.MaxTokens,
			&a.AvailableTools, &a.MaxCostPerRun, &a.TimeoutSeconds,
			&a.IsActive, &a.Version, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, errors.Wrap(err, "scan agent")
		}
		agents = append(agents, a)
	}

	return agents, rows.Err()
}

// Delete deletes an agent by ID
func (r *AgentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM agents WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.Wrap(err, "delete agent")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "get rows affected")
	}

	if rows == 0 {
		return errors.ErrNotFound
	}

	return nil
}
