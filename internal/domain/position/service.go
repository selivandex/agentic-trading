package position

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

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
		return fmt.Errorf("open position: position is nil")
	}
	if position.ID == uuid.Nil {
		position.ID = uuid.New()
	}
	if position.UserID == uuid.Nil {
		return fmt.Errorf("open position: user id is required")
	}
	if !position.Side.Valid() {
		return fmt.Errorf("open position: invalid side")
	}
	if position.Size.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("open position: size must be positive")
	}
	position.Status = PositionOpen
	position.OpenedAt = time.Now().UTC()
	position.UpdatedAt = position.OpenedAt

	if err := s.repo.Create(ctx, position); err != nil {
		return fmt.Errorf("open position: %w", err)
	}
	return nil
}

// GetByID retrieves a position by identifier.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Position, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("get position: id is required")
	}
	pos, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get position: %w", err)
	}
	return pos, nil
}

// GetOpenByUser lists open positions for a user.
func (s *Service) GetOpenByUser(ctx context.Context, userID uuid.UUID) ([]*Position, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("get open positions: user id is required")
	}
	positions, err := s.repo.GetOpenByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get open positions: %w", err)
	}
	return positions, nil
}

// UpdatePnL updates current price and unrealized PnL fields.
func (s *Service) UpdatePnL(ctx context.Context, id uuid.UUID, currentPrice, unrealizedPnL, unrealizedPnLPct decimal.Decimal) error {
	if id == uuid.Nil {
		return fmt.Errorf("update pnl: id is required")
	}
	if err := s.repo.UpdatePnL(ctx, id, currentPrice, unrealizedPnL, unrealizedPnLPct); err != nil {
		return fmt.Errorf("update pnl: %w", err)
	}
	return nil
}

// Close marks a position as closed with realized PnL.
func (s *Service) Close(ctx context.Context, id uuid.UUID, exitPrice, realizedPnL decimal.Decimal) error {
	if id == uuid.Nil {
		return fmt.Errorf("close position: id is required")
	}
	if err := s.repo.Close(ctx, id, exitPrice, realizedPnL); err != nil {
		return fmt.Errorf("close position: %w", err)
	}
	return nil
}

// Update persists general position changes.
func (s *Service) Update(ctx context.Context, position *Position) error {
	if position == nil {
		return fmt.Errorf("update position: position is nil")
	}
	if position.ID == uuid.Nil {
		return fmt.Errorf("update position: id is required")
	}
	if err := s.repo.Update(ctx, position); err != nil {
		return fmt.Errorf("update position: %w", err)
	}
	return nil
}

// Delete removes a position.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("delete position: id is required")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete position: %w", err)
	}
	return nil
}
