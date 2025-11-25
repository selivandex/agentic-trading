package regime

import (
	"context"
	"time"
)

// Repository defines the interface for market regime data access
type Repository interface {
	Store(ctx context.Context, regime *MarketRegime) error
	GetLatest(ctx context.Context, symbol string) (*MarketRegime, error)
	GetHistory(ctx context.Context, symbol string, since time.Time) ([]MarketRegime, error)
}
