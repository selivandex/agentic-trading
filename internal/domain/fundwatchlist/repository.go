package fundwatchlist

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for fund watchlist data access
type Repository interface {
	Create(ctx context.Context, entry *Watchlist) error
	GetByID(ctx context.Context, id uuid.UUID) (*Watchlist, error)
	GetBySymbol(ctx context.Context, symbol, marketType string) (*Watchlist, error)
	GetActive(ctx context.Context) ([]*Watchlist, error)
	GetAll(ctx context.Context) ([]*Watchlist, error)
	Update(ctx context.Context, entry *Watchlist) error
	Delete(ctx context.Context, id uuid.UUID) error
	IsActive(ctx context.Context, symbol, marketType string) (bool, error)
}
