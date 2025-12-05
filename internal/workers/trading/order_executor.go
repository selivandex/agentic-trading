package trading

import (
	"context"
	"fmt"
	"time"

	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/order"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// UserExchangeFactory interface for dependency injection
type UserExchangeFactory interface {
	GetClient(ctx context.Context, account *exchange_account.ExchangeAccount) (exchanges.Exchange, error)
}

// OrderExecutor polls pending orders and executes them on exchanges
// This bridges the gap between AI agent order creation and real exchange execution
type OrderExecutor struct {
	*workers.BaseWorker
	orderRepo           order.Repository
	exchangeAccountRepo exchange_account.Repository
	userExchangeFactory UserExchangeFactory
	log                 *logger.Logger
}

// NewOrderExecutor creates a new order executor worker
func NewOrderExecutor(
	orderRepo order.Repository,
	exchangeAccountRepo exchange_account.Repository,
	userExchangeFactory UserExchangeFactory,
	interval time.Duration,
	enabled bool,
) *OrderExecutor {
	return &OrderExecutor{
		BaseWorker:          workers.NewBaseWorker("order_executor", interval, enabled),
		orderRepo:           orderRepo,
		exchangeAccountRepo: exchangeAccountRepo,
		userExchangeFactory: userExchangeFactory,
		log:                 logger.Get().With("worker", "order_executor"),
	}
}

// Run executes one iteration of order execution
func (oe *OrderExecutor) Run(ctx context.Context) error {
	start := time.Now()
	oe.log.Debug("OrderExecutor: starting iteration")

	// Get pending orders (limit to 50 per iteration to avoid overload)
	pendingOrders, err := oe.orderRepo.GetPending(ctx, 50)
	if err != nil {
		oe.RecordError(err, time.Since(start))
		return errors.Wrap(err, "failed to get pending orders")
	}

	if len(pendingOrders) == 0 {
		oe.log.Debug("OrderExecutor: no pending orders")
		oe.RecordRun(time.Since(start))
		return nil
	}

	oe.log.Infow("OrderExecutor: processing pending orders",
		"count", len(pendingOrders),
	)

	successCount := 0
	failCount := 0

	// Process each pending order
	for _, ord := range pendingOrders {
		// Check for context cancellation (graceful shutdown)
		select {
		case <-ctx.Done():
			oe.log.Infow("OrderExecutor interrupted by shutdown",
				"processed", successCount+failCount,
				"remaining", len(pendingOrders)-successCount-failCount,
			)
			return ctx.Err()
		default:
		}

		if err := oe.executeOrder(ctx, ord); err != nil {
			oe.log.Errorw("Failed to execute order",
				"order_id", ord.ID,
				"symbol", ord.Symbol,
				"user_id", ord.UserID,
				"error", err,
			)
			failCount++
		} else {
			successCount++
		}
	}

	oe.log.Infow("OrderExecutor: iteration complete",
		"success", successCount,
		"failed", failCount,
		"duration", time.Since(start),
	)

	oe.RecordRun(time.Since(start))
	return nil
}

// executeOrder executes a single pending order with retry logic
func (oe *OrderExecutor) executeOrder(ctx context.Context, ord *order.Order) error {
	oe.log.Debugw("Executing order",
		"order_id", ord.ID,
		"user_id", ord.UserID,
		"symbol", ord.Symbol,
		"side", ord.Side,
		"amount", ord.Amount,
	)

	// Get exchange account
	account, err := oe.exchangeAccountRepo.GetByID(ctx, ord.ExchangeAccountID)
	if err != nil {
		// Mark order as rejected - invalid account
		_ = oe.rejectOrder(ctx, ord, "exchange account not found")
		return errors.Wrap(err, "failed to get exchange account")
	}

	// Validate account is active
	if !account.IsActive {
		_ = oe.rejectOrder(ctx, ord, "exchange account is not active")
		return errors.New("exchange account is not active")
	}

	// Get exchange client
	exchangeClient, err := oe.userExchangeFactory.GetClient(ctx, account)
	if err != nil {
		_ = oe.rejectOrder(ctx, ord, "failed to initialize exchange client")
		return errors.Wrap(err, "failed to get exchange client")
	}

	// Retry logic: 3 attempts with exponential backoff
	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Build exchange order request
		req := oe.buildExchangeOrderRequest(ord)

		// Execute on exchange
		exchangeOrder, err := exchangeClient.PlaceOrder(ctx, req)
		if err != nil {
			lastErr = err
			oe.log.Warnw("Order placement failed (will retry)",
				"order_id", ord.ID,
				"attempt", attempt,
				"max_retries", maxRetries,
				"error", err,
			)

			// Check if error is retryable
			if !oe.isRetryableError(err) {
				oe.log.Infow("Non-retryable error, rejecting order immediately",
					"order_id", ord.ID,
					"error", err,
				)
				_ = oe.rejectOrder(ctx, ord, err.Error())
				return err
			}

			// Exponential backoff: 1s, 2s, 4s
			if attempt < maxRetries {
				backoff := time.Duration(1<<(attempt-1)) * time.Second
				oe.log.Debugw("Backing off before retry",
					"order_id", ord.ID,
					"backoff", backoff,
				)
				time.Sleep(backoff)
			}
			continue
		}

		// Success! Update order with exchange order ID and status
		ord.ExchangeOrderID = exchangeOrder.ID
		ord.Status = oe.mapExchangeStatus(exchangeOrder.Status)
		ord.UpdatedAt = time.Now()

		if err := oe.orderRepo.Update(ctx, ord); err != nil {
			oe.log.Errorw("Failed to update order after successful placement",
				"order_id", ord.ID,
				"exchange_order_id", exchangeOrder.ID,
				"error", err,
			)
			return errors.Wrap(err, "failed to update order")
		}

		oe.log.Infow("Order executed successfully",
			"order_id", ord.ID,
			"exchange_order_id", exchangeOrder.ID,
			"symbol", ord.Symbol,
			"status", ord.Status,
			"attempt", attempt,
		)

		return nil
	}

	// All retries exhausted, reject order
	_ = oe.rejectOrder(ctx, ord, lastErr.Error())
	return errors.Wrapf(lastErr, "order execution failed after %d attempts", maxRetries)
}

