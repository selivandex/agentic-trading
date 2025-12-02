package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/position"
	"prometheus/pkg/errors"
)

// Compile-time check
var _ position.Repository = (*PositionRepository)(nil)

// PositionRepository implements position.Repository using sqlx
type PositionRepository struct {
	db DBTX
}

// NewPositionRepository creates a new position repository
func NewPositionRepository(db DBTX) *PositionRepository {
	return &PositionRepository{db: db}
}

// Create inserts a new position
func (r *PositionRepository) Create(ctx context.Context, p *position.Position) error {
	query := `
		INSERT INTO positions (
			id, user_id, strategy_id, exchange_account_id,
			symbol, market_type, side,
			size, entry_price, current_price, liquidation_price,
			leverage, margin_mode,
			unrealized_pnl, unrealized_pnl_pct, realized_pnl,
			stop_loss_price, take_profit_price, trailing_stop_pct,
			stop_loss_order_id, take_profit_order_id,
			open_reasoning,
			status, opened_at, closed_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26
		)`

	_, err := r.db.ExecContext(ctx, query,
		p.ID, p.UserID, p.StrategyID, p.ExchangeAccountID,
		p.Symbol, p.MarketType, p.Side,
		p.Size, p.EntryPrice, p.CurrentPrice, p.LiquidationPrice,
		p.Leverage, p.MarginMode,
		p.UnrealizedPnL, p.UnrealizedPnLPct, p.RealizedPnL,
		p.StopLossPrice, p.TakeProfitPrice, p.TrailingStopPct,
		p.StopLossOrderID, p.TakeProfitOrderID,
		p.OpenReasoning,
		p.Status, p.OpenedAt, p.ClosedAt, p.UpdatedAt,
	)

	return err
}

// GetByID retrieves a position by ID
func (r *PositionRepository) GetByID(ctx context.Context, id uuid.UUID) (*position.Position, error) {
	var p position.Position

	query := `SELECT * FROM positions WHERE id = $1`

	err := r.db.GetContext(ctx, &p, query, id)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

// GetOpenByUser retrieves all open positions for a user
func (r *PositionRepository) GetOpenByUser(ctx context.Context, userID uuid.UUID) ([]*position.Position, error) {
	var positions []*position.Position

	query := `
		SELECT * FROM positions
		WHERE user_id = $1 AND status = 'open'
		ORDER BY opened_at DESC`

	err := r.db.SelectContext(ctx, &positions, query, userID)
	if err != nil {
		return nil, err
	}

	return positions, nil
}

// GetOpenByStrategy retrieves all open positions for a strategy
func (r *PositionRepository) GetOpenByStrategy(ctx context.Context, strategyID uuid.UUID) ([]*position.Position, error) {
	var positions []*position.Position

	query := `
		SELECT * FROM positions
		WHERE strategy_id = $1 AND status = 'open'
		ORDER BY opened_at DESC`

	err := r.db.SelectContext(ctx, &positions, query, strategyID)
	if err != nil {
		return nil, err
	}

	return positions, nil
}

// GetByStrategy retrieves all positions (open and closed) for a strategy
func (r *PositionRepository) GetByStrategy(ctx context.Context, strategyID uuid.UUID) ([]*position.Position, error) {
	var positions []*position.Position

	query := `
		SELECT * FROM positions
		WHERE strategy_id = $1
		ORDER BY opened_at DESC`

	err := r.db.SelectContext(ctx, &positions, query, strategyID)
	if err != nil {
		return nil, err
	}

	return positions, nil
}

// Update updates a position
func (r *PositionRepository) Update(ctx context.Context, p *position.Position) error {
	query := `
		UPDATE positions SET
			current_price = $2,
			unrealized_pnl = $3,
			unrealized_pnl_pct = $4,
			realized_pnl = $5,
			stop_loss_price = $6,
			take_profit_price = $7,
			trailing_stop_pct = $8,
			status = $9,
			closed_at = $10,
			updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		p.ID, p.CurrentPrice, p.UnrealizedPnL, p.UnrealizedPnLPct, p.RealizedPnL,
		p.StopLossPrice, p.TakeProfitPrice, p.TrailingStopPct,
		p.Status, p.ClosedAt,
	)

	return err
}

// UpdatePnL updates position PnL
func (r *PositionRepository) UpdatePnL(ctx context.Context, id uuid.UUID, currentPrice, unrealizedPnL, unrealizedPnLPct decimal.Decimal) error {
	query := `
		UPDATE positions SET
			current_price = $2,
			unrealized_pnl = $3,
			unrealized_pnl_pct = $4,
			updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, currentPrice, unrealizedPnL, unrealizedPnLPct)
	return err
}

