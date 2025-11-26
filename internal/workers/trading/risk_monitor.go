package trading

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/position"
	"prometheus/internal/risk"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// RiskMonitor monitors user risk levels and trips circuit breakers when needed
type RiskMonitor struct {
	*workers.BaseWorker
	riskEngine *risk.RiskEngine
	posRepo    position.Repository
	kafka      *kafka.Producer
}

// NewRiskMonitor creates a new risk monitor worker
func NewRiskMonitor(
	riskEngine *risk.RiskEngine,
	posRepo position.Repository,
	kafka *kafka.Producer,
	interval time.Duration,
	enabled bool,
) *RiskMonitor {
	return &RiskMonitor{
		BaseWorker: workers.NewBaseWorker("risk_monitor", interval, enabled),
		riskEngine: riskEngine,
		posRepo:    posRepo,
		kafka:      kafka,
	}
}

// Run executes one iteration of risk monitoring
func (rm *RiskMonitor) Run(ctx context.Context) error {
	rm.Log().Debug("Risk monitor: starting iteration")

	// This is a simplified implementation
	// In production, we'd need a user repository to get all active users
	// For now, we'll log that this needs implementation
	rm.Log().Warn("Risk monitor: simplified implementation - need user repository to monitor all users")

	return nil
}

// monitorUser monitors risk levels for a specific user
func (rm *RiskMonitor) monitorUser(ctx context.Context, userID uuid.UUID) error {
	// Get user's risk state
	state, err := rm.riskEngine.GetUserState(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "failed to get user risk state")
	}

	rm.Log().Debug("Monitoring user risk",
		"user_id", userID,
		"daily_pnl", state.DailyPnL,
		"consecutive_losses", state.ConsecutiveLosses,
		"can_trade", state.CanTrade,
	)

	// If circuit breaker is already tripped, send notification if not acknowledged
	if state.IsCircuitTripped {
		rm.sendCircuitBreakerAlert(ctx, userID, state)
		return nil
	}

	// Calculate current exposure
	positions, err := rm.posRepo.GetOpenByUser(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "failed to get open positions")
	}

	totalExposure := decimal.Zero
	totalUnrealizedPnL := decimal.Zero
	for _, pos := range positions {
		positionValue := pos.Size.Mul(pos.EntryPrice)
		if pos.Leverage > 0 {
			positionValue = positionValue.Div(decimal.NewFromInt(int64(pos.Leverage)))
		}
		totalExposure = totalExposure.Add(positionValue)
		totalUnrealizedPnL = totalUnrealizedPnL.Add(pos.UnrealizedPnL)
	}

	// Check if approaching drawdown limit (80%)
	if state.DailyPnL.LessThan(decimal.Zero) {
		absDrawdown := state.DailyPnL.Abs()
		warningThreshold := state.MaxDrawdown.Mul(decimal.NewFromFloat(0.80))

		if absDrawdown.GreaterThanOrEqual(warningThreshold) && absDrawdown.LessThan(state.MaxDrawdown) {
			rm.sendDrawdownWarning(ctx, userID, state, absDrawdown)
		}
	}

	// Check consecutive losses warning (at limit - 1)
	if state.ConsecutiveLosses >= state.MaxConsecutiveLoss-1 && state.ConsecutiveLosses < state.MaxConsecutiveLoss {
		rm.sendConsecutiveLossesWarning(ctx, userID, state)
	}

	rm.Log().Debug("Risk monitoring complete",
		"user_id", userID,
		"total_exposure", totalExposure,
		"total_unrealized_pnl", totalUnrealizedPnL,
	)

	return nil
}

// Event structures

type CircuitBreakerAlert struct {
	UserID            string    `json:"user_id"`
	Reason            string    `json:"reason"`
	DailyPnL          string    `json:"daily_pnl"`
	DailyPnLPercent   string    `json:"daily_pnl_percent"`
	ConsecutiveLosses int       `json:"consecutive_losses"`
	MaxDrawdown       string    `json:"max_drawdown"`
	Timestamp         time.Time `json:"timestamp"`
}

type DrawdownWarningEvent struct {
	UserID          string    `json:"user_id"`
	CurrentDrawdown string    `json:"current_drawdown"`
	MaxDrawdown     string    `json:"max_drawdown"`
	Percentage      string    `json:"percentage"` // e.g. "85%"
	DailyPnL        string    `json:"daily_pnl"`
	Timestamp       time.Time `json:"timestamp"`
}

type ConsecutiveLossesWarningEvent struct {
	UserID            string    `json:"user_id"`
	ConsecutiveLosses int       `json:"consecutive_losses"`
	MaxAllowed        int       `json:"max_allowed"`
	Timestamp         time.Time `json:"timestamp"`
}

func (rm *RiskMonitor) sendCircuitBreakerAlert(ctx context.Context, userID uuid.UUID, state *risk.UserRiskState) {
	event := CircuitBreakerAlert{
		UserID:            userID.String(),
		Reason:            state.TripReason,
		DailyPnL:          state.DailyPnL.String(),
		DailyPnLPercent:   state.DailyPnLPercent.String(),
		ConsecutiveLosses: state.ConsecutiveLosses,
		MaxDrawdown:       state.MaxDrawdown.String(),
		Timestamp:         time.Now(),
	}

	if err := rm.kafka.Publish(ctx, "risk.circuit_breaker_tripped", userID.String(), event); err != nil {
		rm.Log().Error("Failed to publish circuit breaker alert", "error", err)
	} else {
		rm.Log().Warn("Circuit breaker alert sent",
			"user_id", userID,
			"reason", state.TripReason,
		)
	}
}

func (rm *RiskMonitor) sendDrawdownWarning(ctx context.Context, userID uuid.UUID, state *risk.UserRiskState, currentDrawdown decimal.Decimal) {
	percentage := currentDrawdown.Div(state.MaxDrawdown).Mul(decimal.NewFromInt(100))

	event := DrawdownWarningEvent{
		UserID:          userID.String(),
		CurrentDrawdown: currentDrawdown.String(),
		MaxDrawdown:     state.MaxDrawdown.String(),
		Percentage:      percentage.StringFixed(1) + "%",
		DailyPnL:        state.DailyPnL.String(),
		Timestamp:       time.Now(),
	}

	if err := rm.kafka.Publish(ctx, "risk.drawdown_warning", userID.String(), event); err != nil {
		rm.Log().Error("Failed to publish drawdown warning", "error", err)
	} else {
		rm.Log().Warn("Drawdown warning sent",
			"user_id", userID,
			"current_drawdown", currentDrawdown,
			"percentage", percentage.StringFixed(1)+"%",
		)
	}
}

func (rm *RiskMonitor) sendConsecutiveLossesWarning(ctx context.Context, userID uuid.UUID, state *risk.UserRiskState) {
	event := ConsecutiveLossesWarningEvent{
		UserID:            userID.String(),
		ConsecutiveLosses: state.ConsecutiveLosses,
		MaxAllowed:        state.MaxConsecutiveLoss,
		Timestamp:         time.Now(),
	}

	if err := rm.kafka.Publish(ctx, "risk.consecutive_losses", userID.String(), event); err != nil {
		rm.Log().Error("Failed to publish consecutive losses warning", "error", err)
	} else {
		rm.Log().Warn("Consecutive losses warning sent",
			"user_id", userID,
			"consecutive_losses", state.ConsecutiveLosses,
			"max_allowed", state.MaxConsecutiveLoss,
		)
	}
}
