package trading

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/order"
	"prometheus/internal/domain/position"
	"prometheus/internal/domain/user"
	"prometheus/internal/events"
	"prometheus/internal/workers"
	"prometheus/pkg/crypto"
	"prometheus/pkg/errors"
)

// OrderSync synchronizes order status with exchanges
type OrderSync struct {
	*workers.BaseWorker
	userRepo       user.Repository
	orderRepo      order.Repository
	posRepo        position.Repository
	accountRepo    exchange_account.Repository
	exchFactory    exchanges.Factory
	encryptor      crypto.Encryptor
	kafka          *kafka.Producer
	eventPublisher *events.WorkerPublisher
}

// NewOrderSync creates a new order sync worker
func NewOrderSync(
	userRepo user.Repository,
	orderRepo order.Repository,
	posRepo position.Repository,
	accountRepo exchange_account.Repository,
	exchFactory exchanges.Factory,
	encryptor crypto.Encryptor,
	kafka *kafka.Producer,
	interval time.Duration,
	enabled bool,
) *OrderSync {
	return &OrderSync{
		BaseWorker:     workers.NewBaseWorker("order_sync", interval, enabled),
		userRepo:       userRepo,
		orderRepo:      orderRepo,
		posRepo:        posRepo,
		accountRepo:    accountRepo,
		exchFactory:    exchFactory,
		encryptor:      encryptor,
		kafka:          kafka,
		eventPublisher: events.NewWorkerPublisher(kafka),
	}
}

// Run executes one iteration of order synchronization
func (os *OrderSync) Run(ctx context.Context) error {
	os.Log().Debug("Order sync: starting iteration")

	// Get all active users (using large limit)
	users, err := os.userRepo.List(ctx, 1000, 0)
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
		os.Log().Debug("No active users to sync orders for")
		return nil
	}

	os.Log().Debug("Syncing orders for users", "user_count", len(activeUsers))

	// Sync orders for each user
	successCount := 0
	errorCount := 0
	for _, usr := range activeUsers {
		if err := os.syncUserOrders(ctx, usr.ID); err != nil {
			os.Log().Error("Failed to sync user orders",
				"user_id", usr.ID,
				"error", err,
			)
			errorCount++
			// Continue with other users
			continue
		}
		successCount++
	}

	os.Log().Info("Order sync: iteration complete",
		"users_processed", successCount,
		"errors", errorCount,
	)

	return nil
}

// syncUserOrders syncs orders for a specific user
func (os *OrderSync) syncUserOrders(ctx context.Context, userID uuid.UUID) error {
	// Get all pending/partially filled orders for this user
	openOrders, err := os.orderRepo.GetOpenByUser(ctx, userID)
	if err != nil {
		return errors.Wrapf(err, "failed to get open orders for user %s", userID)
	}

	if len(openOrders) == 0 {
		return nil // No orders to sync
	}

	os.Log().Debug("Syncing orders", "user_id", userID, "count", len(openOrders))

	// Group orders by exchange account
	ordersByAccount := make(map[uuid.UUID][]*order.Order)
	for _, ord := range openOrders {
		ordersByAccount[ord.ExchangeAccountID] = append(ordersByAccount[ord.ExchangeAccountID], ord)
	}

	// Process each exchange account
	for accountID, accountOrders := range ordersByAccount {
		if err := os.syncAccountOrders(ctx, accountID, accountOrders); err != nil {
			os.Log().Error("Failed to sync orders for account",
				"account_id", accountID,
				"error", err,
			)
			// Continue with other accounts even if one fails
		}
	}

	return nil
}

// syncAccountOrders syncs orders for a specific exchange account
func (os *OrderSync) syncAccountOrders(ctx context.Context, accountID uuid.UUID, orders []*order.Order) error {
	// Get exchange account to get credentials
	account, err := os.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return errors.Wrap(err, "failed to get exchange account")
	}

	// Create exchange client (factory handles decryption internally)
	exchangeClient, err := os.exchFactory.CreateClient(account, &os.encryptor)
	if err != nil {
		return errors.Wrap(err, "failed to create exchange client")
	}

	// Sync each order
	for _, ord := range orders {
		if err := os.syncOrder(ctx, exchangeClient, ord); err != nil {
			os.Log().Error("Failed to sync order",
				"order_id", ord.ID,
				"symbol", ord.Symbol,
				"error", err,
			)
			// Continue with other orders
			continue
		}
	}

	return nil
}

// syncOrder syncs a single order with the exchange
func (os *OrderSync) syncOrder(ctx context.Context, exchange exchanges.Exchange, ord *order.Order) error {
	// If we don't have an exchange order ID yet, skip
	if ord.ExchangeOrderID == "" {
		return nil
	}

	// Get order status from exchange
	// Note: This is simplified - real implementation would need to call exchange API
	// to get order status by exchange_order_id
	// For now, we'll just log
	os.Log().Debug("Would sync order with exchange",
		"order_id", ord.ID,
		"exchange_order_id", ord.ExchangeOrderID,
		"symbol", ord.Symbol,
	)

	// TODO: Implement actual exchange API call
	// exchangeOrder, err := exchange.GetOrder(ctx, ord.Symbol, *ord.ExchangeOrderID)
	// if err != nil {
	//     return errors.Wrap(err, "failed to get order from exchange")
	// }

	// For demonstration, let's assume we got the exchange order status
	// and handle different scenarios:

	// Scenario 1: Order is filled
	// os.handleFilledOrder(ctx, ord, exchangeOrder)

	// Scenario 2: Order is cancelled
	// os.handleCancelledOrder(ctx, ord)

	// Scenario 3: Order is partially filled
	// os.handlePartiallyFilledOrder(ctx, ord, exchangeOrder)

	return nil
}

