package risk

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/order"
	"prometheus/internal/domain/position"
	domainRisk "prometheus/internal/domain/risk"
	"prometheus/pkg/logger"
)

const (
	killSwitchKeyPrefix = "killswitch:"
	killSwitchTTL       = 24 * time.Hour
)

// KillSwitchResult contains the results of a kill switch activation
type KillSwitchResult struct {
	UserID            uuid.UUID
	PositionsClosed   int
	OrdersCancelled   int
	Errors            []string
	CircuitBreakerSet bool
	ActivatedAt       time.Time
}

// KillSwitch handles emergency shutdown of all trading activities for a user
type KillSwitch struct {
	posRepo   position.Repository
	orderRepo order.Repository
	riskRepo  domainRisk.Repository
	redis     RedisClient
	log       *logger.Logger
}

// NewKillSwitch creates a new kill switch handler
func NewKillSwitch(
	posRepo position.Repository,
	orderRepo order.Repository,
	riskRepo domainRisk.Repository,
	redis RedisClient,
	log *logger.Logger,
) *KillSwitch {
	return &KillSwitch{
		posRepo:   posRepo,
		orderRepo: orderRepo,
		riskRepo:  riskRepo,
		redis:     redis,
		log:       log,
	}
}

// Activate triggers the kill switch for a user
// This will:
// 1. Get all open positions
// 2. Close all positions with market orders
// 3. Cancel all open orders
// 4. Set circuit breaker to BLOCKED state
// 5. Store kill switch state in Redis
func (k *KillSwitch) Activate(ctx context.Context, userID uuid.UUID, reason string) (*KillSwitchResult, error) {
	k.log.Warn("Kill switch activated",
		"user_id", userID,
		"reason", reason,
	)

	result := &KillSwitchResult{
		UserID:      userID,
		ActivatedAt: time.Now(),
		Errors:      make([]string, 0),
	}

	// 1. Close all open positions
	positions, err := k.posRepo.GetOpenByUser(ctx, userID)
	if err != nil {
		errMsg := fmt.Sprintf("failed to fetch open positions: %v", err)
		result.Errors = append(result.Errors, errMsg)
		k.log.Error("Kill switch: failed to fetch positions", "error", err, "user_id", userID)
	} else {
		for _, pos := range positions {
			// Use current price or entry price as fallback
			closePrice := pos.CurrentPrice
			if closePrice.IsZero() {
				closePrice = pos.EntryPrice
			}

			// Calculate realized PnL
			var realizedPnL decimal.Decimal
			if pos.Side == position.PositionLong {
				// Long: PnL = (exit_price - entry_price) * size
				realizedPnL = closePrice.Sub(pos.EntryPrice).Mul(pos.Size)
			} else {
				// Short: PnL = (entry_price - exit_price) * size
				realizedPnL = pos.EntryPrice.Sub(closePrice).Mul(pos.Size)
			}

			// Close position
			if err := k.posRepo.Close(ctx, pos.ID, closePrice, realizedPnL); err != nil {
				errMsg := fmt.Sprintf("failed to close position %s: %v", pos.ID, err)
				result.Errors = append(result.Errors, errMsg)
				k.log.Error("Kill switch: failed to close position",
					"error", err,
					"position_id", pos.ID,
					"symbol", pos.Symbol,
				)
			} else {
				result.PositionsClosed++
				k.log.Info("Kill switch: position closed",
					"position_id", pos.ID,
					"symbol", pos.Symbol,
					"realized_pnl", realizedPnL,
				)
			}
		}
	}

	// 2. Cancel all open orders
	orders, err := k.orderRepo.GetOpenByUser(ctx, userID)
	if err != nil {
		errMsg := fmt.Sprintf("failed to fetch open orders: %v", err)
		result.Errors = append(result.Errors, errMsg)
		k.log.Error("Kill switch: failed to fetch orders", "error", err, "user_id", userID)
	} else {
		for _, ord := range orders {
			if err := k.orderRepo.Cancel(ctx, ord.ID); err != nil {
				errMsg := fmt.Sprintf("failed to cancel order %s: %v", ord.ID, err)
				result.Errors = append(result.Errors, errMsg)
				k.log.Error("Kill switch: failed to cancel order",
					"error", err,
					"order_id", ord.ID,
					"symbol", ord.Symbol,
				)
			} else {
				result.OrdersCancelled++
				k.log.Info("Kill switch: order cancelled",
					"order_id", ord.ID,
					"symbol", ord.Symbol,
				)
			}
		}
	}

	// 3. Set circuit breaker to BLOCKED state
	state, err := k.riskRepo.GetState(ctx, userID)
	if err != nil {
		// Create new state if doesn't exist
		state = k.createDefaultState(userID)
	}

	now := time.Now()
	state.IsTriggered = true
	state.TriggeredAt = &now
	state.TriggerReason = fmt.Sprintf("Kill switch activated: %s", reason)
	state.UpdatedAt = now

	if err := k.riskRepo.SaveState(ctx, state); err != nil {
		errMsg := fmt.Sprintf("failed to set circuit breaker: %v", err)
		result.Errors = append(result.Errors, errMsg)
		k.log.Error("Kill switch: failed to save circuit breaker state", "error", err)
	} else {
		result.CircuitBreakerSet = true
	}

	// 4. Create critical risk event
	event := &domainRisk.RiskEvent{
		ID:           uuid.New(),
		UserID:       userID,
		Timestamp:    now,
		EventType:    domainRisk.RiskEventKillSwitch,
		Severity:     "critical",
		Message:      fmt.Sprintf("Kill switch activated: %s. Closed %d positions, cancelled %d orders", reason, result.PositionsClosed, result.OrdersCancelled),
		Data:         fmt.Sprintf(`{"positions_closed": %d, "orders_cancelled": %d, "reason": "%s"}`, result.PositionsClosed, result.OrdersCancelled, reason),
		Acknowledged: false,
	}

	if err := k.riskRepo.CreateEvent(ctx, event); err != nil {
		k.log.Error("Kill switch: failed to create risk event", "error", err)
	}

	// 5. Store kill switch state in Redis
	killSwitchKey := killSwitchKeyPrefix + userID.String()
	killSwitchData := map[string]interface{}{
		"activated_at":     result.ActivatedAt,
		"reason":           reason,
		"positions_closed": result.PositionsClosed,
		"orders_cancelled": result.OrdersCancelled,
		"circuit_breaker":  result.CircuitBreakerSet,
	}

	if err := k.redis.Set(ctx, killSwitchKey, killSwitchData, killSwitchTTL); err != nil {
		k.log.Error("Kill switch: failed to store state in Redis", "error", err)
	}

	k.log.Info("Kill switch activation complete",
		"user_id", userID,
		"positions_closed", result.PositionsClosed,
		"orders_cancelled", result.OrdersCancelled,
		"errors", len(result.Errors),
	)

	return result, nil
}

