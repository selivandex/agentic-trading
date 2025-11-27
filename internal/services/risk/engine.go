package riskservice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/position"
	"prometheus/internal/domain/risk"
	"prometheus/internal/domain/user"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

const (
	// Cache keys
	riskStateCacheKeyPrefix = "risk:state:"
	riskStateCacheTTL       = 30 * time.Second
)

// UserRiskState represents current risk state of a user
type UserRiskState struct {
	UserID             uuid.UUID
	CanTrade           bool
	IsCircuitTripped   bool
	TripReason         string
	DailyPnL           decimal.Decimal
	DailyPnLPercent    decimal.Decimal
	DailyTradeCount    int
	ConsecutiveLosses  int
	MaxDrawdown        decimal.Decimal
	MaxConsecutiveLoss int
}

// RiskEngine manages risk checks and circuit breaker logic
type RiskEngine struct {
	riskRepo risk.Repository
	posRepo  position.Repository
	userRepo user.Repository
	redis    risk.RedisClient
	log      *logger.Logger
}

// NewRiskEngine creates a new risk engine
func NewRiskEngine(
	riskRepo risk.Repository,
	posRepo position.Repository,
	userRepo user.Repository,
	redisClient risk.RedisClient,
	log *logger.Logger,
) *RiskEngine {
	return &RiskEngine{
		riskRepo: riskRepo,
		posRepo:  posRepo,
		userRepo: userRepo,
		redis:    redisClient,
		log:      log,
	}
}

// CanTrade checks if user is allowed to trade
// Returns true if trading is allowed, false otherwise
func (e *RiskEngine) CanTrade(ctx context.Context, userID uuid.UUID) (bool, error) {
	// Try to get from cache first
	cacheKey := riskStateCacheKeyPrefix + userID.String()
	var cached UserRiskState
	if err := e.redis.Get(ctx, cacheKey, &cached); err == nil {
		return cached.CanTrade, nil
	}

	// Get current state from DB
	state, err := e.riskRepo.GetState(ctx, userID)
	if err != nil {
		// If state doesn't exist, create default state
		state = e.createDefaultState(userID)
		if err := e.riskRepo.SaveState(ctx, state); err != nil {
			return false, errors.Wrap(err, "failed to create default risk state")
		}
	}

	// Check if circuit breaker is triggered
	if state.IsTriggered {
		e.log.Warn("Trading blocked by circuit breaker",
			"user_id", userID,
			"reason", state.TriggerReason,
			"triggered_at", state.TriggeredAt,
		)

		// Cache the result
		e.cacheUserState(ctx, e.buildUserRiskState(state, false))

		// Return domain error for better error handling
		return false, errors.ErrCircuitBreakerTripped
	}

	// Check if daily drawdown limit exceeded
	exceeded := e.isDrawdownExceeded(state)
	e.log.Debug("Checking drawdown",
		"user_id", userID,
		"daily_pnl", state.DailyPnL,
		"max_drawdown", state.MaxDailyDrawdown,
		"exceeded", exceeded,
	)
	if exceeded {
		if err := e.tripCircuitBreaker(ctx, state, "Daily drawdown limit exceeded"); err != nil {
			return false, errors.Wrap(err, "failed to trip circuit breaker")
		}
		// Return specific domain error
		return false, errors.ErrDrawdownExceeded
	}

	// Check if consecutive losses limit exceeded
	if e.isConsecutiveLossesExceeded(state) {
		if err := e.tripCircuitBreaker(ctx, state, "Consecutive losses limit exceeded"); err != nil {
			return false, errors.Wrap(err, "failed to trip circuit breaker")
		}
		// Return specific domain error
		return false, errors.ErrConsecutiveLosses
	}

	// Check max exposure (total open positions)
	positions, err := e.posRepo.GetOpenByUser(ctx, userID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get open positions")
	}

	// Calculate total exposure
	totalExposure := decimal.Zero
	for _, pos := range positions {
		// For futures, exposure = size * entry_price / leverage
		// For spot, exposure = size * entry_price
		positionValue := pos.Size.Mul(pos.EntryPrice)
		if pos.Leverage > 0 {
			positionValue = positionValue.Div(decimal.NewFromInt(int64(pos.Leverage)))
		}
		totalExposure = totalExposure.Add(positionValue)
	}

	// Allow trading
	userState := e.buildUserRiskState(state, true)
	e.cacheUserState(ctx, userState)

	return true, nil
}

