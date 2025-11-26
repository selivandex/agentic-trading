package trading_pair

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for trading pair data access
type Repository interface {
	Create(ctx context.Context, pair *TradingPair) error
	GetByID(ctx context.Context, id uuid.UUID) (*TradingPair, error)
	GetByUser(ctx context.Context, userID uuid.UUID) ([]*TradingPair, error)
	GetActiveByUser(ctx context.Context, userID uuid.UUID) ([]*TradingPair, error)
	GetActiveBySymbol(ctx context.Context, symbol string) ([]*TradingPair, error)
	FindAllActive(ctx context.Context) ([]*TradingPair, error)
	Update(ctx context.Context, pair *TradingPair) error
	Pause(ctx context.Context, id uuid.UUID, reason string) error
	Resume(ctx context.Context, id uuid.UUID) error
	Disable(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}
