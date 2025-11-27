package trading

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/position"
	"prometheus/internal/domain/user"
	"prometheus/internal/events"
	riskservice "prometheus/internal/services/risk"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// RiskMonitor monitors user risk levels and trips circuit breakers when needed
type RiskMonitor struct {
	*workers.BaseWorker
	userRepo       user.Repository
	riskEngine     *riskservice.RiskEngine
	posRepo        position.Repository
	kafka          *kafka.Producer
	eventPublisher *events.WorkerPublisher
}

// NewRiskMonitor creates a new risk monitor worker
func NewRiskMonitor(
	userRepo user.Repository,
	riskEngine *riskservice.RiskEngine,
	posRepo position.Repository,
	kafka *kafka.Producer,
	interval time.Duration,
	enabled bool,
) *RiskMonitor {
	return &RiskMonitor{
		BaseWorker:     workers.NewBaseWorker("risk_monitor", interval, enabled),
		userRepo:       userRepo,
		riskEngine:     riskEngine,
		posRepo:        posRepo,
		kafka:          kafka,
		eventPublisher: events.NewWorkerPublisher(kafka),
	}
}

// Run executes one iteration of risk monitoring
func (rm *RiskMonitor) Run(ctx context.Context) error {
	rm.Log().Debug("Risk monitor: starting iteration")

	// Get all active users (using large limit)
	users, err := rm.userRepo.List(ctx, 1000, 0)
	if err != nil {
		return errors.Wrap(err, "failed to list users")
	}

	// Filter only active users
	var activeUsers []*user.User
	for _, usr := range users {
		if usr.IsActive {
			activeUsers = append(activeUsers, usr)
		}
	}

	if len(activeUsers) == 0 {
		rm.Log().Debug("No active users to monitor")
		return nil
	}

	rm.Log().Debug("Monitoring risk for users", "user_count", len(activeUsers))

	// Monitor risk for each user
	successCount := 0
	errorCount := 0
	for _, usr := range activeUsers {
		// Check for context cancellation (graceful shutdown)
		select {
		case <-ctx.Done():
			rm.Log().Info("Risk monitoring interrupted by shutdown",
				"users_processed", successCount,
				"users_remaining", len(activeUsers)-successCount-errorCount,
			)
			return ctx.Err()
		default:
		}

		if err := rm.monitorUser(ctx, usr.ID); err != nil {
			rm.Log().Error("Failed to monitor user risk",
				"user_id", usr.ID,
				"error", err,
			)
			errorCount++
			// Continue with other users
			continue
		}
		successCount++
	}

	rm.Log().Info("Risk monitor: iteration complete",
		"users_processed", successCount,
		"errors", errorCount,
	)

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

func (rm *RiskMonitor) sendCircuitBreakerAlert(ctx context.Context, userID uuid.UUID, state *riskservice.UserRiskState) {
	dailyPnL, _ := state.DailyPnL.Float64()
	maxDrawdown, _ := state.MaxDrawdown.Float64()
	threshold := maxDrawdown * 0.9 // Assuming 90% threshold

	if err := rm.eventPublisher.PublishCircuitBreakerTripped(
		ctx,
		userID.String(),
		state.TripReason,
		dailyPnL,
		threshold,
		maxDrawdown,
		true, // auto_resume after 24h
	); err != nil {
		rm.Log().Error("Failed to publish circuit breaker alert", "error", err)
	} else {
		rm.Log().Warn("Circuit breaker alert sent",
			"user_id", userID,
			"reason", state.TripReason,
		)
	}
}

func (rm *RiskMonitor) sendDrawdownWarning(ctx context.Context, userID uuid.UUID, state *riskservice.UserRiskState, currentDrawdown decimal.Decimal) {
	percentage := currentDrawdown.Div(state.MaxDrawdown).Mul(decimal.NewFromInt(100))

	currentDrawdownFloat, _ := currentDrawdown.Float64()
	maxDrawdownFloat, _ := state.MaxDrawdown.Float64()
	percentageFloat, _ := percentage.Float64()
	dailyPnLFloat, _ := state.DailyPnL.Float64()

	if err := rm.eventPublisher.PublishDrawdownAlert(
		ctx,
		userID.String(),
		"approaching_limit",
		currentDrawdownFloat,
		maxDrawdownFloat,
		percentageFloat,
		dailyPnLFloat,
	); err != nil {
		rm.Log().Error("Failed to publish drawdown warning", "error", err)
	} else {
		rm.Log().Warn("Drawdown warning sent",
			"user_id", userID,
			"current_drawdown", currentDrawdown,
			"percentage", percentage.StringFixed(1)+"%",
		)
	}
}

func (rm *RiskMonitor) sendConsecutiveLossesWarning(ctx context.Context, userID uuid.UUID, state *riskservice.UserRiskState) {
	if err := rm.eventPublisher.PublishConsecutiveLossesAlert(
		ctx,
		userID.String(),
		state.ConsecutiveLosses,
		state.MaxConsecutiveLoss,
	); err != nil {
		rm.Log().Error("Failed to publish consecutive losses warning", "error", err)
	} else {
		rm.Log().Warn("Consecutive losses warning sent",
			"user_id", userID,
			"consecutive_losses", state.ConsecutiveLosses,
			"max_allowed", state.MaxConsecutiveLoss,
		)
	}
}
