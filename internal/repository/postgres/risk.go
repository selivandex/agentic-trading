package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"prometheus/internal/domain/risk"
)

// Compile-time check
var _ risk.Repository = (*RiskRepository)(nil)

// RiskRepository implements risk.Repository using sqlx
type RiskRepository struct {
	db *sqlx.DB
}

// NewRiskRepository creates a new risk repository
func NewRiskRepository(db *sqlx.DB) *RiskRepository {
	return &RiskRepository{db: db}
}

// GetState retrieves circuit breaker state for a user
func (r *RiskRepository) GetState(ctx context.Context, userID uuid.UUID) (*risk.CircuitBreakerState, error) {
	var state risk.CircuitBreakerState

	query := `SELECT * FROM circuit_breaker_states WHERE user_id = $1`

	err := r.db.GetContext(ctx, &state, query, userID)
	if err != nil {
		return nil, err
	}

	return &state, nil
}

// SaveState saves circuit breaker state
func (r *RiskRepository) SaveState(ctx context.Context, state *risk.CircuitBreakerState) error {
	query := `
		INSERT INTO circuit_breaker_states (
			user_id, is_triggered, triggered_at, trigger_reason,
			daily_pnl, daily_pnl_percent, daily_trade_count,
			daily_wins, daily_losses, consecutive_losses,
			max_daily_drawdown, max_consecutive_loss,
			reset_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
		ON CONFLICT (user_id) DO UPDATE SET
			is_triggered = $2,
			triggered_at = $3,
			trigger_reason = $4,
			daily_pnl = $5,
			daily_pnl_percent = $6,
			daily_trade_count = $7,
			daily_wins = $8,
			daily_losses = $9,
			consecutive_losses = $10,
			max_daily_drawdown = $11,
			max_consecutive_loss = $12,
			reset_at = $13,
			updated_at = NOW()`

	_, err := r.db.ExecContext(ctx, query,
		state.UserID, state.IsTriggered, state.TriggeredAt, state.TriggerReason,
		state.DailyPnL, state.DailyPnLPercent, state.DailyTradeCount,
		state.DailyWins, state.DailyLosses, state.ConsecutiveLosses,
		state.MaxDailyDrawdown, state.MaxConsecutiveLoss,
		state.ResetAt, state.UpdatedAt,
	)

	return err
}

// ResetDaily resets all daily stats for all users
func (r *RiskRepository) ResetDaily(ctx context.Context) error {
	query := `
		UPDATE circuit_breaker_states SET
			daily_pnl = 0,
			daily_pnl_percent = 0,
			daily_trade_count = 0,
			daily_wins = 0,
			daily_losses = 0,
			consecutive_losses = 0,
			is_triggered = false,
			triggered_at = NULL,
			trigger_reason = NULL,
			reset_at = DATE_TRUNC('day', NOW()) + INTERVAL '1 day',
			updated_at = NOW()`

	_, err := r.db.ExecContext(ctx, query)
	return err
}

// CreateEvent creates a new risk event
func (r *RiskRepository) CreateEvent(ctx context.Context, event *risk.RiskEvent) error {
	query := `
		INSERT INTO risk_events (
			id, user_id, timestamp, event_type, severity, message, data, acknowledged
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)`

	_, err := r.db.ExecContext(ctx, query,
		event.ID, event.UserID, event.Timestamp, event.EventType,
		event.Severity, event.Message, event.Data, event.Acknowledged,
	)

	return err
}

// GetEvents retrieves risk events for a user
func (r *RiskRepository) GetEvents(ctx context.Context, userID uuid.UUID, limit int) ([]*risk.RiskEvent, error) {
	var events []*risk.RiskEvent

	query := `
		SELECT * FROM risk_events
		WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT $2`

	err := r.db.SelectContext(ctx, &events, query, userID, limit)
	if err != nil {
		return nil, err
	}

	return events, nil
}

// AcknowledgeEvent marks a risk event as acknowledged
func (r *RiskRepository) AcknowledgeEvent(ctx context.Context, eventID uuid.UUID) error {
	query := `UPDATE risk_events SET acknowledged = true WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, eventID)
	return err
}
