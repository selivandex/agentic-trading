package position

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/position"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// CriticalEventHandler handles critical position events with immediate algorithmic action
// NO LLM, NO delays - deterministic and fast (<1 second)
type CriticalEventHandler struct {
	posRepo position.Repository
	log     *logger.Logger
}

// NewCriticalEventHandler creates a new critical event handler
func NewCriticalEventHandler(
	posRepo position.Repository,
	log *logger.Logger,
) *CriticalEventHandler {
	return &CriticalEventHandler{
		posRepo: posRepo,
		log:     log,
	}
}

// HandleStopLossHit handles stop loss hit - closes position immediately
func (h *CriticalEventHandler) HandleStopLossHit(ctx context.Context, event *eventspb.PositionClosedEvent) error {
	h.log.Warn("CRITICAL: Stop loss hit - closing position immediately",
		"position_id", event.PositionId,
		"symbol", event.Symbol,
		"user_id", event.Base.UserId,
		"stop_price", event.ExitPrice,
	)

	// Parse position ID
	positionID, err := uuid.Parse(event.PositionId)
	if err != nil {
		return errors.Wrap(err, "invalid position ID")
	}

	// Get position
	pos, err := h.posRepo.GetByID(ctx, positionID)
	if err != nil {
		return errors.Wrap(err, "failed to get position")
	}

	if pos == nil {
		h.log.Warn("Position not found (already closed?)", "position_id", event.PositionId)
		return nil
	}

	// If position already closed, skip
	if pos.Status == position.PositionClosed {
		h.log.Infow("Position already closed", "position_id", event.PositionId)
		return nil
	}

	// Close position immediately (in production, this would place a market sell order on exchange)
	exitPrice := decimal.NewFromFloat(event.ExitPrice)
	pnl := exitPrice.Sub(pos.EntryPrice).Mul(pos.Size) // Simplified PnL calculation
	if pos.Side == position.PositionShort {
		pnl = pnl.Neg()
	}

	if err := h.posRepo.Close(ctx, positionID, exitPrice, pnl); err != nil {
		h.log.Errorw("Failed to close position at stop loss",
			"position_id", event.PositionId,
			"error", err,
		)
		return errors.Wrap(err, "failed to close position")
	}

	h.log.Infow("Position closed at stop loss",
		"position_id", event.PositionId,
		"symbol", event.Symbol,
		"exit_price", event.ExitPrice,
		"pnl_pct", event.PnlPercent,
	)

	// TODO: Send notification to user (when telegram bot is implemented)

	return nil
}

// HandleTakeProfitHit handles take profit hit - closes position immediately
func (h *CriticalEventHandler) HandleTakeProfitHit(ctx context.Context, event *eventspb.PositionClosedEvent) error {
	h.log.Infow("CRITICAL: Take profit hit - closing position immediately",
		"position_id", event.PositionId,
		"symbol", event.Symbol,
		"user_id", event.Base.UserId,
		"target_price", event.ExitPrice,
	)

	// Parse position ID
	positionID, err := uuid.Parse(event.PositionId)
	if err != nil {
		return errors.Wrap(err, "invalid position ID")
	}

	// Get position
	pos, err := h.posRepo.GetByID(ctx, positionID)
	if err != nil {
		return errors.Wrap(err, "failed to get position")
	}

	if pos == nil {
		h.log.Warnw("Position not found (already closed?)", "position_id", event.PositionId)
		return nil
	}

	// If position already closed, skip
	if pos.Status == position.PositionClosed {
		h.log.Infow("Position already closed", "position_id", event.PositionId)
		return nil
	}

	// Close position immediately
	exitPrice := decimal.NewFromFloat(event.ExitPrice)
	pnl := exitPrice.Sub(pos.EntryPrice).Mul(pos.Size)
	if pos.Side == position.PositionShort {
		pnl = pnl.Neg()
	}

	if err := h.posRepo.Close(ctx, positionID, exitPrice, pnl); err != nil {
		h.log.Errorw("Failed to close position at take profit",
			"position_id", event.PositionId,
			"error", err,
		)
		return errors.Wrap(err, "failed to close position")
	}

	h.log.Infow("Position closed at take profit",
		"position_id", event.PositionId,
		"symbol", event.Symbol,
		"exit_price", event.ExitPrice,
		"pnl_pct", event.PnlPercent,
	)

	// TODO: Send notification to user (when telegram bot is implemented)

	return nil
}

// HandleCircuitBreakerTripped handles circuit breaker event - closes ALL user positions
func (h *CriticalEventHandler) HandleCircuitBreakerTripped(ctx context.Context, event *eventspb.CircuitBreakerTrippedEvent) error {
	h.log.Errorw("CRITICAL: Circuit breaker tripped - closing ALL positions",
		"user_id", event.Base.UserId,
		"reason", event.Reason,
		"drawdown", event.Drawdown,
	)

	// Parse user ID
	userID, err := uuid.Parse(event.Base.UserId)
	if err != nil {
		return errors.Wrap(err, "invalid user ID")
	}

	// Get all open positions for this user
	positions, err := h.posRepo.GetOpenByUser(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "failed to get open positions")
	}

	if len(positions) == 0 {
		h.log.Infow("No open positions to close", "user_id", event.Base.UserId)
		return nil
	}

	h.log.Warnw("Closing positions due to circuit breaker",
		"user_id", event.Base.UserId,
		"positions_count", len(positions),
	)

	// Close all positions
	successCount := 0
	failCount := 0
	for _, pos := range positions {
		// Close at current price with current PnL
		if err := h.posRepo.Close(ctx, pos.ID, pos.CurrentPrice, pos.UnrealizedPnL); err != nil {
			h.log.Errorw("Failed to close position during circuit breaker",
				"position_id", pos.ID,
				"symbol", pos.Symbol,
				"error", err,
			)
			failCount++
		} else {
			successCount++
		}
	}

	h.log.Warnw("Circuit breaker positions closed",
		"user_id", event.Base.UserId,
		"success", successCount,
		"failed", failCount,
	)

	// TODO: Send notification to user (when telegram bot is implemented)

	return nil
}
