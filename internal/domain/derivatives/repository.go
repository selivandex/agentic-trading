package derivatives

import (
	"context"
	"time"
)

// Repository defines the interface for derivatives data access
type Repository interface {
	// Options snapshot operations
	InsertOptionsSnapshot(ctx context.Context, snapshot *OptionsSnapshot) error
	GetLatestOptionsSnapshot(ctx context.Context, symbol string) (*OptionsSnapshot, error)
	GetOptionsHistory(ctx context.Context, symbol string, since time.Time) ([]OptionsSnapshot, error)

	// Options flow operations
	InsertOptionsFlow(ctx context.Context, flow *OptionsFlow) error
	GetLargeOptionsFlows(ctx context.Context, symbol string, minPremium float64, limit int) ([]OptionsFlow, error)
	GetRecentOptionsFlows(ctx context.Context, symbol string, since time.Time) ([]OptionsFlow, error)
}
