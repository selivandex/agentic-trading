package order

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Repository defines the interface for order data access
type Repository interface {
	Create(ctx context.Context, order *Order) error
	CreateBatch(ctx context.Context, orders []*Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*Order, error)
	GetByExchangeOrderID(ctx context.Context, exchangeOrderID string) (*Order, error)
	GetOpenByUser(ctx context.Context, userID uuid.UUID) ([]*Order, error)
	GetByStrategy(ctx context.Context, strategyID uuid.UUID) ([]*Order, error)
	GetPending(ctx context.Context, limit int) ([]*Order, error)
	GetPendingByUser(ctx context.Context, userID uuid.UUID) ([]*Order, error)
	Update(ctx context.Context, order *Order) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status OrderStatus, filledAmount, avgPrice decimal.Decimal) error
	Cancel(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}
