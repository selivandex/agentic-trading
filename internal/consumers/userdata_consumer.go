package consumers

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"

	kafkaadapter "prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/order"
	"prometheus/internal/domain/position"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// UserDataConsumer consumes User Data events from Kafka and updates the database
// Handles order updates, position updates, balance updates, and margin calls
type UserDataConsumer struct {
	consumer        *kafkaadapter.Consumer
	orderRepo       order.Repository
	positionRepo    position.Repository
	logger          *logger.Logger
	processingCount int64
}

// NewUserDataConsumer creates a new User Data event consumer
func NewUserDataConsumer(
	consumer *kafkaadapter.Consumer,
	orderRepo order.Repository,
	positionRepo position.Repository,
	log *logger.Logger,
) *UserDataConsumer {
	return &UserDataConsumer{
		consumer:     consumer,
		orderRepo:    orderRepo,
		positionRepo: positionRepo,
		logger:       log,
	}
}

// Start begins consuming User Data events
func (c *UserDataConsumer) Start(ctx context.Context) error {
	c.logger.Info("Starting User Data Consumer")
	return c.consumer.Consume(ctx, c.handleMessage)
}

// handleMessage routes Kafka messages to appropriate handlers
func (c *UserDataConsumer) handleMessage(ctx context.Context, msg kafka.Message) error {
	c.processingCount++

	// Route based on topic
	switch msg.Topic {
	case "user-data-order-updates":
		return c.handleOrderUpdate(ctx, msg.Value)
	case "user-data-position-updates":
		return c.handlePositionUpdate(ctx, msg.Value)
	case "user-data-balance-updates":
		return c.handleBalanceUpdate(ctx, msg.Value)
	case "user-data-margin-calls":
		return c.handleMarginCall(ctx, msg.Value)
	case "user-data-account-config":
		return c.handleAccountConfigUpdate(ctx, msg.Value)
	default:
		c.logger.Warn("Unknown topic", "topic", msg.Topic)
		return nil
	}
}

// handleOrderUpdate processes order update events
func (c *UserDataConsumer) handleOrderUpdate(ctx context.Context, data []byte) error {
	var event eventspb.UserDataOrderUpdateEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "failed to unmarshal order update event")
	}

	userID, err := uuid.Parse(event.Base.UserId)
	if err != nil {
		return errors.Wrap(err, "invalid user_id")
	}

	accountID, err := uuid.Parse(event.AccountId)
	if err != nil {
		return errors.Wrap(err, "invalid account_id")
	}

	c.logger.Info("Processing order update",
		"user_id", userID,
		"account_id", accountID,
		"order_id", event.OrderId,
		"symbol", event.Symbol,
		"status", event.Status,
	)

	// For now, just log the event
	// TODO: Implement full order creation/update logic
	// This requires proper order repository methods

	return nil
}

// handlePositionUpdate processes position update events
func (c *UserDataConsumer) handlePositionUpdate(ctx context.Context, data []byte) error {
	var event eventspb.UserDataPositionUpdateEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "failed to unmarshal position update event")
	}

	userID, err := uuid.Parse(event.Base.UserId)
	if err != nil {
		return errors.Wrap(err, "invalid user_id")
	}

	accountID, err := uuid.Parse(event.AccountId)
	if err != nil {
		return errors.Wrap(err, "invalid account_id")
	}

	c.logger.Info("Processing position update",
		"user_id", userID,
		"account_id", accountID,
		"symbol", event.Symbol,
		"side", event.Side,
		"amount", event.Amount,
	)

	// Parse amount
	amount, err := parseDecimal(event.Amount)
	if err != nil {
		return errors.Wrap(err, "invalid amount")
	}

	// If amount is zero, position is closed
	if amount.IsZero() {
		return c.closePosition(ctx, userID, accountID, event.Symbol)
	}

	// Get existing position
	positions, err := c.positionRepo.GetOpenByUser(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "failed to get positions")
	}

	// Find position for this symbol and account
	var existingPos *position.Position
	for _, pos := range positions {
		if pos.Symbol == event.Symbol && pos.ExchangeAccountID == accountID {
			existingPos = pos
			break
		}
	}

	if existingPos == nil {
		// Create new position
		entryPrice, _ := parseDecimal(event.EntryPrice)
		markPrice, _ := parseDecimal(event.MarkPrice)
		unrealizedPnL, _ := parseDecimal(event.UnrealizedPnl)

		newPos := &position.Position{
			ID:                uuid.New(),
			UserID:            userID,
			ExchangeAccountID: accountID,
			Symbol:            event.Symbol,
			Side:              mapPositionSide(event.Side),
			Size:              amount,
			EntryPrice:        entryPrice,
			CurrentPrice:      markPrice,
			UnrealizedPnL:     unrealizedPnL,
			Status:            position.PositionOpen,
		}

		if err := c.positionRepo.Create(ctx, newPos); err != nil {
			return errors.Wrap(err, "failed to create position")
		}

		c.logger.Info("Created new position",
			"position_id", newPos.ID,
			"symbol", event.Symbol,
		)
	} else {
		// Update existing position
		markPrice, _ := parseDecimal(event.MarkPrice)
		unrealizedPnL, _ := parseDecimal(event.UnrealizedPnl)

		existingPos.Size = amount
		existingPos.CurrentPrice = markPrice
		existingPos.UnrealizedPnL = unrealizedPnL

		if err := c.positionRepo.Update(ctx, existingPos); err != nil {
			return errors.Wrap(err, "failed to update position")
		}

		c.logger.Debug("Updated position",
			"position_id", existingPos.ID,
			"symbol", event.Symbol,
		)
	}

	return nil
}