// buildExchangeOrderRequest converts domain order to exchange API request
func (oe *OrderExecutor) buildExchangeOrderRequest(ord *order.Order) *exchanges.OrderRequest {
	return &exchanges.OrderRequest{
		Symbol:        ord.Symbol,
		Market:        oe.mapMarketType(ord.MarketType),
		Side:          oe.mapOrderSide(ord.Side),
		Type:          oe.mapOrderType(ord.Type),
		Quantity:      ord.Amount,
		Price:         ord.Price,
		StopPrice:     ord.StopPrice,
		TimeInForce:   exchanges.TimeInForceGTC, // Good-Till-Canceled by default
		ReduceOnly:    ord.ReduceOnly,
		ClientOrderID: ord.ID.String(), // Use our order ID as client order ID for tracking
	}
}

// mapMarketType converts domain market type to exchange market type
func (oe *OrderExecutor) mapMarketType(marketType string) exchanges.MarketType {
	switch marketType {
	case "spot":
		return exchanges.MarketTypeSpot
	case "futures", "linear_perp":
		return exchanges.MarketTypeLinearPerp
	case "inverse_perp":
		return exchanges.MarketTypeInversePerp
	default:
		return exchanges.MarketTypeSpot
	}
}

// mapOrderSide converts domain order side to exchange order side
func (oe *OrderExecutor) mapOrderSide(side order.OrderSide) exchanges.OrderSide {
	switch side {
	case order.OrderSideBuy:
		return exchanges.OrderSideBuy
	case order.OrderSideSell:
		return exchanges.OrderSideSell
	default:
		return exchanges.OrderSideBuy
	}
}

// mapOrderType converts domain order type to exchange order type
func (oe *OrderExecutor) mapOrderType(orderType order.OrderType) exchanges.OrderType {
	switch orderType {
	case order.OrderTypeMarket:
		return exchanges.OrderTypeMarket
	case order.OrderTypeLimit:
		return exchanges.OrderTypeLimit
	case order.OrderTypeStopMarket:
		return exchanges.OrderTypeStopMarket
	case order.OrderTypeStopLimit:
		return exchanges.OrderTypeStopLimit
	default:
		return exchanges.OrderTypeLimit
	}
}

// mapExchangeStatus converts exchange order status to domain order status
func (oe *OrderExecutor) mapExchangeStatus(status exchanges.OrderStatus) order.OrderStatus {
	switch status {
	case exchanges.OrderStatusNew, exchanges.OrderStatusOpen:
		return order.OrderStatusOpen
	case exchanges.OrderStatusPartial:
		return order.OrderStatusPartial
	case exchanges.OrderStatusFilled:
		return order.OrderStatusFilled
	case exchanges.OrderStatusCanceled:
		return order.OrderStatusCanceled
	case exchanges.OrderStatusRejected:
		return order.OrderStatusRejected
	case exchanges.OrderStatusExpired:
		return order.OrderStatusExpired
	default:
		return order.OrderStatusOpen
	}
}

// isRetryableError determines if an error should trigger a retry
func (oe *OrderExecutor) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Network errors - retryable
	if contains(errStr, "timeout", "connection", "network", "unavailable") {
		return true
	}

	// Rate limit errors - retryable
	if contains(errStr, "rate limit", "too many requests") {
		return true
	}

	// Temporary exchange errors - retryable
	if contains(errStr, "service unavailable", "temporarily unavailable") {
		return true
	}

	// Permanent errors - NOT retryable
	if contains(errStr, "insufficient", "invalid", "not found", "unauthorized", "forbidden") {
		return false
	}

	// Default: retry if uncertain
	return true
}

// contains checks if s contains any of the substrings (case-insensitive)
func contains(s string, substrs ...string) bool {
	lower := s
	for _, substr := range substrs {
		if len(substr) > 0 && len(lower) >= len(substr) {
			// Simple case-insensitive contains check
			for i := 0; i <= len(lower)-len(substr); i++ {
				match := true
				for j := 0; j < len(substr); j++ {
					c1 := lower[i+j]
					c2 := substr[j]
					if c1 >= 'A' && c1 <= 'Z' {
						c1 += 32
					}
					if c2 >= 'A' && c2 <= 'Z' {
						c2 += 32
					}
					if c1 != c2 {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
		}
	}
	return false
}

// rejectOrder marks an order as rejected with reason
func (oe *OrderExecutor) rejectOrder(ctx context.Context, ord *order.Order, reason string) error {
	oe.log.Warnw("Rejecting order",
		"order_id", ord.ID,
		"reason", reason,
	)

	ord.Status = order.OrderStatusRejected
	ord.Reasoning = fmt.Sprintf("Rejected: %s. Original reasoning: %s", reason, ord.Reasoning)
	ord.UpdatedAt = time.Now()

	if err := oe.orderRepo.Update(ctx, ord); err != nil {
		return errors.Wrap(err, "failed to update rejected order")
	}

	return nil
}
