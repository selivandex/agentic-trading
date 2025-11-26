package exchange_account

import (
	"context"

	"github.com/google/uuid"

	"prometheus/pkg/errors"
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
		return errors.ErrInvalidInput
	}
	if account.ID == uuid.Nil {
		account.ID = uuid.New()
	}
	if account.UserID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if !account.Exchange.Valid() {
		return errors.Wrapf(errors.ErrInternal, "create exchange account: exchange is invalid")
	}

	if err := s.repo.Create(ctx, account); err != nil {
		return errors.Wrap(err, "create exchange account")
	}
	return nil
}

// GetByID fetches an exchange account by identifier.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*ExchangeAccount, error) {
	if id == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}
	account, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get exchange account")
	}
	return account, nil
}

// ListByUser returns all exchange accounts for a user.
func (s *Service) ListByUser(ctx context.Context, userID uuid.UUID) ([]*ExchangeAccount, error) {
	if userID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}
	accounts, err := s.repo.GetByUser(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "list exchange accounts")
	}
	return accounts, nil
}

// ListActiveByUser returns active accounts only.
func (s *Service) ListActiveByUser(ctx context.Context, userID uuid.UUID) ([]*ExchangeAccount, error) {
	if userID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}
	accounts, err := s.repo.GetActiveByUser(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "list active exchange accounts")
	}
	return accounts, nil
}

// Update persists account changes.
func (s *Service) Update(ctx context.Context, account *ExchangeAccount) error {
	if account == nil {
		return errors.ErrInvalidInput
	}
	if account.ID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if err := s.repo.Update(ctx, account); err != nil {
		return errors.Wrap(err, "update exchange account")
	}
	return nil
}

// UpdateLastSync updates the last synchronization time.
func (s *Service) UpdateLastSync(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if err := s.repo.UpdateLastSync(ctx, id); err != nil {
		return errors.Wrap(err, "update last sync")
	}
	return nil
}

// Delete removes an exchange account.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "delete exchange account")
	}
	return nil
}
