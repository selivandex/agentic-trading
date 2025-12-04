package fund_watchlist

import (
	"time"

	"github.com/google/uuid"
)

// FundWatchlist represents a globally monitored symbol for the fund
// This is NOT per-user - it's a shared watchlist for all investors
type FundWatchlist struct {
	ID         uuid.UUID `db:"id"`
	Symbol     string    `db:"symbol"`      // BTC/USDT
	MarketType string    `db:"market_type"` // spot, futures

	// Metadata for categorization
	Category string `db:"category"` // major, defi, layer1, layer2, meme, etc
	Tier     int    `db:"tier"`     // 1=BTC/ETH, 2=top20, 3=top100

	// State
	IsActive     bool    `db:"is_active"`
	IsPaused     bool    `db:"is_paused"`
	PausedReason *string `db:"paused_reason"`

	// Analytics
	LastAnalyzedAt *time.Time `db:"last_analyzed_at"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// IsMonitored returns true if symbol should be actively monitored
func (f *FundWatchlist) IsMonitored() bool {
	return f.IsActive && !f.IsPaused
}


