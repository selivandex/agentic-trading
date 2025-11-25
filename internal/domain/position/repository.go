package position

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Repository defines the interface for position data access
type Repository interface {
	Create(ctx context.Context, position *Position) error
	GetByID(ctx context.Context, id uuid.UUID) (*Position, error)
	GetOpenByUser(ctx context.Context, userID uuid.UUID) ([]*Position, error)
	GetByTradingPair(ctx context.Context, tradingPairID uuid.UUID) ([]*Position, error)
	Update(ctx context.Context, position *Position) error
	UpdatePnL(ctx context.Context, id uuid.UUID, currentPrice, unrealizedPnL, unrealizedPnLPct decimal.Decimal) error
	Close(ctx context.Context, id uuid.UUID, exitPrice, realizedPnL decimal.Decimal) error
	Delete(ctx context.Context, id uuid.UUID) error
}
