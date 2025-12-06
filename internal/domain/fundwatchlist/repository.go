package fundwatchlist

import (
	"context"

	"github.com/google/uuid"
)

// FilterOptions contains filter criteria for fund watchlist queries
type FilterOptions struct {
	Scope      *string
	MarketType *string
	Category   *string
	Tiers      []int
	Search     *string
}

// Repository defines the interface for fund watchlist data access
type Repository interface {
	Create(ctx context.Context, entry *Watchlist) error
	GetByID(ctx context.Context, id uuid.UUID) (*Watchlist, error)
	GetBySymbol(ctx context.Context, symbol, marketType string) (*Watchlist, error)
	GetActive(ctx context.Context) ([]*Watchlist, error)
	GetAll(ctx context.Context) ([]*Watchlist, error)
	GetWithFilters(ctx context.Context, filter FilterOptions) ([]*Watchlist, error)
	CountByScope(ctx context.Context) (map[string]int, error)
	Update(ctx context.Context, entry *Watchlist) error
	Delete(ctx context.Context, id uuid.UUID) error
	IsActive(ctx context.Context, symbol, marketType string) (bool, error)
}
