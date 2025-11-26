package exchange_account

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"prometheus/pkg/logger"
)

// Service handles business logic for exchange accounts.
type Service struct {
	repo Repository
	log  *logger.Logger
}

// NewService constructs the exchange account service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, log: logger.Get()}
}

// Create registers a new exchange account for a user.
func (s *Service) Create(ctx context.Context, account *ExchangeAccount) error {
	if account == nil {
		return fmt.Errorf("create exchange account: account is nil")
	}
	if account.ID == uuid.Nil {
		account.ID = uuid.New()
	}
	if account.UserID == uuid.Nil {
		return fmt.Errorf("create exchange account: user id is required")
	}
	if !account.Exchange.Valid() {
		return fmt.Errorf("create exchange account: exchange is invalid")
	}

	if err := s.repo.Create(ctx, account); err != nil {
		return fmt.Errorf("create exchange account: %w", err)
	}
	return nil
}

// GetByID fetches an exchange account by identifier.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*ExchangeAccount, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("get exchange account: id is required")
	}
	account, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get exchange account: %w", err)
	}
	return account, nil
}

// ListByUser returns all exchange accounts for a user.
func (s *Service) ListByUser(ctx context.Context, userID uuid.UUID) ([]*ExchangeAccount, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("list exchange accounts: user id is required")
	}
	accounts, err := s.repo.GetByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list exchange accounts: %w", err)
	}
	return accounts, nil
}

// ListActiveByUser returns active accounts only.
func (s *Service) ListActiveByUser(ctx context.Context, userID uuid.UUID) ([]*ExchangeAccount, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("list active exchange accounts: user id is required")
	}
	accounts, err := s.repo.GetActiveByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list active exchange accounts: %w", err)
	}
	return accounts, nil
}

// Update persists account changes.
func (s *Service) Update(ctx context.Context, account *ExchangeAccount) error {
	if account == nil {
		return fmt.Errorf("update exchange account: account is nil")
	}
	if account.ID == uuid.Nil {
		return fmt.Errorf("update exchange account: id is required")
	}
	if err := s.repo.Update(ctx, account); err != nil {
		return fmt.Errorf("update exchange account: %w", err)
	}
	return nil
}

// UpdateLastSync updates the last synchronization time.
func (s *Service) UpdateLastSync(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("update last sync: id is required")
	}
	if err := s.repo.UpdateLastSync(ctx, id); err != nil {
		return fmt.Errorf("update last sync: %w", err)
	}
	return nil
}

// Delete removes an exchange account.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("delete exchange account: id is required")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete exchange account: %w", err)
	}
	return nil
}
