package trading_pair

import (
	"context"

	"github.com/google/uuid"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service coordinates trading pair operations.
type Service struct {
	repo Repository
	log  *logger.Logger
}

// NewService constructs a trading pair service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, log: logger.Get()}
}

// Create registers a new trading pair configuration.
func (s *Service) Create(ctx context.Context, pair *TradingPair) error {
	if pair == nil {
		return errors.ErrInvalidInput
	}
	if pair.ID == uuid.Nil {
		pair.ID = uuid.New()
	}
	if pair.UserID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if pair.Symbol == "" {
		return errors.ErrInvalidInput
	}

	if err := s.repo.Create(ctx, pair); err != nil {
		return errors.Wrap(err, "create trading pair")
	}
	return nil
}

// GetByID returns a trading pair by identifier.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*TradingPair, error) {
	if id == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}
	pair, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get trading pair")
	}
	return pair, nil
}

// ListByUser returns pairs for a user.
func (s *Service) ListByUser(ctx context.Context, userID uuid.UUID) ([]*TradingPair, error) {
	if userID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}
	pairs, err := s.repo.GetByUser(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "list trading pairs")
	}
	return pairs, nil
}

// ListActiveByUser returns active pairs.
func (s *Service) ListActiveByUser(ctx context.Context, userID uuid.UUID) ([]*TradingPair, error) {
	if userID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}
	pairs, err := s.repo.GetActiveByUser(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "list active trading pairs")
	}
	return pairs, nil
}

// ListAllActive returns all active pairs across users.
func (s *Service) ListAllActive(ctx context.Context) ([]*TradingPair, error) {
	pairs, err := s.repo.FindAllActive(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "list all active trading pairs")
	}
	return pairs, nil
}

// Update persists changes to a trading pair.
func (s *Service) Update(ctx context.Context, pair *TradingPair) error {
	if pair == nil {
		return errors.ErrInvalidInput
	}
	if pair.ID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if err := s.repo.Update(ctx, pair); err != nil {
		return errors.Wrap(err, "update trading pair")
	}
	return nil
}

// Pause marks a trading pair as paused with a reason.
func (s *Service) Pause(ctx context.Context, id uuid.UUID, reason string) error {
	if id == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if err := s.repo.Pause(ctx, id, reason); err != nil {
		return errors.Wrap(err, "pause trading pair")
	}
	return nil
}

// Resume resumes trading on a pair.
func (s *Service) Resume(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if err := s.repo.Resume(ctx, id); err != nil {
		return errors.Wrap(err, "resume trading pair")
	}
	return nil
}

// Delete removes a trading pair configuration.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "delete trading pair")
	}
	return nil
}
