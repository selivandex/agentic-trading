package consumers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/order"
	"prometheus/internal/domain/position"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// UserDataConsumer consumes User Data events from Kafka and updates the database
// Handles order updates, position updates, balance updates, and margin calls
type UserDataConsumer struct {
	consumer        *kafka.Consumer
	orderRepo       order.Repository
	positionRepo    position.Repository
	logger          *logger.Logger
	processingCount int64
}

// NewUserDataConsumer creates a new User Data event consumer
func NewUserDataConsumer(
	consumer *kafka.Consumer,
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

// Start begins consuming User Data events from multiple topics
func (c *UserDataConsumer) Start(ctx context.Context) error {
	topics := []string{
		"user-data-order-updates",
		"user-data-position-updates",
		"user-data-balance-updates",
		"user-data-margin-calls",
		"user-data-account-config",
	}

	c.logger.Info("Starting User Data Consumer",
		"topics", topics,
	)

	return c.consumer.Consume(ctx, topics, c.handleMessage)
}

// handleMessage routes Kafka messages to appropriate handlers based on event type
func (c *UserDataConsumer) handleMessage(ctx context.Context, topic string, key, value []byte) error {
	c.processingCount++

	switch topic {
	case "user-data-order-updates":
		return c.handleOrderUpdate(ctx, value)
	case "user-data-position-updates":
		return c.handlePositionUpdate(ctx, value)
	case "user-data-balance-updates":
		return c.handleBalanceUpdate(ctx, value)
	case "user-data-margin-calls":
		return c.handleMarginCall(ctx, value)
	case "user-data-account-config":
		return c.handleAccountConfigUpdate(ctx, value)
	default:
		c.logger.Warn("Unknown topic",
			"topic", topic,
		)
		return nil
	}
}

// handleOrderUpdate processes order update events and updates orders table
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

	c.logger.Debug("Processing order update",
		"user_id", userID,
		"account_id", accountID,
		"order_id", event.OrderId,
		"symbol", event.Symbol,
		"status", event.Status,
		"execution_type", event.ExecutionType,
	)

	// Find existing order by exchange_order_id or client_order_id
	existingOrder, err := c.orderRepo.GetByExchangeOrderID(ctx, event.OrderId)
	if err != nil && !errors.IsNotFound(err) {
		return errors.Wrap(err, "failed to fetch existing order")
	}

	now := time.Now()

	if existingOrder == nil {
		// Create new order
		newOrder := &order.Order{
			ID:                uuid.New(),
			UserID:            userID,
			ExchangeAccountID: accountID,
			ExchangeOrderID:   &event.OrderId,
			Symbol:            event.Symbol,
			Side:              mapOrderSide(event.Side),
			Type:              mapOrderType(event.Type),
			Status:            mapOrderStatus(event.Status),
			Price:             parseDecimalSafe(event.AvgPrice),
			Amount:            parseDecimalSafe(event.OriginalQty),
			FilledAmount:      parseDecimalSafe(event.FilledQty),
			AvgFillPrice:      parseDecimalSafe(event.AvgPrice),
			StopPrice:         parseDecimalSafe(event.StopPrice),
			Fee:               parseDecimalSafe(event.Commission),
			FeeCurrency:       event.CommissionAsset,
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		// Set filled_at if order is filled
		if event.Status == "FILLED" || event.Status == "PARTIALLY_FILLED" {
			filledAt := event.TradeTime.AsTime()
			newOrder.FilledAt = &filledAt
		}

		if err := c.orderRepo.Create(ctx, newOrder); err != nil {
			return errors.Wrap(err, "failed to create order")
		}

		c.logger.Info("Created new order from User Data event",
			"order_id", newOrder.ID,
			"exchange_order_id", event.OrderId,
			"symbol", event.Symbol,
			"status", event.Status,
		)
	} else {
		// Update existing order
		existingOrder.Status = mapOrderStatus(event.Status)
		existingOrder.FilledAmount = parseDecimalSafe(event.FilledQty)
		existingOrder.AvgFillPrice = parseDecimalSafe(event.AvgPrice)
		existingOrder.UpdatedAt = now

		// Update fee if provided
		if event.Commission != "" && event.Commission != "0" {
			existingOrder.Fee = parseDecimalSafe(event.Commission)
			existingOrder.FeeCurrency = event.CommissionAsset
		}

		// Set filled_at if order just got filled
		if (event.Status == "FILLED" || event.Status == "PARTIALLY_FILLED") && existingOrder.FilledAt == nil {
			filledAt := event.TradeTime.AsTime()
			existingOrder.FilledAt = &filledAt
		}

		if err := c.orderRepo.Update(ctx, existingOrder); err != nil {
			return errors.Wrap(err, "failed to update order")
		}

		c.logger.Debug("Updated order from User Data event",
			"order_id", existingOrder.ID,
			"exchange_order_id", event.OrderId,
			"status", event.Status,
		)
	}

	return nil
}

// handlePositionUpdate processes position update events and updates positions table
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

	c.logger.Debug("Processing position update",
		"user_id", userID,
		"account_id", accountID,
		"symbol", event.Symbol,
		"side", event.Side,
		"amount", event.Amount,
	)

	// Check if amount is zero (position closed)
	amount := parseDecimalSafe(event.Amount)
	if amount == nil || *amount == 0 {
		// Close position
		if err := c.closePosition(ctx, userID, accountID, event.Symbol); err != nil {
			return errors.Wrap(err, "failed to close position")
		}
		return nil
	}

	// Find existing open position
	existingPos, err := c.positionRepo.GetOpenBySymbol(ctx, userID, accountID, event.Symbol)
	if err != nil && !errors.IsNotFound(err) {
		return errors.Wrap(err, "failed to fetch existing position")
	}

	now := time.Now()

	if existingPos == nil {
		// Create new position
		newPos := &position.Position{
			ID:                uuid.New(),
			UserID:            userID,
			ExchangeAccountID: accountID,
			Symbol:            event.Symbol,
			Side:              mapPositionSide(event.Side),
			Size:              *amount,
			EntryPrice:        parseDecimalSafe(event.EntryPrice),
			CurrentPrice:      parseDecimalSafe(event.MarkPrice),
			UnrealizedPnL:     parseDecimalSafe(event.UnrealizedPnl),
			Status:            position.StatusOpen,
			OpenedAt:          now,
			UpdatedAt:         now,
		}

		// Calculate PnL percentage if we have entry and mark price
		if newPos.EntryPrice != nil && newPos.CurrentPrice != nil && *newPos.EntryPrice != 0 {
			pnlPct := ((*newPos.CurrentPrice - *newPos.EntryPrice) / *newPos.EntryPrice) * 100
			if newPos.Side == position.SideShort {
				pnlPct = -pnlPct
			}
			newPos.UnrealizedPnLPct = &pnlPct
		}

		if err := c.positionRepo.Create(ctx, newPos); err != nil {
			return errors.Wrap(err, "failed to create position")
		}

		c.logger.Info("Created new position from User Data event",
			"position_id", newPos.ID,
			"symbol", event.Symbol,
			"side", event.Side,
			"size", amount,
		)
	} else {
		// Update existing position
		existingPos.Size = *amount
		existingPos.CurrentPrice = parseDecimalSafe(event.MarkPrice)
		existingPos.UnrealizedPnL = parseDecimalSafe(event.UnrealizedPnl)
		existingPos.UpdatedAt = now

		// Update PnL percentage
		if existingPos.EntryPrice != nil && existingPos.CurrentPrice != nil && *existingPos.EntryPrice != 0 {
			pnlPct := ((*existingPos.CurrentPrice - *existingPos.EntryPrice) / *existingPos.EntryPrice) * 100
			if existingPos.Side == position.SideShort {
				pnlPct = -pnlPct
			}
			existingPos.UnrealizedPnLPct = &pnlPct
		}

		if err := c.positionRepo.Update(ctx, existingPos); err != nil {
			return errors.Wrap(err, "failed to update position")
		}

		c.logger.Debug("Updated position from User Data event",
			"position_id", existingPos.ID,
			"symbol", event.Symbol,
			"unrealized_pnl", event.UnrealizedPnl,
		)
	}

	return nil
}

