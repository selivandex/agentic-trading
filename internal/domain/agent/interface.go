package agent

import "context"

// DomainService defines interface for agent domain operations
// This allows application layer to depend on interface, not concrete implementation
type DomainService interface {
	Create(ctx context.Context, a *Agent) error
	GetByIdentifier(ctx context.Context, identifier string) (*Agent, error)
	GetByID(ctx context.Context, id int) (*Agent, error)
	Update(ctx context.Context, a *Agent) error
	FindOrCreate(ctx context.Context, a *Agent) (*Agent, bool, error)
	ListActive(ctx context.Context) ([]*Agent, error)
	ListByCategory(ctx context.Context, category string) ([]*Agent, error)
	List(ctx context.Context) ([]*Agent, error)
}

// Compile-time check that Service implements DomainService
var _ DomainService = (*Service)(nil)
