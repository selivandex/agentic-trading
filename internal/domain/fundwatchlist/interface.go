package fundwatchlist

import (
	"context"

	"github.com/google/uuid"
)

// DomainService defines interface for fundwatchlist domain operations
// This allows application layer to depend on interface, not concrete implementation
type DomainService interface {
	Create(ctx context.Context, entry *Watchlist) error
	GetByID(ctx context.Context, id uuid.UUID) (*Watchlist, error)
	GetBySymbol(ctx context.Context, symbol, marketType string) (*Watchlist, error)
	GetActive(ctx context.Context) ([]*Watchlist, error)
	GetAll(ctx context.Context) ([]*Watchlist, error)
	Update(ctx context.Context, entry *Watchlist) error
	Delete(ctx context.Context, id uuid.UUID) error
	IsActive(ctx context.Context, symbol, marketType string) (bool, error)
	Pause(ctx context.Context, symbol, marketType, reason string) error
	Resume(ctx context.Context, symbol, marketType string) error
}

// Compile-time check that Service implements DomainService
var _ DomainService = (*Service)(nil)
