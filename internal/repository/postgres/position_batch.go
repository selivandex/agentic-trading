package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/position"
	"prometheus/pkg/errors"
)

// UpdatePnLBatch updates PnL for multiple positions in a single transaction
func (r *PositionRepository) UpdatePnLBatch(ctx context.Context, updates []PositionPnLUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	query := `
		UPDATE positions SET
			current_price = $2,
			unrealized_pnl = $3,
			unrealized_pnl_pct = $4,
			updated_at = NOW()
		WHERE id = $1`

	for i, update := range updates {
		result, err := tx.ExecContext(ctx, query,
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

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit batch PnL update")
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

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	query := `
		UPDATE positions SET
			realized_pnl = $2,
			status = $3,
			closed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1`

	for i, update := range updates {
		result, err := tx.ExecContext(ctx, query,
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

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit batch realized PnL update")
	}

	return nil
}

// PositionRealizedPnLUpdate represents a realized PnL update
type PositionRealizedPnLUpdate struct {
	PositionID  uuid.UUID
	RealizedPnL decimal.Decimal
}
