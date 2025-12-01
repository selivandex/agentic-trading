package strategy

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service manages strategy lifecycle operations (Domain Layer - Simple CRUD)
type Service struct {
	repo Repository
	log  *logger.Logger
}

// NewService constructs a strategy service
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		log:  logger.Get().With("component", "strategy_domain_service"),
	}
}

// Create creates a new strategy with validation
func (s *Service) Create(ctx context.Context, strategy *Strategy) error {
	if strategy == nil {
		return errors.ErrInvalidInput
	}
	if strategy.ID == uuid.Nil {
		strategy.ID = uuid.New()
	}
	if strategy.UserID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if strategy.AllocatedCapital.LessThanOrEqual(decimal.Zero) {
		return errors.New("allocated_capital must be positive")
	}
	if !strategy.RiskTolerance.Valid() {
		return errors.New("invalid risk_tolerance")
	}
	if !strategy.Status.Valid() {
		strategy.Status = StrategyActive
	}

	// Set timestamps
	now := time.Now()
	strategy.CreatedAt = now
	strategy.UpdatedAt = now

	// Initialize financial fields if not set
	if strategy.CurrentEquity.IsZero() {
		strategy.CurrentEquity = strategy.AllocatedCapital
	}
	if strategy.CashReserve.IsZero() {
		strategy.CashReserve = strategy.AllocatedCapital
	}
	if strategy.TotalPnL.IsZero() {
		strategy.TotalPnL = decimal.Zero
	}
	if strategy.TotalPnLPercent.IsZero() {
		strategy.TotalPnLPercent = decimal.Zero
	}

	if err := s.repo.Create(ctx, strategy); err != nil {
		return errors.Wrap(err, "create strategy")
	}

	s.log.Infow("Strategy created",
		"strategy_id", strategy.ID,
		"user_id", strategy.UserID,
		"allocated_capital", strategy.AllocatedCapital,
	)

	return nil
}

// GetByID retrieves a strategy by identifier
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Strategy, error) {
	if id == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}
	strategy, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get strategy")
	}
	return strategy, nil
}

// GetByUserID retrieves all strategies for a user
func (s *Service) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Strategy, error) {
	if userID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}
	strategies, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "get user strategies")
	}
	return strategies, nil
}

// GetActiveByUserID retrieves active strategies for a user
func (s *Service) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*Strategy, error) {
	if userID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}
	strategies, err := s.repo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "get active strategies")
	}
	return strategies, nil
}

// Update persists strategy changes
func (s *Service) Update(ctx context.Context, strategy *Strategy) error {
	if strategy == nil {
		return errors.ErrInvalidInput
	}
	if strategy.ID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if !strategy.Status.Valid() {
		return errors.New("invalid status")
	}

	strategy.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, strategy); err != nil {
		return errors.Wrap(err, "update strategy")
	}

	s.log.Debugw("Strategy updated",
		"strategy_id", strategy.ID,
		"status", strategy.Status,
	)

	return nil
}

// UpdateEquity updates equity and calculates PnL
func (s *Service) UpdateEquity(ctx context.Context, id uuid.UUID, positionsValue decimal.Decimal) error {
	if id == uuid.Nil {
		return errors.ErrInvalidInput
	}

	strategy, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "get strategy for equity update")
	}

	// Use entity method to recalculate
	strategy.UpdateEquity(positionsValue)

	if err := s.repo.Update(ctx, strategy); err != nil {
		return errors.Wrap(err, "update strategy equity")
	}

	s.log.Debugw("Strategy equity updated",
		"strategy_id", id,
		"current_equity", strategy.CurrentEquity,
		"total_pnl", strategy.TotalPnL,
	)

	return nil
}

// Close closes a strategy
func (s *Service) Close(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.ErrInvalidInput
	}

	strategy, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "get strategy for closing")
	}

	// Use entity method
	strategy.Close()

	if err := s.repo.Update(ctx, strategy); err != nil {
		return errors.Wrap(err, "close strategy")
	}

	s.log.Infow("Strategy closed",
		"strategy_id", id,
		"final_equity", strategy.CurrentEquity,
		"total_pnl", strategy.TotalPnL,
	)

	return nil
}

// Delete removes a strategy
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "delete strategy")
	}

	s.log.Infow("Strategy deleted", "strategy_id", id)
	return nil
}
