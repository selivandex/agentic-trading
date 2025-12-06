package seeds

import (
	"context"

	"github.com/google/uuid"

	"prometheus/internal/domain/fundwatchlist"
)

// FundWatchlistBuilder builds fundwatchlist entities for testing
type FundWatchlistBuilder struct {
	db  DBTX
	ctx context.Context
	w   *fundwatchlist.Watchlist
}

// NewFundWatchlistBuilder creates a new FundWatchlist builder
func NewFundWatchlistBuilder(db DBTX, ctx context.Context) *FundWatchlistBuilder {
	return &FundWatchlistBuilder{
		db:  db,
		ctx: ctx,
		w: &fundwatchlist.Watchlist{
			ID:         uuid.New(),
			Symbol:     "TEST/USDT",
			MarketType: "spot",
			Category:   "major",
			Tier:       1,
			IsActive:   true,
			IsPaused:   false,
		},
	}
}

// WithSymbol sets the symbol
func (b *FundWatchlistBuilder) WithSymbol(symbol string) *FundWatchlistBuilder {
	b.w.Symbol = symbol
	return b
}

// WithMarketType sets the market type
func (b *FundWatchlistBuilder) WithMarketType(marketType string) *FundWatchlistBuilder {
	b.w.MarketType = marketType
	return b
}

// WithCategory sets the category
func (b *FundWatchlistBuilder) WithCategory(category string) *FundWatchlistBuilder {
	b.w.Category = category
	return b
}

// WithTier sets the tier
func (b *FundWatchlistBuilder) WithTier(tier int) *FundWatchlistBuilder {
	b.w.Tier = tier
	return b
}

// WithIsActive sets the active status
func (b *FundWatchlistBuilder) WithIsActive(isActive bool) *FundWatchlistBuilder {
	b.w.IsActive = isActive
	return b
}

// WithIsPaused sets the paused status
func (b *FundWatchlistBuilder) WithIsPaused(isPaused bool) *FundWatchlistBuilder {
	b.w.IsPaused = isPaused
	return b
}

// WithPausedReason sets the paused reason
func (b *FundWatchlistBuilder) WithPausedReason(reason string) *FundWatchlistBuilder {
	b.w.PausedReason = &reason
	return b
}

// MustInsert inserts the watchlist entry into the database
func (b *FundWatchlistBuilder) MustInsert() *fundwatchlist.Watchlist {
	const query = `
		INSERT INTO fund_watchlist (
			id, symbol, market_type, category, tier,
			is_active, is_paused, paused_reason
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) RETURNING created_at, updated_at
	`

	err := b.db.QueryRowContext(
		b.ctx,
		query,
		b.w.ID,
		b.w.Symbol,
		b.w.MarketType,
		b.w.Category,
		b.w.Tier,
		b.w.IsActive,
		b.w.IsPaused,
		b.w.PausedReason,
	).Scan(&b.w.CreatedAt, &b.w.UpdatedAt)

	if err != nil {
		panic("failed to insert fund watchlist: " + err.Error())
	}

	return b.w
}