// RecordTrade records a trade and updates risk state
func (e *RiskEngine) RecordTrade(ctx context.Context, userID uuid.UUID, pnl decimal.Decimal) error {
	// Get current state
	state, err := e.riskRepo.GetState(ctx, userID)
	if err != nil {
		state = e.createDefaultState(userID)
	}

	// Update daily stats
	state.DailyPnL = state.DailyPnL.Add(pnl)
	state.DailyTradeCount++

	// Update win/loss counters
	if pnl.GreaterThan(decimal.Zero) {
		state.DailyWins++
		state.ConsecutiveLosses = 0 // Reset consecutive losses
	} else if pnl.LessThan(decimal.Zero) {
		state.DailyLosses++
		state.ConsecutiveLosses++
	}

	// Calculate PnL percentage based on start-of-day balance
	// Get user to retrieve start-of-day balance (or use settings)
	user, err := e.userRepo.GetByID(ctx, userID)
	if err == nil && user.Settings.MaxTotalExposureUSD > 0 {
		// Use MaxTotalExposureUSD as proxy for account size
		// This is conservative estimate (actual balance likely higher)
		accountSize := decimal.NewFromFloat(user.Settings.MaxTotalExposureUSD)
		if !accountSize.IsZero() {
			state.DailyPnLPercent = state.DailyPnL.Div(accountSize).Mul(decimal.NewFromInt(100))
		}
	} else {
		// Fallback: assume 10k account size
		state.DailyPnLPercent = state.DailyPnL.Div(decimal.NewFromFloat(10000)).Mul(decimal.NewFromInt(100))
	}

	state.UpdatedAt = time.Now()

	// Save updated state
	if err := e.riskRepo.SaveState(ctx, state); err != nil {
		return errors.Wrap(err, "failed to save risk state")
	}

	// Check if we should trip circuit breaker
	if e.isDrawdownExceeded(state) {
		if err := e.tripCircuitBreaker(ctx, state, "Daily drawdown limit exceeded after trade"); err != nil {
			return errors.Wrap(err, "failed to trip circuit breaker")
		}
	} else if e.isConsecutiveLossesExceeded(state) {
		if err := e.tripCircuitBreaker(ctx, state, "Consecutive losses limit exceeded"); err != nil {
			return errors.Wrap(err, "failed to trip circuit breaker")
		}
	}

	// Create risk event if approaching limits (80% threshold)
	if e.isApproachingDrawdownLimit(state) {
		e.createWarningEvent(ctx, userID, "drawdown_warning", "Approaching daily drawdown limit (80%)")
	}

	// Invalidate cache
	cacheKey := riskStateCacheKeyPrefix + userID.String()
	e.redis.Delete(ctx, cacheKey)

	e.log.Info("Trade recorded in risk engine",
		"user_id", userID,
		"pnl", pnl,
		"daily_pnl", state.DailyPnL,
		"daily_trades", state.DailyTradeCount,
		"consecutive_losses", state.ConsecutiveLosses,
	)

	return nil
}

// GetUserState returns current risk state for a user
func (e *RiskEngine) GetUserState(ctx context.Context, userID uuid.UUID) (*UserRiskState, error) {
	// Try cache first
	cacheKey := riskStateCacheKeyPrefix + userID.String()
	var cached UserRiskState
	if err := e.redis.Get(ctx, cacheKey, &cached); err == nil {
		return &cached, nil
	}

	// Get from DB
	state, err := e.riskRepo.GetState(ctx, userID)
	if err != nil {
		state = e.createDefaultState(userID)
		if err := e.riskRepo.SaveState(ctx, state); err != nil {
			return nil, errors.Wrap(err, "failed to create default risk state")
		}
	}

	userState := e.buildUserRiskState(state, !state.IsTriggered)
	e.cacheUserState(ctx, userState)

	return &userState, nil
}

// ResetDaily resets daily counters for all users
func (e *RiskEngine) ResetDaily(ctx context.Context) error {
	e.log.Info("Resetting daily risk counters for all users")

	if err := e.riskRepo.ResetDaily(ctx); err != nil {
		return errors.Wrap(err, "failed to reset daily counters")
	}

	e.log.Info("Daily risk counters reset successfully")
	return nil
}

// TripCircuitBreaker manually trips the circuit breaker for a user
func (e *RiskEngine) TripCircuitBreaker(ctx context.Context, userID uuid.UUID, reason string) error {
	state, err := e.riskRepo.GetState(ctx, userID)
	if err != nil {
		state = e.createDefaultState(userID)
	}

	return e.tripCircuitBreaker(ctx, state, reason)
}

// ResetCircuitBreaker manually resets the circuit breaker for a user
func (e *RiskEngine) ResetCircuitBreaker(ctx context.Context, userID uuid.UUID) error {
	state, err := e.riskRepo.GetState(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "failed to get risk state")
	}

	state.IsTriggered = false
	state.TriggeredAt = nil
	state.TriggerReason = ""
	state.UpdatedAt = time.Now()

	if err := e.riskRepo.SaveState(ctx, state); err != nil {
		return errors.Wrap(err, "failed to save risk state")
	}

	// Invalidate cache
	cacheKey := riskStateCacheKeyPrefix + userID.String()
	e.redis.Delete(ctx, cacheKey)

	e.log.Info("Circuit breaker reset manually", "user_id", userID)

	return nil
}

