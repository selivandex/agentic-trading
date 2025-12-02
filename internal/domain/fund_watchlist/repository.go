package fund_watchlist

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for fund watchlist data access
type Repository interface {
	Create(ctx context.Context, entry *FundWatchlist) error
	GetByID(ctx context.Context, id uuid.UUID) (*FundWatchlist, error)
	GetBySymbol(ctx context.Context, symbol, marketType string) (*FundWatchlist, error)
	GetActive(ctx context.Context) ([]*FundWatchlist, error)
	GetAll(ctx context.Context) ([]*FundWatchlist, error)
	Update(ctx context.Context, entry *FundWatchlist) error
	Delete(ctx context.Context, id uuid.UUID) error
	IsActive(ctx context.Context, symbol, marketType string) (bool, error)
}
