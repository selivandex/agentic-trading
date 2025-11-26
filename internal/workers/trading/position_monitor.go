package trading

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/position"
	"prometheus/internal/domain/user"
	"prometheus/internal/workers"
	"prometheus/pkg/crypto"
	"prometheus/pkg/errors"
)

// PositionMonitor monitors all open positions and updates their PnL
type PositionMonitor struct {
	*workers.BaseWorker
	userRepo    user.Repository
	posRepo     position.Repository
	accountRepo exchange_account.Repository
	exchFactory exchanges.Factory
	encryptor   crypto.Encryptor
	kafka       *kafka.Producer
}

// NewPositionMonitor creates a new position monitor worker
func NewPositionMonitor(
	userRepo user.Repository,
	posRepo position.Repository,
	accountRepo exchange_account.Repository,
	exchFactory exchanges.Factory,
	encryptor crypto.Encryptor,
	kafka *kafka.Producer,
	interval time.Duration,
	enabled bool,
) *PositionMonitor {
	return &PositionMonitor{
		BaseWorker:  workers.NewBaseWorker("position_monitor", interval, enabled),
		userRepo:    userRepo,
		posRepo:     posRepo,
		accountRepo: accountRepo,
		exchFactory: exchFactory,
		encryptor:   encryptor,
		kafka:       kafka,
	}
}

// Run executes one iteration of position monitoring
func (pm *PositionMonitor) Run(ctx context.Context) error {
	pm.Log().Debug("Position monitor: starting iteration")

	// Get all active users (using large limit)
	// In production, you'd want pagination or stream processing for large user bases
	users, err := pm.userRepo.List(ctx, 1000, 0)
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
		pm.Log().Debug("No active users to monitor")
		return nil
	}

	pm.Log().Debug("Monitoring positions for users", "user_count", len(activeUsers))

	// Monitor positions for each user
	successCount := 0
	errorCount := 0
	for _, usr := range activeUsers {
		if err := pm.monitorUserPositions(ctx, usr.ID); err != nil {
			pm.Log().Error("Failed to monitor user positions",
				"user_id", usr.ID,
				"error", err,
			)
			errorCount++
			// Continue with other users
			continue
		}
		successCount++
	}

	pm.Log().Info("Position monitor: iteration complete",
		"users_processed", successCount,
		"errors", errorCount,
	)

	return nil
}

// monitorUserPositions monitors positions for a specific user
func (pm *PositionMonitor) monitorUserPositions(ctx context.Context, userID uuid.UUID) error {
	// Get all open positions for this user
	positions, err := pm.posRepo.GetOpenByUser(ctx, userID)
	if err != nil {
		return errors.Wrapf(err, "failed to get open positions for user %s", userID)
	}

	if len(positions) == 0 {
		return nil // No positions to monitor
	}

	pm.Log().Debug("Monitoring positions", "user_id", userID, "count", len(positions))

	// Group positions by exchange account
	positionsByAccount := make(map[uuid.UUID][]*position.Position)
	for _, pos := range positions {
		positionsByAccount[pos.ExchangeAccountID] = append(positionsByAccount[pos.ExchangeAccountID], pos)
	}

	// Process each exchange account
	for accountID, accountPositions := range positionsByAccount {
		if err := pm.monitorAccountPositions(ctx, accountID, accountPositions); err != nil {
			pm.Log().Error("Failed to monitor positions for account",
				"account_id", accountID,
				"error", err,
			)
			// Continue with other accounts even if one fails
		}
	}

	return nil
}

// monitorAccountPositions monitors positions for a specific exchange account
func (pm *PositionMonitor) monitorAccountPositions(ctx context.Context, accountID uuid.UUID, positions []*position.Position) error {
	// Get exchange account to get credentials
	account, err := pm.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return errors.Wrap(err, "failed to get exchange account")
	}

	// Create exchange client (factory handles decryption internally)
	exchangeClient, err := pm.exchFactory.CreateClient(account, &pm.encryptor)
	if err != nil {
		return errors.Wrap(err, "failed to create exchange client")
	}

	// Update each position
	for _, pos := range positions {
		if err := pm.updatePosition(ctx, exchangeClient, pos); err != nil {
			pm.Log().Error("Failed to update position",
				"position_id", pos.ID,
				"symbol", pos.Symbol,
				"error", err,
			)
			// Continue with other positions
			continue
		}
	}

	return nil
}

