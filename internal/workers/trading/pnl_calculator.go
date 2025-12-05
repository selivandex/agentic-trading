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

// PnLCalculator calculates daily PnL for all users and updates risk state
// This worker runs periodically to aggregate closed positions and update risk metrics
type PnLCalculator struct {
	*workers.BaseWorker
	userRepo       user.Repository
	posRepo        position.Repository
	riskEngine     *riskservice.RiskEngine
	kafka          *kafka.Producer
	eventPublisher *events.WorkerPublisher
}

// NewPnLCalculator creates a new PnL calculator worker
func NewPnLCalculator(
	userRepo user.Repository,
	posRepo position.Repository,
	riskEngine *riskservice.RiskEngine,
	kafka *kafka.Producer,
	interval time.Duration,
	enabled bool,
) *PnLCalculator {
	return &PnLCalculator{
		BaseWorker:     workers.NewBaseWorker("pnl_calculator", interval, enabled),
		userRepo:       userRepo,
		posRepo:        posRepo,
		riskEngine:     riskEngine,
		kafka:          kafka,
		eventPublisher: events.NewWorkerPublisher(kafka),
	}
}

// Run executes one iteration of PnL calculation
func (pc *PnLCalculator) Run(ctx context.Context) error {
	pc.Log().Debug("PnL calculator: starting iteration")

	// Get all active users
	users, err := pc.userRepo.List(ctx, 1000, 0)
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
		pc.Log().Debug("No active users to calculate PnL for")
		return nil
	}

	pc.Log().Debugw("Calculating PnL for users", "user_count", len(activeUsers))

	// Calculate PnL for each user
	successCount := 0
	errorCount := 0
	for _, usr := range activeUsers {
		// Check for context cancellation (graceful shutdown)
		select {
		case <-ctx.Done():
		pc.Log().Infow("PnL calculation interrupted by shutdown",
			"users_processed", successCount,
			"users_remaining", len(activeUsers)-successCount-errorCount,
		)
			return ctx.Err()
		default:
		}

		if err := pc.calculateUserPnL(ctx, usr.ID); err != nil {
		pc.Log().Errorw("Failed to calculate user PnL",
			"user_id", usr.ID,
			"error", err,
		)
			errorCount++
			// Continue with other users
			continue
		}
		successCount++
	}

	pc.Log().Infow("PnL calculator: iteration complete",
		"users_processed", successCount,
		"errors", errorCount,
	)

	return nil
}

// calculateUserPnL calculates daily PnL for a specific user
func (pc *PnLCalculator) calculateUserPnL(ctx context.Context, userID uuid.UUID) error {
	// Calculate date range (today 00:00 to now UTC)
	now := time.Now().UTC()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Get closed positions for today
	closedPositions, err := pc.posRepo.GetClosedInRange(ctx, userID, startOfToday, now)
	if err != nil {
		return errors.Wrap(err, "failed to get closed positions")
	}

	// Calculate daily PnL
	dailyPnL := decimal.Zero
	for _, pos := range closedPositions {
		dailyPnL = dailyPnL.Add(pos.RealizedPnL)
	}

	// Get open positions to calculate total exposure and unrealized PnL
	openPositions, err := pc.posRepo.GetOpenByUser(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "failed to get open positions")
	}

	totalUnrealizedPnL := decimal.Zero
	totalExposure := decimal.Zero
	for _, pos := range openPositions {
		totalUnrealizedPnL = totalUnrealizedPnL.Add(pos.UnrealizedPnL)

		// Calculate position value (size * entry price / leverage)
		positionValue := pos.Size.Mul(pos.EntryPrice)
		if pos.Leverage > 0 {
			positionValue = positionValue.Div(decimal.NewFromInt(int64(pos.Leverage)))
		}
		totalExposure = totalExposure.Add(positionValue)
	}

	// Total PnL = realized + unrealized
	totalPnL := dailyPnL.Add(totalUnrealizedPnL)

	pc.Log().Debugw("User PnL calculated",
		"user_id", userID,
		"daily_realized_pnl", dailyPnL,
		"total_unrealized_pnl", totalUnrealizedPnL,
		"total_pnl", totalPnL,
		"total_exposure", totalExposure,
		"closed_positions", len(closedPositions),
		"open_positions", len(openPositions),
	)

	// Update risk engine state (this will check circuit breakers)
	// Note: This is a simplified version - in production you'd want to:
	// 1. Track consecutive losses properly
	// 2. Calculate daily PnL percentage based on account balance
	// 3. Handle win/loss streak detection
	// For now, we'll just send PnL update event
	pc.sendPnLUpdateEvent(ctx, userID, dailyPnL, totalUnrealizedPnL, totalPnL, len(closedPositions))

	return nil
}

// Event structures

type UserPnLUpdateEvent struct {
	UserID              string    `json:"user_id"`
	DailyRealizedPnL    string    `json:"daily_realized_pnl"`
	TotalUnrealizedPnL  string    `json:"total_unrealized_pnl"`
	TotalPnL            string    `json:"total_pnl"`
	ClosedPositionCount int       `json:"closed_position_count"`
	Timestamp           time.Time `json:"timestamp"`
}

func (pc *PnLCalculator) sendPnLUpdateEvent(
	ctx context.Context,
	userID uuid.UUID,
	dailyRealizedPnL, totalUnrealizedPnL, totalPnL decimal.Decimal,
	closedPositionCount int,
) {
	// Only publish if there's significant activity (at least one closed position today)
	if closedPositionCount > 0 || !totalPnL.IsZero() {
		dailyPnLFloat, _ := dailyRealizedPnL.Float64()
		totalPnLFloat, _ := totalPnL.Float64()

		if err := pc.eventPublisher.PublishPnLUpdated(
			ctx,
			userID.String(),
			dailyPnLFloat,
			0.0, // daily_pnl_percent - TODO: calculate
			totalPnLFloat,
			closedPositionCount,
			0,   // winning_trades - TODO: calculate
			0,   // losing_trades - TODO: calculate
			0.0, // win_rate - TODO: calculate
		); err != nil {
			pc.Log().Errorw("Failed to publish PnL update event", "error", err)
		}
	}
}
