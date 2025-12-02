package seeds

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/order"
	"prometheus/internal/testsupport"
)

// OrderBuilder provides a fluent API for creating Order entities
type OrderBuilder struct {
	db     DBTX
	ctx    context.Context
	entity *order.Order
}

// NewOrderBuilder creates a new OrderBuilder with sensible defaults
func NewOrderBuilder(db DBTX, ctx context.Context) *OrderBuilder {
	now := time.Now()
	return &OrderBuilder{
		db:  db,
		ctx: ctx,
		entity: &order.Order{
			ID:                uuid.New(),
			UserID:            uuid.Nil, // Must be set
			StrategyID:        nil,
			ExchangeAccountID: uuid.Nil, // Must be set
			ExchangeOrderID:   testsupport.UniqueName("ORDER"),
			Symbol:            "BTC/USDT",
			MarketType:        "spot",
			Side:              order.OrderSideBuy,
			Type:              order.OrderTypeLimit,
			Status:            order.OrderStatusOpen,
			Price:             decimal.NewFromFloat(40000.0),
			Amount:            decimal.NewFromFloat(0.1),
			FilledAmount:      decimal.Zero,
			AvgFillPrice:      decimal.Zero,
			StopPrice:         decimal.Zero,
			ReduceOnly:        false,
			AgentID:           "test_agent",
			Reasoning:         "Test order",
			ParentOrderID:     nil,
			Fee:               decimal.Zero,
			FeeCurrency:       "USDT",
			CreatedAt:         now,
			UpdatedAt:         now,
			FilledAt:          nil,
		},
	}
}

// WithID sets a specific ID
func (b *OrderBuilder) WithID(id uuid.UUID) *OrderBuilder {
	b.entity.ID = id
	return b
}

// WithUserID sets the user ID (required)
func (b *OrderBuilder) WithUserID(userID uuid.UUID) *OrderBuilder {
	b.entity.UserID = userID
	return b
}

// WithStrategyID sets the strategy ID
func (b *OrderBuilder) WithStrategyID(strategyID uuid.UUID) *OrderBuilder {
	b.entity.StrategyID = &strategyID
	return b
}

// WithExchangeAccountID sets the exchange account ID (required)
func (b *OrderBuilder) WithExchangeAccountID(accountID uuid.UUID) *OrderBuilder {
	b.entity.ExchangeAccountID = accountID
	return b
}

// WithExchangeOrderID sets the exchange order ID
func (b *OrderBuilder) WithExchangeOrderID(id string) *OrderBuilder {
	b.entity.ExchangeOrderID = id
	return b
}

// WithSymbol sets the trading symbol
func (b *OrderBuilder) WithSymbol(symbol string) *OrderBuilder {
	b.entity.Symbol = symbol
	return b
}

// WithMarketType sets the market type
func (b *OrderBuilder) WithMarketType(marketType string) *OrderBuilder {
	b.entity.MarketType = marketType
	return b
}

// WithSide sets the order side
func (b *OrderBuilder) WithSide(side order.OrderSide) *OrderBuilder {
	b.entity.Side = side
	return b
}

// WithBuy sets order side to buy
func (b *OrderBuilder) WithBuy() *OrderBuilder {
	b.entity.Side = order.OrderSideBuy
	return b
}

// WithSell sets order side to sell
func (b *OrderBuilder) WithSell() *OrderBuilder {
	b.entity.Side = order.OrderSideSell
	return b
}

// WithType sets the order type
func (b *OrderBuilder) WithType(orderType order.OrderType) *OrderBuilder {
	b.entity.Type = orderType
	return b
}

// WithMarket sets order type to market
func (b *OrderBuilder) WithMarket() *OrderBuilder {
	b.entity.Type = order.OrderTypeMarket
	return b
}

// WithLimit sets order type to limit
func (b *OrderBuilder) WithLimit() *OrderBuilder {
	b.entity.Type = order.OrderTypeLimit
	return b
}

// WithStatus sets the order status
func (b *OrderBuilder) WithStatus(status order.OrderStatus) *OrderBuilder {
	b.entity.Status = status
	return b
}

// WithOpen sets order status to open
func (b *OrderBuilder) WithOpen() *OrderBuilder {
	b.entity.Status = order.OrderStatusOpen
	return b
}