// updatePosition updates a single position with current price and PnL
func (pm *PositionMonitor) updatePosition(ctx context.Context, exchange exchanges.Exchange, pos *position.Position) error {
	// Get current price from exchange
	ticker, err := exchange.GetTicker(ctx, pos.Symbol)
	if err != nil {
		return errors.Wrapf(err, "failed to get ticker for %s", pos.Symbol)
	}

	currentPrice := ticker.LastPrice

	// Calculate unrealized PnL
	var unrealizedPnL decimal.Decimal
	var unrealizedPnLPct decimal.Decimal

	if pos.Side == position.PositionLong {
		// Long: PnL = (current_price - entry_price) * size
		unrealizedPnL = currentPrice.Sub(pos.EntryPrice).Mul(pos.Size)
	} else {
		// Short: PnL = (entry_price - current_price) * size
		unrealizedPnL = pos.EntryPrice.Sub(currentPrice).Mul(pos.Size)
	}

	// Calculate PnL percentage
	positionValue := pos.EntryPrice.Mul(pos.Size)
	if !positionValue.IsZero() {
		unrealizedPnLPct = unrealizedPnL.Div(positionValue).Mul(decimal.NewFromInt(100))
	}

	// Update position in database
	if err := pm.posRepo.UpdatePnL(ctx, pos.ID, currentPrice, unrealizedPnL, unrealizedPnLPct); err != nil {
		return errors.Wrap(err, "failed to update position PnL")
	}

	// Check if stop loss was hit
	if !pos.StopLossPrice.IsZero() {
		slHit := false
		if pos.Side == position.PositionLong {
			slHit = currentPrice.LessThanOrEqual(pos.StopLossPrice)
		} else {
			slHit = currentPrice.GreaterThanOrEqual(pos.StopLossPrice)
		}

		if slHit {
			pm.sendStopLossEvent(ctx, pos, currentPrice)
		}
	}

	// Check if take profit was hit
	if !pos.TakeProfitPrice.IsZero() {
		tpHit := false
		if pos.Side == position.PositionLong {
			tpHit = currentPrice.GreaterThanOrEqual(pos.TakeProfitPrice)
		} else {
			tpHit = currentPrice.LessThanOrEqual(pos.TakeProfitPrice)
		}

		if tpHit {
			pm.sendTakeProfitEvent(ctx, pos, currentPrice)
		}
	}

	// Send PnL update event
	pm.sendPnLUpdateEvent(ctx, pos, currentPrice, unrealizedPnL, unrealizedPnLPct)

	pm.Log().Debug("Position updated",
		"position_id", pos.ID,
		"symbol", pos.Symbol,
		"current_price", currentPrice,
		"unrealized_pnl", unrealizedPnL,
		"unrealized_pnl_pct", unrealizedPnLPct,
	)

	return nil
}

// Event structures

type StopLossEvent struct {
	PositionID    string    `json:"position_id"`
	UserID        string    `json:"user_id"`
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"`
	EntryPrice    string    `json:"entry_price"`
	StopLossPrice string    `json:"stop_loss_price"`
	CurrentPrice  string    `json:"current_price"`
	Size          string    `json:"size"`
	Timestamp     time.Time `json:"timestamp"`
}

type TakeProfitEvent struct {
	PositionID      string    `json:"position_id"`
	UserID          string    `json:"user_id"`
	Symbol          string    `json:"symbol"`
	Side            string    `json:"side"`
	EntryPrice      string    `json:"entry_price"`
	TakeProfitPrice string    `json:"take_profit_price"`
	CurrentPrice    string    `json:"current_price"`
	Size            string    `json:"size"`
	Timestamp       time.Time `json:"timestamp"`
}

type PnLUpdateEvent struct {
	PositionID       string    `json:"position_id"`
	UserID           string    `json:"user_id"`
	Symbol           string    `json:"symbol"`
	CurrentPrice     string    `json:"current_price"`
	UnrealizedPnL    string    `json:"unrealized_pnl"`
	UnrealizedPnLPct string    `json:"unrealized_pnl_pct"`
	Timestamp        time.Time `json:"timestamp"`
}

func (pm *PositionMonitor) sendStopLossEvent(ctx context.Context, pos *position.Position, currentPrice decimal.Decimal) {
	event := StopLossEvent{
		PositionID:    pos.ID.String(),
		UserID:        pos.UserID.String(),
		Symbol:        pos.Symbol,
		Side:          pos.Side.String(),
		EntryPrice:    pos.EntryPrice.String(),
		StopLossPrice: pos.StopLossPrice.String(),
		CurrentPrice:  currentPrice.String(),
		Size:          pos.Size.String(),
		Timestamp:     time.Now(),
	}

	if err := pm.kafka.Publish(ctx, "position.sl_hit", event.PositionID, event); err != nil {
		pm.Log().Error("Failed to publish stop loss event", "error", err)
	} else {
		pm.Log().Info("Stop loss hit",
			"position_id", pos.ID,
			"symbol", pos.Symbol,
			"stop_loss_price", pos.StopLossPrice,
			"current_price", currentPrice,
		)
	}
}

func (pm *PositionMonitor) sendTakeProfitEvent(ctx context.Context, pos *position.Position, currentPrice decimal.Decimal) {
	event := TakeProfitEvent{
		PositionID:      pos.ID.String(),
		UserID:          pos.UserID.String(),
		Symbol:          pos.Symbol,
		Side:            pos.Side.String(),
		EntryPrice:      pos.EntryPrice.String(),
		TakeProfitPrice: pos.TakeProfitPrice.String(),
		CurrentPrice:    currentPrice.String(),
		Size:            pos.Size.String(),
		Timestamp:       time.Now(),
	}

	if err := pm.kafka.Publish(ctx, "position.tp_hit", event.PositionID, event); err != nil {
		pm.Log().Error("Failed to publish take profit event", "error", err)
	} else {
		pm.Log().Info("Take profit hit",
			"position_id", pos.ID,
			"symbol", pos.Symbol,
			"take_profit_price", pos.TakeProfitPrice,
			"current_price", currentPrice,
		)
	}
}

func (pm *PositionMonitor) sendPnLUpdateEvent(ctx context.Context, pos *position.Position, currentPrice, unrealizedPnL, unrealizedPnLPct decimal.Decimal) {
	event := PnLUpdateEvent{
		PositionID:       pos.ID.String(),
		UserID:           pos.UserID.String(),
		Symbol:           pos.Symbol,
		CurrentPrice:     currentPrice.String(),
		UnrealizedPnL:    unrealizedPnL.String(),
		UnrealizedPnLPct: unrealizedPnLPct.String(),
		Timestamp:        time.Now(),
	}

	// Only publish significant PnL updates (to avoid spamming)
	// For example, only publish if PnL changed by more than 1%
	if unrealizedPnLPct.Abs().GreaterThan(decimal.NewFromFloat(1.0)) {
		if err := pm.kafka.Publish(ctx, "position.pnl_updated", event.PositionID, event); err != nil {
			pm.Log().Error("Failed to publish PnL update event", "error", err)
		}
	}
}