// handleFilledOrder handles a filled order
func (os *OrderSync) handleFilledOrder(ctx context.Context, ord *order.Order, filledAmount, avgPrice decimal.Decimal) error {
	// Update order status
	if err := os.orderRepo.UpdateStatus(ctx, ord.ID, order.OrderStatusFilled, filledAmount, avgPrice); err != nil {
		return errors.Wrap(err, "failed to update order status")
	}

	os.Log().Info("Order filled",
		"order_id", ord.ID,
		"symbol", ord.Symbol,
		"filled_amount", filledAmount,
		"avg_price", avgPrice,
	)

	// Create or update position
	if err := os.createOrUpdatePosition(ctx, ord, filledAmount, avgPrice); err != nil {
		os.Log().Error("Failed to create/update position", "error", err)
	}

	// Send Kafka event (protobuf)
	filledAmountFloat, _ := filledAmount.Float64()
	avgPriceFloat, _ := avgPrice.Float64()

	if err := os.eventPublisher.PublishOrderFilled(
		ctx,
		ord.UserID.String(),
		ord.ID.String(),
		ord.Symbol,
		"", // exchange - TODO: get from accountRepo
		ord.Side.String(),
		avgPriceFloat,
		filledAmountFloat,
		0.0,    // fee - TODO: get from exchange response
		"USDT", // fee_currency
	); err != nil {
		os.Log().Error("Failed to publish order filled event", "error", err)
	}

	return nil
}

// handleCancelledOrder handles a cancelled order
func (os *OrderSync) handleCancelledOrder(ctx context.Context, ord *order.Order) error {
	// Update order status
	if err := os.orderRepo.Cancel(ctx, ord.ID); err != nil {
		return errors.Wrap(err, "failed to cancel order")
	}

	os.Log().Info("Order cancelled",
		"order_id", ord.ID,
		"symbol", ord.Symbol,
	)

	// Send Kafka event (protobuf)
	if err := os.eventPublisher.PublishOrderCancelled(
		ctx,
		ord.UserID.String(),
		ord.ID.String(),
		ord.Symbol,
		"",               // exchange - TODO: get from accountRepo
		"user_requested", // reason
	); err != nil {
		os.Log().Error("Failed to publish order cancelled event", "error", err)
	}

	return nil
}

// handlePartiallyFilledOrder handles a partially filled order
func (os *OrderSync) handlePartiallyFilledOrder(ctx context.Context, ord *order.Order, filledAmount, avgPrice decimal.Decimal) error {
	// Update order status (using Pending status for partially filled as specific status may not exist)
	if err := os.orderRepo.UpdateStatus(ctx, ord.ID, order.OrderStatusPending, filledAmount, avgPrice); err != nil {
		return errors.Wrap(err, "failed to update order status")
	}

	os.Log().Info("Order partially filled",
		"order_id", ord.ID,
		"symbol", ord.Symbol,
		"filled_amount", filledAmount,
		"total_amount", ord.Amount,
	)

	// Update position if applicable
	if err := os.createOrUpdatePosition(ctx, ord, filledAmount, avgPrice); err != nil {
		os.Log().Error("Failed to create/update position", "error", err)
	}

	// Get exchange from account
	account, err := os.accountRepo.GetByID(ctx, ord.ExchangeAccountID)
	exchangeName := "unknown"
	if err == nil && account != nil {
		exchangeName = account.Exchange.String()
	}

	// Send Kafka event (protobuf) - use OrderFilled with partial status
	filledAmountFloat, _ := filledAmount.Float64()
	avgPriceFloat, _ := avgPrice.Float64()

	if err := os.eventPublisher.PublishOrderFilled(
		ctx,
		ord.UserID.String(),
		ord.ID.String(),
		ord.Symbol,
		exchangeName,
		ord.Side.String(),
		avgPriceFloat,
		filledAmountFloat,
		0.0,
		"USDT",
	); err != nil {
		os.Log().Error("Failed to publish order partially filled event", "error", err)
	}

	return nil
}

// createOrUpdatePosition creates a new position or updates existing one
func (os *OrderSync) createOrUpdatePosition(ctx context.Context, ord *order.Order, filledAmount, avgPrice decimal.Decimal) error {
	// Check if position already exists for this trading pair
	// This is simplified - in reality you'd need more logic to handle:
	// - Multiple positions for the same symbol
	// - Closing positions vs opening new ones
	// - Partial fills accumulating into one position

	os.Log().Debug("Would create/update position",
		"order_id", ord.ID,
		"symbol", ord.Symbol,
		"filled_amount", filledAmount,
		"avg_price", avgPrice,
	)

	// TODO: Implement actual position logic
	// For now, we'll skip this as it requires complex business logic

	return nil
}

// Event structures

type OrderFilledEvent struct {
	OrderID      string    `json:"order_id"`
	UserID       string    `json:"user_id"`
	Symbol       string    `json:"symbol"`
	Side         string    `json:"side"`
	Type         string    `json:"type"`
	FilledAmount string    `json:"filled_amount"`
	AvgPrice     string    `json:"avg_price"`
	Timestamp    time.Time `json:"timestamp"`
}

type OrderCancelledEvent struct {
	OrderID   string    `json:"order_id"`
	UserID    string    `json:"user_id"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Timestamp time.Time `json:"timestamp"`
}

type OrderPartiallyFilledEvent struct {
	OrderID      string    `json:"order_id"`
	UserID       string    `json:"user_id"`
	Symbol       string    `json:"symbol"`
	Side         string    `json:"side"`
	FilledAmount string    `json:"filled_amount"`
	TotalAmount  string    `json:"total_amount"`
	AvgPrice     string    `json:"avg_price"`
	Timestamp    time.Time `json:"timestamp"`
}
