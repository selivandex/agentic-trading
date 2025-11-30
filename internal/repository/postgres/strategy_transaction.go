package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/strategy"
	"prometheus/pkg/errors"
)

// Compile-time check
var _ strategy.TransactionRepository = (*StrategyTransactionRepository)(nil)

// StrategyTransactionRepository implements strategy.TransactionRepository using PostgreSQL
type StrategyTransactionRepository struct {
	db *sqlx.DB
}

// NewStrategyTransactionRepository creates a new strategy transaction repository
func NewStrategyTransactionRepository(db *sqlx.DB) *StrategyTransactionRepository {
	return &StrategyTransactionRepository{db: db}
}

// Create inserts a new transaction
func (r *StrategyTransactionRepository) Create(ctx context.Context, tx *strategy.Transaction) error {
	query := `
		INSERT INTO strategy_transactions (
			id, strategy_id, user_id,
			type, amount, balance_before, balance_after,
			position_id, order_id,
			description, metadata,
			created_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7,
			$8, $9,
			$10, $11,
			$12
		)`

	_, err := r.db.ExecContext(ctx, query,
		tx.ID, tx.StrategyID, tx.UserID,
		tx.Type, tx.Amount, tx.BalanceBefore, tx.BalanceAfter,
		tx.PositionID, tx.OrderID,
		tx.Description, tx.Metadata,
		tx.CreatedAt,
	)

	if err != nil {
		return errors.Wrap(err, "failed to create strategy transaction")
	}

	return nil
}

// GetByID retrieves a transaction by ID
func (r *StrategyTransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*strategy.Transaction, error) {
	var tx strategy.Transaction

	query := `
		SELECT id, strategy_id, user_id,
			   type, amount, balance_before, balance_after,
			   position_id, order_id,
			   description, metadata,
			   created_at
		FROM strategy_transactions
		WHERE id = $1`

	err := r.db.GetContext(ctx, &tx, query, id)
	if err == sql.ErrNoRows {
		return nil, errors.Wrap(errors.ErrNotFound, "transaction not found")
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get transaction")
	}

	return &tx, nil
}

// GetByStrategyID retrieves all transactions for a strategy
func (r *StrategyTransactionRepository) GetByStrategyID(ctx context.Context, strategyID uuid.UUID, limit int) ([]*strategy.Transaction, error) {
	if limit <= 0 {
		limit = 100 // Default limit
	}

	var transactions []*strategy.Transaction

	query := `
		SELECT id, strategy_id, user_id,
			   type, amount, balance_before, balance_after,
			   position_id, order_id,
			   description, metadata,
			   created_at
		FROM strategy_transactions
		WHERE strategy_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	err := r.db.SelectContext(ctx, &transactions, query, strategyID, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get transactions")
	}

	return transactions, nil
}

// GetByUserID retrieves all transactions for a user
func (r *StrategyTransactionRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*strategy.Transaction, error) {
	if limit <= 0 {
		limit = 100 // Default limit
	}

	var transactions []*strategy.Transaction

	query := `
		SELECT id, strategy_id, user_id,
			   type, amount, balance_before, balance_after,
			   position_id, order_id,
			   description, metadata,
			   created_at
		FROM strategy_transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	err := r.db.SelectContext(ctx, &transactions, query, userID, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get transactions")
	}

	return transactions, nil
}

// GetByPositionID retrieves all transactions related to a position
func (r *StrategyTransactionRepository) GetByPositionID(ctx context.Context, positionID uuid.UUID) ([]*strategy.Transaction, error) {
	var transactions []*strategy.Transaction

	query := `
		SELECT id, strategy_id, user_id,
			   type, amount, balance_before, balance_after,
			   position_id, order_id,
			   description, metadata,
			   created_at
		FROM strategy_transactions
		WHERE position_id = $1
		ORDER BY created_at ASC`

	err := r.db.SelectContext(ctx, &transactions, query, positionID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get transactions by position")
	}

	return transactions, nil
}

// GetByDateRange retrieves transactions in a date range
func (r *StrategyTransactionRepository) GetByDateRange(ctx context.Context, strategyID uuid.UUID, start, end time.Time) ([]*strategy.Transaction, error) {
	var transactions []*strategy.Transaction

	query := `
		SELECT id, strategy_id, user_id,
			   type, amount, balance_before, balance_after,
			   position_id, order_id,
			   description, metadata,
			   created_at
		FROM strategy_transactions
		WHERE strategy_id = $1
		  AND created_at >= $2
		  AND created_at <= $3
		ORDER BY created_at ASC`

	err := r.db.SelectContext(ctx, &transactions, query, strategyID, start, end)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get transactions by date range")
	}

	return transactions, nil
}

// GetLatestBalance gets the most recent balance_after for a strategy
func (r *StrategyTransactionRepository) GetLatestBalance(ctx context.Context, strategyID uuid.UUID) (decimal.Decimal, error) {
	var balance decimal.Decimal

	query := `
		SELECT balance_after
		FROM strategy_transactions
		WHERE strategy_id = $1
		ORDER BY created_at DESC
		LIMIT 1`

	err := r.db.GetContext(ctx, &balance, query, strategyID)
	if err == sql.ErrNoRows {
		// No transactions yet, return zero
		return decimal.Zero, nil
	}
	if err != nil {
		return decimal.Zero, errors.Wrap(err, "failed to get latest balance")
	}

	return balance, nil
}

// GetTransactionStats returns aggregate stats for a strategy
func (r *StrategyTransactionRepository) GetTransactionStats(ctx context.Context, strategyID uuid.UUID) (*strategy.TransactionStats, error) {
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN type = 'deposit' THEN amount ELSE 0 END), 0) as total_deposits,
			COALESCE(SUM(CASE WHEN type = 'withdrawal' THEN ABS(amount) ELSE 0 END), 0) as total_withdrawals,
			COALESCE(SUM(CASE WHEN type = 'fee' THEN ABS(amount) ELSE 0 END), 0) as total_fees,
			COALESCE(SUM(CASE WHEN type = 'position_close' THEN amount ELSE 0 END), 0) as total_pnl,
			COUNT(*) as transaction_count
		FROM strategy_transactions
		WHERE strategy_id = $1`

	var stats strategy.TransactionStats
	err := r.db.GetContext(ctx, &stats, query, strategyID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get transaction stats")
	}

	return &stats, nil
}
