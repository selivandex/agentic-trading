package position

import (
	"context"

	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service manages position lifecycle operations.
type Service struct {
	repo Repository
	log  *logger.Logger
}

// NewService constructs a position service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, log: logger.Get()}
}

// Open records a new position.
func (s *Service) Open(ctx context.Context, position *Position) error {
	if position == nil {
		return errors.ErrInvalidInput
	}
	if position.ID == uuid.Nil {
		position.ID = uuid.New()
	}
	if position.UserID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if !position.Side.Valid() {
		return errors.ErrInvalidInput
	}
	if position.Size.LessThanOrEqual(decimal.Zero) {
		return errors.ErrInvalidInput
	}
	position.Status = PositionOpen
	position.OpenedAt = time.Now().UTC()
	position.UpdatedAt = position.OpenedAt

	if err := s.repo.Create(ctx, position); err != nil {
		return errors.Wrap(err, "open position")
	}
	return nil
}

// GetByID retrieves a position by identifier.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Position, error) {
	if id == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}
	pos, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get position")
	}
	return pos, nil
}

// GetOpenByUser lists open positions for a user.
func (s *Service) GetOpenByUser(ctx context.Context, userID uuid.UUID) ([]*Position, error) {
	if userID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}
	positions, err := s.repo.GetOpenByUser(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "get open positions")
	}
	return positions, nil
}

// UpdatePnL updates current price and unrealized PnL fields.
func (s *Service) UpdatePnL(ctx context.Context, id uuid.UUID, currentPrice, unrealizedPnL, unrealizedPnLPct decimal.Decimal) error {
	if id == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if err := s.repo.UpdatePnL(ctx, id, currentPrice, unrealizedPnL, unrealizedPnLPct); err != nil {
		return errors.Wrap(err, "update pnl")
	}
	return nil
}

// Close marks a position as closed with realized PnL.
func (s *Service) Close(ctx context.Context, id uuid.UUID, exitPrice, realizedPnL decimal.Decimal) error {
	if id == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if err := s.repo.Close(ctx, id, exitPrice, realizedPnL); err != nil {
		return errors.Wrap(err, "close position")
	}
	return nil
}

// Update persists general position changes.
func (s *Service) Update(ctx context.Context, position *Position) error {
	if position == nil {
		return errors.ErrInvalidInput
	}
	if position.ID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if err := s.repo.Update(ctx, position); err != nil {
		return errors.Wrap(err, "update position")
	}
	return nil
}

// Delete removes a position.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "delete position")
	}
	return nil
}