// IsActive checks if kill switch is active for a user
func (k *KillSwitch) IsActive(ctx context.Context, userID uuid.UUID) (bool, error) {
	killSwitchKey := killSwitchKeyPrefix + userID.String()
	exists, err := k.redis.Exists(ctx, killSwitchKey)
	if err != nil {
		return false, fmt.Errorf("failed to check kill switch state: %w", err)
	}
	return exists, nil
}

// Deactivate manually deactivates the kill switch and resets circuit breaker
func (k *KillSwitch) Deactivate(ctx context.Context, userID uuid.UUID) error {
	k.log.Info("Deactivating kill switch", "user_id", userID)

	// Reset circuit breaker
	state, err := k.riskRepo.GetState(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get risk state: %w", err)
	}

	state.IsTriggered = false
	state.TriggeredAt = nil
	state.TriggerReason = ""
	state.UpdatedAt = time.Now()

	if err := k.riskRepo.SaveState(ctx, state); err != nil {
		return fmt.Errorf("failed to reset circuit breaker: %w", err)
	}

	// Remove Redis flag
	killSwitchKey := killSwitchKeyPrefix + userID.String()
	if err := k.redis.Delete(ctx, killSwitchKey); err != nil {
		k.log.Error("Failed to delete kill switch state from Redis", "error", err)
	}

	k.log.Info("Kill switch deactivated", "user_id", userID)
	return nil
}

// Helper functions

func (k *KillSwitch) createDefaultState(userID uuid.UUID) *domainRisk.CircuitBreakerState {
	now := time.Now()
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)

	return &domainRisk.CircuitBreakerState{
		UserID:             userID,
		IsTriggered:        false,
		DailyPnL:           decimal.Zero,
		DailyPnLPercent:    decimal.Zero,
		DailyTradeCount:    0,
		DailyWins:          0,
		DailyLosses:        0,
		ConsecutiveLosses:  0,
		MaxDailyDrawdown:   decimal.NewFromFloat(0.10),
		MaxConsecutiveLoss: 3,
		ResetAt:            tomorrow,
		UpdatedAt:          now,
	}
}
