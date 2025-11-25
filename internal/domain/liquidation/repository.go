package liquidation

import (
	"context"
	"time"
)

// Repository defines the interface for liquidation data access
type Repository interface {
	// Liquidation operations
	InsertLiquidation(ctx context.Context, liq *Liquidation) error
	InsertLiquidationBatch(ctx context.Context, liqs []Liquidation) error
	GetRecentLiquidations(ctx context.Context, exchange, symbol string, since time.Time) ([]Liquidation, error)
	GetLiquidationsByValue(ctx context.Context, symbol string, minValue float64, limit int) ([]Liquidation, error)

	// Heatmap operations (stored as aggregated data or calculated on-the-fly)
	GetLiquidationHeatmap(ctx context.Context, symbol string) (*LiquidationHeatmap, error)
}