// WithFilled sets order status to filled
func (b *OrderBuilder) WithFilled() *OrderBuilder {
	now := time.Now()
	b.entity.Status = order.OrderStatusFilled
	b.entity.FilledAmount = b.entity.Amount
	b.entity.AvgFillPrice = b.entity.Price
	b.entity.FilledAt = &now
	return b
}

// WithCanceled sets order status to canceled
func (b *OrderBuilder) WithCanceled() *OrderBuilder {
	b.entity.Status = order.OrderStatusCanceled
	return b
}

// WithPrice sets the order price
func (b *OrderBuilder) WithPrice(price decimal.Decimal) *OrderBuilder {
	b.entity.Price = price
	return b
}

// WithAmount sets the order amount
func (b *OrderBuilder) WithAmount(amount decimal.Decimal) *OrderBuilder {
	b.entity.Amount = amount
	return b
}

// WithFilledAmount sets the filled amount
func (b *OrderBuilder) WithFilledAmount(amount decimal.Decimal) *OrderBuilder {
	b.entity.FilledAmount = amount
	return b
}

// WithAvgFillPrice sets the average fill price
func (b *OrderBuilder) WithAvgFillPrice(price decimal.Decimal) *OrderBuilder {
	b.entity.AvgFillPrice = price
	return b
}

// WithStopPrice sets the stop price
func (b *OrderBuilder) WithStopPrice(price decimal.Decimal) *OrderBuilder {
	b.entity.StopPrice = price
	return b
}

// WithReduceOnly sets reduce only flag
func (b *OrderBuilder) WithReduceOnly(reduceOnly bool) *OrderBuilder {
	b.entity.ReduceOnly = reduceOnly
	return b
}

// WithAgentID sets the agent ID
func (b *OrderBuilder) WithAgentID(agentID string) *OrderBuilder {
	b.entity.AgentID = agentID
	return b
}

// WithReasoning sets the reasoning
func (b *OrderBuilder) WithReasoning(reasoning string) *OrderBuilder {
	b.entity.Reasoning = reasoning
	return b
}

// WithParentOrder sets the parent order ID
func (b *OrderBuilder) WithParentOrder(parentID uuid.UUID) *OrderBuilder {
	b.entity.ParentOrderID = &parentID
	return b
}

// WithFee sets the fee
func (b *OrderBuilder) WithFee(fee decimal.Decimal, currency string) *OrderBuilder {
	b.entity.Fee = fee
	b.entity.FeeCurrency = currency
	return b
}

// Build returns the built entity without inserting to DB
func (b *OrderBuilder) Build() *order.Order {
	return b.entity
}

// Insert inserts the order into the database and returns the entity
func (b *OrderBuilder) Insert() (*order.Order, error) {
	if b.entity.UserID == uuid.Nil {
		return nil, fmt.Errorf("user_id is required")
	}
	if b.entity.ExchangeAccountID == uuid.Nil {
		return nil, fmt.Errorf("exchange_account_id is required")
	}

	query := `
		INSERT INTO orders (
			id, user_id, strategy_id, exchange_account_id, exchange_order_id,
			symbol, market_type, side, type, status,
			price, amount, filled_amount, avg_fill_price, stop_price,
			reduce_only, agent_id, reasoning, parent_order_id,
			fee, fee_currency, created_at, updated_at, filled_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
	`

	_, err := b.db.ExecContext(
		b.ctx,
		query,
		b.entity.ID,
		b.entity.UserID,
		b.entity.StrategyID,
		b.entity.ExchangeAccountID,
		b.entity.ExchangeOrderID,
		b.entity.Symbol,
		b.entity.MarketType,
		b.entity.Side,
		b.entity.Type,
		b.entity.Status,
		b.entity.Price,
		b.entity.Amount,
		b.entity.FilledAmount,
		b.entity.AvgFillPrice,
		b.entity.StopPrice,
		b.entity.ReduceOnly,
		b.entity.AgentID,
		b.entity.Reasoning,
		b.entity.ParentOrderID,
		b.entity.Fee,
		b.entity.FeeCurrency,
		b.entity.CreatedAt,
		b.entity.UpdatedAt,
		b.entity.FilledAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to insert order: %w", err)
	}

	return b.entity, nil
}

// MustInsert inserts the order and panics on error (useful for tests)
func (b *OrderBuilder) MustInsert() *order.Order {
	entity, err := b.Insert()
	if err != nil {
		panic(err)
	}
	return entity
}
