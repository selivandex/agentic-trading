package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/order"
)

// Compile-time check
var _ order.Repository = (*OrderRepository)(nil)

// OrderRepository implements order.Repository using sqlx
type OrderRepository struct {
	db *sqlx.DB
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *sqlx.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create inserts a new order
func (r *OrderRepository) Create(ctx context.Context, o *order.Order) error {
	query := `
		INSERT INTO orders (
			id, user_id, trading_pair_id, exchange_account_id,
			exchange_order_id, symbol, market_type,
			side, type, status,
			price, amount, filled_amount, avg_fill_price,
			stop_price, reduce_only,
			agent_id, reasoning, parent_order_id,
			fee, fee_currency,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		)`

	_, err := r.db.ExecContext(ctx, query,
		o.ID, o.UserID, o.TradingPairID, o.ExchangeAccountID,
		o.ExchangeOrderID, o.Symbol, o.MarketType,
		o.Side, o.Type, o.Status,
		o.Price, o.Amount, o.FilledAmount, o.AvgFillPrice,
		o.StopPrice, o.ReduceOnly,
		o.AgentID, o.Reasoning, o.ParentOrderID,
		o.Fee, o.FeeCurrency,
		o.CreatedAt, o.UpdatedAt,
	)

	return err
}

// CreateBatch inserts multiple orders in a transaction
func (r *OrderRepository) CreateBatch(ctx context.Context, orders []*order.Order) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO orders (
			id, user_id, trading_pair_id, exchange_account_id,
			symbol, side, type, status, price, amount, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)`

	for _, o := range orders {
		_, err := tx.ExecContext(ctx, query,
			o.ID, o.UserID, o.TradingPairID, o.ExchangeAccountID,
			o.Symbol, o.Side, o.Type, o.Status, o.Price, o.Amount, o.CreatedAt, o.UpdatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetByID retrieves an order by ID
func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*order.Order, error) {
	var o order.Order

	query := `SELECT * FROM orders WHERE id = $1`

	err := r.db.GetContext(ctx, &o, query, id)
	if err != nil {
		return nil, err
	}

	return &o, nil
}

// GetByExchangeOrderID retrieves an order by exchange order ID
func (r *OrderRepository) GetByExchangeOrderID(ctx context.Context, exchangeOrderID string) (*order.Order, error) {
	var o order.Order

	query := `SELECT * FROM orders WHERE exchange_order_id = $1`

	err := r.db.GetContext(ctx, &o, query, exchangeOrderID)
	if err != nil {
		return nil, err
	}

	return &o, nil
}

// GetOpenByUser retrieves all open orders for a user
func (r *OrderRepository) GetOpenByUser(ctx context.Context, userID uuid.UUID) ([]*order.Order, error) {
	var orders []*order.Order

	query := `
		SELECT * FROM orders
		WHERE user_id = $1 AND status IN ('pending', 'open', 'partial')
		ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &orders, query, userID)
	if err != nil {
		return nil, err
	}

	return orders, nil
}

// GetByTradingPair retrieves all orders for a trading pair
func (r *OrderRepository) GetByTradingPair(ctx context.Context, tradingPairID uuid.UUID) ([]*order.Order, error) {
	var orders []*order.Order

	query := `
		SELECT * FROM orders
		WHERE trading_pair_id = $1
		ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &orders, query, tradingPairID)
	if err != nil {
		return nil, err
	}

	return orders, nil
}

// Update updates an order
func (r *OrderRepository) Update(ctx context.Context, o *order.Order) error {
	query := `
		UPDATE orders SET
			exchange_order_id = $2,
			status = $3,
			filled_amount = $4,
			avg_fill_price = $5,
			fee = $6,
			fee_currency = $7,
			updated_at = NOW(),
			filled_at = CASE WHEN $3 = 'filled' THEN NOW() ELSE filled_at END
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		o.ID, o.ExchangeOrderID, o.Status, o.FilledAmount,
		o.AvgFillPrice, o.Fee, o.FeeCurrency,
	)

	return err
}

// UpdateStatus updates order status and fill details
func (r *OrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status order.OrderStatus, filledAmount, avgPrice decimal.Decimal) error {
	query := `
		UPDATE orders SET
			status = $2,
			filled_amount = $3,
			avg_fill_price = $4,
			updated_at = NOW(),
			filled_at = CASE WHEN $2 = 'filled' THEN NOW() ELSE filled_at END
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, status, filledAmount, avgPrice)
	return err
}

// Cancel marks an order as canceled
func (r *OrderRepository) Cancel(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE orders SET status = 'canceled', updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// Delete deletes an order
func (r *OrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM orders WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
