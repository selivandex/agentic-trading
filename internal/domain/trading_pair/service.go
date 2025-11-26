package trading_pair

import (
	"context"
	"fmt"

	"github.com/google/uuid"

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
		return fmt.Errorf("create trading pair: pair is nil")
	}
	if pair.ID == uuid.Nil {
		pair.ID = uuid.New()
	}
	if pair.UserID == uuid.Nil {
		return fmt.Errorf("create trading pair: user id is required")
	}
	if pair.Symbol == "" {
		return fmt.Errorf("create trading pair: symbol is required")
	}

	if err := s.repo.Create(ctx, pair); err != nil {
		return fmt.Errorf("create trading pair: %w", err)
	}
	return nil
}

// GetByID returns a trading pair by identifier.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*TradingPair, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("get trading pair: id is required")
	}
	pair, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get trading pair: %w", err)
	}
	return pair, nil
}

// ListByUser returns pairs for a user.
func (s *Service) ListByUser(ctx context.Context, userID uuid.UUID) ([]*TradingPair, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("list trading pairs: user id is required")
	}
	pairs, err := s.repo.GetByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list trading pairs: %w", err)
	}
	return pairs, nil
}

// ListActiveByUser returns active pairs.
func (s *Service) ListActiveByUser(ctx context.Context, userID uuid.UUID) ([]*TradingPair, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("list active trading pairs: user id is required")
	}
	pairs, err := s.repo.GetActiveByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list active trading pairs: %w", err)
	}
	return pairs, nil
}

// ListAllActive returns all active pairs across users.
func (s *Service) ListAllActive(ctx context.Context) ([]*TradingPair, error) {
	pairs, err := s.repo.FindAllActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all active trading pairs: %w", err)
	}
	return pairs, nil
}

// Update persists changes to a trading pair.
func (s *Service) Update(ctx context.Context, pair *TradingPair) error {
	if pair == nil {
		return fmt.Errorf("update trading pair: pair is nil")
	}
	if pair.ID == uuid.Nil {
		return fmt.Errorf("update trading pair: id is required")
	}
	if err := s.repo.Update(ctx, pair); err != nil {
		return fmt.Errorf("update trading pair: %w", err)
	}
	return nil
}

// Pause marks a trading pair as paused with a reason.
func (s *Service) Pause(ctx context.Context, id uuid.UUID, reason string) error {
	if id == uuid.Nil {
		return fmt.Errorf("pause trading pair: id is required")
	}
	if err := s.repo.Pause(ctx, id, reason); err != nil {
		return fmt.Errorf("pause trading pair: %w", err)
	}
	return nil
}

// Resume resumes trading on a pair.
func (s *Service) Resume(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("resume trading pair: id is required")
	}
	if err := s.repo.Resume(ctx, id); err != nil {
		return fmt.Errorf("resume trading pair: %w", err)
	}
	return nil
}

// Delete removes a trading pair configuration.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("delete trading pair: id is required")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete trading pair: %w", err)
	}
	return nil
}