// Close closes a position
func (r *PositionRepository) Close(ctx context.Context, id uuid.UUID, exitPrice, realizedPnL decimal.Decimal) error {
	query := `
		UPDATE positions SET
			current_price = $2,
			realized_pnl = $3,
			status = 'closed',
			closed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, exitPrice, realizedPnL)
	return err
}

// GetClosedInRange retrieves closed positions for a user within a time range
func (r *PositionRepository) GetClosedInRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*position.Position, error) {
	var positions []*position.Position

	query := `
		SELECT * FROM positions
		WHERE user_id = $1
		  AND status = 'closed'
		  AND closed_at >= $2
		  AND closed_at < $3
		ORDER BY closed_at DESC`

	err := r.db.SelectContext(ctx, &positions, query, userID, start, end)
	if err != nil {
		return nil, err
	}

	return positions, nil
}

// Delete deletes a position
func (r *PositionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM positions WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// UpdatePnLBatch updates PnL for multiple positions
// NOTE: Caller should manage transaction if atomicity is required
func (r *PositionRepository) UpdatePnLBatch(ctx context.Context, updates []PositionPnLUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	query := `
		UPDATE positions SET
			current_price = $2,
			unrealized_pnl = $3,
			unrealized_pnl_pct = $4,
			updated_at = NOW()
		WHERE id = $1`

	for i, update := range updates {
		result, err := r.db.ExecContext(ctx, query,
			update.PositionID,
			update.CurrentPrice,
			update.UnrealizedPnL,
			update.UnrealizedPnLPct,
		)
		if err != nil {
			return errors.Wrapf(err, "failed to update position at index %d", i)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return errors.Wrapf(err, "failed to get rows affected at index %d", i)
		}

		if rows == 0 {
			return errors.Wrapf(errors.ErrPositionNotFound, "position %s at index %d", update.PositionID, i)
		}
	}

	return nil
}

// PositionPnLUpdate represents a batch PnL update
type PositionPnLUpdate struct {
	PositionID       uuid.UUID
	CurrentPrice     decimal.Decimal
	UnrealizedPnL    decimal.Decimal
	UnrealizedPnLPct decimal.Decimal
}

// ClosePositionsBatch closes multiple positions
func (r *PositionRepository) ClosePositionsBatch(ctx context.Context, positionIDs []uuid.UUID, reason string) error {
	if len(positionIDs) == 0 {
		return nil
	}

	query := `
		UPDATE positions SET
			status = $2,
			closed_at = NOW(),
			updated_at = NOW()
		WHERE id = ANY($1)
			AND status = 'open'`

	result, err := r.db.ExecContext(ctx, query, positionIDs, position.PositionClosed)
	if err != nil {
		return errors.Wrap(err, "failed to close positions")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return errors.Wrap(errors.ErrPositionNotFound, "no positions were closed")
	}

	return nil
}

// UpdateRealizedPnLBatch updates realized PnL for closed positions
func (r *PositionRepository) UpdateRealizedPnLBatch(ctx context.Context, updates []PositionRealizedPnLUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	// NOTE: Caller should manage transaction if atomicity is required
	query := `
		UPDATE positions SET
			realized_pnl = $2,
			status = $3,
			closed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1`

	for i, update := range updates {
		result, err := r.db.ExecContext(ctx, query,
			update.PositionID,
			update.RealizedPnL,
			position.PositionClosed,
		)
		if err != nil {
			return errors.Wrapf(err, "failed to update realized PnL at index %d", i)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return errors.Wrapf(err, "failed to get rows affected at index %d", i)
		}

		if rows == 0 {
			return errors.Wrapf(errors.ErrPositionNotFound, "position %s at index %d", update.PositionID, i)
		}
	}

	return nil
}

// PositionRealizedPnLUpdate represents a realized PnL update
type PositionRealizedPnLUpdate struct {
	PositionID  uuid.UUID
	RealizedPnL decimal.Decimal
}