// handleBalanceUpdate processes balance update events
func (c *UserDataConsumer) handleBalanceUpdate(ctx context.Context, data []byte) error {
	var event eventspb.UserDataBalanceUpdateEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "failed to unmarshal balance update event")
	}

	c.logger.Info("Balance update received",
		"user_id", event.Base.UserId,
		"account_id", event.AccountId,
		"asset", event.Asset,
		"wallet_balance", event.WalletBalance,
		"reason", event.ReasonType,
	)

	// TODO: Store balance updates in a separate table if needed
	return nil
}

// handleMarginCall processes margin call events (CRITICAL!)
func (c *UserDataConsumer) handleMarginCall(ctx context.Context, data []byte) error {
	var event eventspb.UserDataMarginCallEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "failed to unmarshal margin call event")
	}

	c.logger.Error("⚠️ MARGIN CALL RECEIVED",
		"user_id", event.Base.UserId,
		"account_id", event.AccountId,
		"positions_at_risk", len(event.PositionsAtRisk),
		"cross_wallet_balance", event.CrossWalletBalance,
	)

	// TODO: Implement emergency actions:
	// 1. Trigger circuit breaker
	// 2. Send Telegram alert
	// 3. Consider automatic position reduction
	// 4. Log to risk_events table

	return nil
}

// handleAccountConfigUpdate processes account config update events
func (c *UserDataConsumer) handleAccountConfigUpdate(ctx context.Context, data []byte) error {
	var event eventspb.UserDataAccountConfigEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "failed to unmarshal account config update event")
	}

	c.logger.Info("Account config updated",
		"user_id", event.Base.UserId,
		"account_id", event.AccountId,
		"symbol", event.Symbol,
		"leverage", event.Leverage,
	)

	// TODO: Store leverage changes in trading_pairs or positions table
	return nil
}

// closePosition closes an open position
func (c *UserDataConsumer) closePosition(ctx context.Context, userID, accountID uuid.UUID, symbol string) error {
	positions, err := c.positionRepo.GetOpenByUser(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "failed to get positions")
	}

	// Find position for this symbol and account
	for _, pos := range positions {
		if pos.Symbol == symbol && pos.ExchangeAccountID == accountID {
			pos.Status = position.PositionClosed

			if err := c.positionRepo.Update(ctx, pos); err != nil {
				return errors.Wrap(err, "failed to close position")
			}

			c.logger.Info("Closed position",
				"position_id", pos.ID,
				"symbol", symbol,
			)
			return nil
		}
	}

	return nil // Position not found (already closed)
}

// Helper functions

func mapPositionSide(side string) position.PositionSide {
	switch side {
	case "LONG":
		return position.PositionLong
	case "SHORT":
		return position.PositionShort
	default:
		return position.PositionLong
	}
}

func parseDecimal(s string) (decimal.Decimal, error) {
	if s == "" || s == "0" {
		return decimal.Zero, nil
	}

	val, err := decimal.NewFromString(s)
	if err != nil {
		// Try parsing as float
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return decimal.Zero, err
		}
		return decimal.NewFromFloat(f), nil
	}

	return val, nil
}
