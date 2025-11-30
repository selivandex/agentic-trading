package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/strategy"
	"prometheus/pkg/errors"
)

// Compile-time check
var _ strategy.Repository = (*StrategyRepository)(nil)

// StrategyRepository implements strategy.Repository using PostgreSQL
type StrategyRepository struct {
	db *sqlx.DB
}

// NewStrategyRepository creates a new strategy repository
func NewStrategyRepository(db *sqlx.DB) *StrategyRepository {
	return &StrategyRepository{db: db}
}

// Create inserts a new strategy
func (r *StrategyRepository) Create(ctx context.Context, s *strategy.Strategy) error {
	query := `
		INSERT INTO user_strategies (
			id, user_id, name, description, status,
			allocated_capital, current_equity, cash_reserve,
			risk_tolerance, rebalance_frequency, target_allocations,
			total_pnl, total_pnl_percent, sharpe_ratio, max_drawdown, win_rate,
			created_at, updated_at, closed_at, last_rebalanced_at,
			reasoning_log
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11,
			$12, $13, $14, $15, $16,
			$17, $18, $19, $20,
			$21
		)`

	_, err := r.db.ExecContext(ctx, query,
		s.ID, s.UserID, s.Name, s.Description, s.Status,
		s.AllocatedCapital, s.CurrentEquity, s.CashReserve,
		s.RiskTolerance, s.RebalanceFrequency, s.TargetAllocations,
		s.TotalPnL, s.TotalPnLPercent, s.SharpeRatio, s.MaxDrawdown, s.WinRate,
		s.CreatedAt, s.UpdatedAt, s.ClosedAt, s.LastRebalancedAt,
		s.ReasoningLog,
	)

	if err != nil {
		return errors.Wrap(err, "failed to create strategy")
	}

	return nil
}

// GetByID retrieves a strategy by ID
func (r *StrategyRepository) GetByID(ctx context.Context, id uuid.UUID) (*strategy.Strategy, error) {
	var s strategy.Strategy

	query := `
		SELECT id, user_id, name, description, status,
			   allocated_capital, current_equity, cash_reserve,
			   risk_tolerance, rebalance_frequency, target_allocations,
			   total_pnl, total_pnl_percent, sharpe_ratio, max_drawdown, win_rate,
			   created_at, updated_at, closed_at, last_rebalanced_at,
			   reasoning_log
		FROM user_strategies
		WHERE id = $1`

	err := r.db.GetContext(ctx, &s, query, id)
	if err == sql.ErrNoRows {
		return nil, errors.Wrap(errors.ErrNotFound, "strategy not found")
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get strategy")
	}

	return &s, nil
}

// GetByUserID retrieves all strategies for a user
func (r *StrategyRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*strategy.Strategy, error) {
	var strategies []*strategy.Strategy

	query := `
		SELECT id, user_id, name, description, status,
			   allocated_capital, current_equity, cash_reserve,
			   risk_tolerance, rebalance_frequency, target_allocations,
			   total_pnl, total_pnl_percent, sharpe_ratio, max_drawdown, win_rate,
			   created_at, updated_at, closed_at, last_rebalanced_at,
			   reasoning_log
		FROM user_strategies
		WHERE user_id = $1
		ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &strategies, query, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get strategies")
	}

	return strategies, nil
}

// GetActiveByUserID retrieves all active strategies for a user
func (r *StrategyRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*strategy.Strategy, error) {
	var strategies []*strategy.Strategy

	query := `
		SELECT id, user_id, name, description, status,
			   allocated_capital, current_equity, cash_reserve,
			   risk_tolerance, rebalance_frequency, target_allocations,
			   total_pnl, total_pnl_percent, sharpe_ratio, max_drawdown, win_rate,
			   created_at, updated_at, closed_at, last_rebalanced_at,
			   reasoning_log
		FROM user_strategies
		WHERE user_id = $1 AND status = 'active'
		ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &strategies, query, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get active strategies")
	}

	return strategies, nil
}

// Update updates an existing strategy
func (r *StrategyRepository) Update(ctx context.Context, s *strategy.Strategy) error {
	query := `
		UPDATE user_strategies
		SET name = $2,
			description = $3,
			status = $4,
			allocated_capital = $5,
			current_equity = $6,
			cash_reserve = $7,
			risk_tolerance = $8,
			rebalance_frequency = $9,
			target_allocations = $10,
			total_pnl = $11,
			total_pnl_percent = $12,
			sharpe_ratio = $13,
			max_drawdown = $14,
			win_rate = $15,
			updated_at = $16,
			closed_at = $17,
			last_rebalanced_at = $18,
			reasoning_log = $19
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		s.ID, s.Name, s.Description, s.Status,
		s.AllocatedCapital, s.CurrentEquity, s.CashReserve,
		s.RiskTolerance, s.RebalanceFrequency, s.TargetAllocations,
		s.TotalPnL, s.TotalPnLPercent, s.SharpeRatio, s.MaxDrawdown, s.WinRate,
		s.UpdatedAt, s.ClosedAt, s.LastRebalancedAt,
		s.ReasoningLog,
	)
	if err != nil {
		return errors.Wrap(err, "failed to update strategy")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	if rows == 0 {
		return errors.Wrap(errors.ErrNotFound, "strategy not found")
	}

	return nil
}

// UpdateEquity updates strategy equity and PnL fields (optimized)
func (r *StrategyRepository) UpdateEquity(ctx context.Context, strategyID uuid.UUID, currentEquity, cashReserve, totalPnL, totalPnLPercent decimal.Decimal) error {
	query := `
		UPDATE user_strategies
		SET current_equity = $2,
			cash_reserve = $3,
			total_pnl = $4,
			total_pnl_percent = $5,
			updated_at = NOW()
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		strategyID, currentEquity, cashReserve, totalPnL, totalPnLPercent,
	)
	if err != nil {
		return errors.Wrap(err, "failed to update strategy equity")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	if rows == 0 {
		return errors.Wrap(errors.ErrNotFound, "strategy not found")
	}

	return nil
}

// Delete soft-deletes a strategy
func (r *StrategyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE user_strategies
		SET status = 'closed',
			closed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete strategy")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	if rows == 0 {
		return errors.Wrap(errors.ErrNotFound, "strategy not found")
	}

	return nil
}

// GetTotalAllocatedCapital returns sum of allocated capital across all active strategies
func (r *StrategyRepository) GetTotalAllocatedCapital(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	var total decimal.Decimal

	query := `
		SELECT COALESCE(SUM(allocated_capital), 0)
		FROM user_strategies
		WHERE user_id = $1 AND status = 'active'`

	err := r.db.GetContext(ctx, &total, query, userID)
	if err != nil {
		return decimal.Zero, errors.Wrap(err, "failed to get total allocated capital")
	}

	return total, nil
}

// GetTotalCurrentEquity returns sum of current equity across all active strategies
func (r *StrategyRepository) GetTotalCurrentEquity(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	var total decimal.Decimal

	query := `
		SELECT COALESCE(SUM(current_equity), 0)
		FROM user_strategies
		WHERE user_id = $1 AND status = 'active'`

	err := r.db.GetContext(ctx, &total, query, userID)
	if err != nil {
		return decimal.Zero, errors.Wrap(err, "failed to get total current equity")
	}

	return total, nil
}
