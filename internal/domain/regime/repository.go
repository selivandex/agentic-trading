package regime

import (
	"context"
	"time"
)

// Repository defines the interface for market regime data access
type Repository interface {
	// Regime operations
	Store(ctx context.Context, regime *MarketRegime) error
	GetLatest(ctx context.Context, symbol string) (*MarketRegime, error)
	GetHistory(ctx context.Context, symbol string, since time.Time) ([]MarketRegime, error)

	// Feature operations (for ML pipeline)
	StoreFeatures(ctx context.Context, features *Features) error
	GetLatestFeatures(ctx context.Context, symbol string) (*Features, error)
	GetFeaturesHistory(ctx context.Context, symbol string, since time.Time, limit int) ([]Features, error)
}