// handleBalanceUpdate processes balance update events
// Currently just logs - could update user balance table if we add one
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
	// For now, we just log them for observability

	return nil
}

// handleMarginCall processes margin call events (CRITICAL!)
func (c *UserDataConsumer) handleMarginCall(ctx context.Context, data []byte) error {
	var event eventspb.UserDataMarginCallEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "failed to unmarshal margin call event")
	}

	userID, err := uuid.Parse(event.Base.UserId)
	if err != nil {
		return errors.Wrap(err, "invalid user_id")
	}

	c.logger.Error("⚠️ MARGIN CALL RECEIVED",
		"user_id", userID,
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
	existingPos, err := c.positionRepo.GetOpenBySymbol(ctx, userID, accountID, symbol)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil // Position already closed
		}
		return errors.Wrap(err, "failed to fetch position")
	}

	now := time.Now()
	existingPos.Status = position.StatusClosed
	existingPos.ClosedAt = &now
	existingPos.UpdatedAt = now

	// Calculate realized PnL if we have prices
	if existingPos.EntryPrice != nil && existingPos.CurrentPrice != nil {
		realizedPnL := (existingPos.CurrentPrice.Sub(*existingPos.EntryPrice)).Mul(existingPos.Size)
		if existingPos.Side == position.SideShort {
			realizedPnL = realizedPnL.Neg()
		}
		existingPos.RealizedPnL = &realizedPnL
	}

	if err := c.positionRepo.Update(ctx, existingPos); err != nil {
		return errors.Wrap(err, "failed to close position")
	}

	c.logger.Info("Closed position from User Data event",
		"position_id", existingPos.ID,
		"symbol", symbol,
		"realized_pnl", existingPos.RealizedPnL,
	)

	return nil
}

// Helper functions to map string values to domain types

func mapOrderSide(side string) order.Side {
	switch side {
	case "BUY":
		return order.SideBuy
	case "SELL":
		return order.SideSell
	default:
		return order.SideBuy
	}
}

func mapOrderType(orderType string) order.Type {
	switch orderType {
	case "MARKET":
		return order.TypeMarket
	case "LIMIT":
		return order.TypeLimit
	case "STOP_MARKET":
		return order.TypeStopMarket
	case "STOP", "STOP_LOSS":
		return order.TypeStopMarket
	default:
		return order.TypeMarket
	}
}

func mapOrderStatus(status string) order.Status {
	switch status {
	case "NEW":
		return order.StatusOpen
	case "PARTIALLY_FILLED":
		return order.StatusPartial
	case "FILLED":
		return order.StatusFilled
	case "CANCELED", "CANCELLED":
		return order.StatusCanceled
	case "REJECTED":
		return order.StatusRejected
	case "EXPIRED":
		return order.StatusExpired
	default:
		return order.StatusPending
	}
}

func mapPositionSide(side string) position.Side {
	switch side {
	case "LONG":
		return position.SideLong
	case "SHORT":
		return position.SideShort
	default:
		return position.SideLong
	}
}

func parseDecimalSafe(s string) *float64 {
	if s == "" || s == "0" {
		return nil
	}
	var val float64
	if _, err := fmt.Sscanf(s, "%f", &val); err != nil {
		return nil
	}
	return &val
}

