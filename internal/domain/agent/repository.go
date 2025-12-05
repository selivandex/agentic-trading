package agent

import (
	"context"
)

// Repository defines the interface for agent data access
type Repository interface {
	// Create creates a new agent
	Create(ctx context.Context, agent *Agent) error

	// GetByID retrieves agent by primary key
	GetByID(ctx context.Context, id int) (*Agent, error)

	// GetByIdentifier retrieves agent by unique identifier
	GetByIdentifier(ctx context.Context, identifier string) (*Agent, error)

	// FindOrCreate gets existing agent or creates new one
	// Returns (agent, wasCreated, error)
	FindOrCreate(ctx context.Context, agent *Agent) (*Agent, bool, error)

	// Update updates agent configuration (prompts, model, etc)
	Update(ctx context.Context, agent *Agent) error

	// List retrieves all agents
	List(ctx context.Context) ([]*Agent, error)

	// ListActive retrieves only active agents
	ListActive(ctx context.Context) ([]*Agent, error)

	// ListByCategory retrieves agents by category (expert, coordinator, etc)
	ListByCategory(ctx context.Context, category string) ([]*Agent, error)
}
