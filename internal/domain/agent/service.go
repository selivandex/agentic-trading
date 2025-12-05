package agent

import (
	"context"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service handles agent business logic (domain layer)
type Service struct {
	repo Repository
	log  *logger.Logger
}

// NewService creates a new agent domain service
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		log:  logger.Get().With("component", "agent_domain_service"),
	}
}

// Create creates a new agent
func (s *Service) Create(ctx context.Context, a *Agent) error {
	if a.Identifier == "" {
		return errors.ErrInvalidInput
	}
	if a.Name == "" {
		return errors.ErrInvalidInput
	}
	if a.SystemPrompt == "" {
		return errors.ErrInvalidInput
	}

	return s.repo.Create(ctx, a)
}

// GetByIdentifier retrieves agent by identifier
func (s *Service) GetByIdentifier(ctx context.Context, identifier string) (*Agent, error) {
	if identifier == "" {
		return nil, errors.ErrInvalidInput
	}
	return s.repo.GetByIdentifier(ctx, identifier)
}

// GetByID retrieves agent by ID
func (s *Service) GetByID(ctx context.Context, id int) (*Agent, error) {
	if id <= 0 {
		return nil, errors.ErrInvalidInput
	}
	return s.repo.GetByID(ctx, id)
}

// Update updates agent configuration
func (s *Service) Update(ctx context.Context, a *Agent) error {
	if a == nil || a.ID <= 0 {
		return errors.ErrInvalidInput
	}
	if a.SystemPrompt == "" {
		return errors.ErrInvalidInput
	}

	return s.repo.Update(ctx, a)
}

// FindOrCreate gets existing agent or creates new one
func (s *Service) FindOrCreate(ctx context.Context, a *Agent) (*Agent, bool, error) {
	if a == nil || a.Identifier == "" {
		return nil, false, errors.ErrInvalidInput
	}

	return s.repo.FindOrCreate(ctx, a)
}

// ListActive retrieves all active agents
func (s *Service) ListActive(ctx context.Context) ([]*Agent, error) {
	return s.repo.ListActive(ctx)
}

// ListByCategory retrieves agents by category
func (s *Service) ListByCategory(ctx context.Context, category string) ([]*Agent, error) {
	if category == "" {
		return nil, errors.ErrInvalidInput
	}
	return s.repo.ListByCategory(ctx, category)
}

// List retrieves all agents
func (s *Service) List(ctx context.Context) ([]*Agent, error) {
	return s.repo.List(ctx)
}