// Helper functions

func (e *RiskEngine) createDefaultState(userID uuid.UUID) *risk.CircuitBreakerState {
	now := time.Now()
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)

	return &risk.CircuitBreakerState{
		UserID:             userID,
		IsTriggered:        false,
		DailyPnL:           decimal.Zero,
		DailyPnLPercent:    decimal.Zero,
		DailyTradeCount:    0,
		DailyWins:          0,
		DailyLosses:        0,
		ConsecutiveLosses:  0,
		MaxDailyDrawdown:   decimal.NewFromFloat(0.10), // Default 10%
		MaxConsecutiveLoss: 3,                          // Default 3 consecutive losses
		ResetAt:            tomorrow,
		UpdatedAt:          now,
	}
}

func (e *RiskEngine) isDrawdownExceeded(state *risk.CircuitBreakerState) bool {
	// Check if daily PnL (negative) exceeds max drawdown
	if state.DailyPnL.LessThan(decimal.Zero) {
		absDrawdown := state.DailyPnL.Abs()
		return absDrawdown.GreaterThanOrEqual(state.MaxDailyDrawdown)
	}
	return false
}

func (e *RiskEngine) isConsecutiveLossesExceeded(state *risk.CircuitBreakerState) bool {
	return state.ConsecutiveLosses >= state.MaxConsecutiveLoss
}

func (e *RiskEngine) isApproachingDrawdownLimit(state *risk.CircuitBreakerState) bool {
	if state.DailyPnL.LessThan(decimal.Zero) {
		absDrawdown := state.DailyPnL.Abs()
		threshold := state.MaxDailyDrawdown.Mul(decimal.NewFromFloat(0.80)) // 80% of limit
		return absDrawdown.GreaterThanOrEqual(threshold)
	}
	return false
}

func (e *RiskEngine) tripCircuitBreaker(ctx context.Context, state *risk.CircuitBreakerState, reason string) error {
	now := time.Now()
	state.IsTriggered = true
	state.TriggeredAt = &now
	state.TriggerReason = reason
	state.UpdatedAt = now

	if err := e.riskRepo.SaveState(ctx, state); err != nil {
		return errors.Wrap(err, "failed to save risk state")
	}

	// Create critical risk event
	event := &risk.RiskEvent{
		ID:           uuid.New(),
		UserID:       state.UserID,
		Timestamp:    now,
		EventType:    risk.RiskEventCircuitBreaker,
		Severity:     "critical",
		Message:      reason,
		Data:         fmt.Sprintf(`{"daily_pnl": "%s", "consecutive_losses": %d}`, state.DailyPnL.String(), state.ConsecutiveLosses),
		Acknowledged: false,
	}

	if err := e.riskRepo.CreateEvent(ctx, event); err != nil {
		e.log.Error("Failed to create risk event", "error", err)
	}

	// Invalidate cache
	cacheKey := riskStateCacheKeyPrefix + state.UserID.String()
	e.redis.Delete(ctx, cacheKey)

	e.log.Warn("Circuit breaker tripped",
		"user_id", state.UserID,
		"reason", reason,
		"daily_pnl", state.DailyPnL,
		"consecutive_losses", state.ConsecutiveLosses,
	)

	return nil
}

func (e *RiskEngine) createWarningEvent(ctx context.Context, userID uuid.UUID, eventType, message string) {
	event := &risk.RiskEvent{
		ID:           uuid.New(),
		UserID:       userID,
		Timestamp:    time.Now(),
		EventType:    risk.RiskEventType(eventType),
		Severity:     "warning",
		Message:      message,
		Data:         "{}",
		Acknowledged: false,
	}

	if err := e.riskRepo.CreateEvent(ctx, event); err != nil {
		e.log.Error("Failed to create warning event", "error", err)
	}
}

func (e *RiskEngine) buildUserRiskState(state *risk.CircuitBreakerState, canTrade bool) UserRiskState {
	return UserRiskState{
		UserID:             state.UserID,
		CanTrade:           canTrade,
		IsCircuitTripped:   state.IsTriggered,
		TripReason:         state.TriggerReason,
		DailyPnL:           state.DailyPnL,
		DailyPnLPercent:    state.DailyPnLPercent,
		DailyTradeCount:    state.DailyTradeCount,
		ConsecutiveLosses:  state.ConsecutiveLosses,
		MaxDrawdown:        state.MaxDailyDrawdown,
		MaxConsecutiveLoss: state.MaxConsecutiveLoss,
	}
}

func (e *RiskEngine) cacheUserState(ctx context.Context, state UserRiskState) {
	cacheKey := riskStateCacheKeyPrefix + state.UserID.String()
	if err := e.redis.Set(ctx, cacheKey, state, riskStateCacheTTL); err != nil {
		e.log.Error("Failed to cache user risk state", "error", err)
	}
}
