package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"prometheus/internal/domain/fundwatchlist"
	"prometheus/pkg/errors"
)

// Compile-time check
var _ fundwatchlist.Repository = (*FundWatchlistRepository)(nil)

// FundWatchlistRepository implements fundwatchlist.Repository using sqlx
type FundWatchlistRepository struct {
	db DBTX
}

// NewFundWatchlistRepository creates a new fund watchlist repository
func NewFundWatchlistRepository(db DBTX) *FundWatchlistRepository {
	return &FundWatchlistRepository{db: db}
}

// Create inserts a new fund watchlist entry
func (r *FundWatchlistRepository) Create(ctx context.Context, entry *fundwatchlist.Watchlist) error {
	query := `
		INSERT INTO fund_watchlist (
			id, symbol, market_type, category, tier,
			is_active, is_paused, paused_reason,
			last_analyzed_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`

	_, err := r.db.ExecContext(ctx, query,
		entry.ID, entry.Symbol, entry.MarketType, entry.Category, entry.Tier,
		entry.IsActive, entry.IsPaused, entry.PausedReason,
		entry.LastAnalyzedAt, entry.CreatedAt, entry.UpdatedAt,
	)

	return err
}

// GetByID retrieves a fund watchlist entry by ID
func (r *FundWatchlistRepository) GetByID(ctx context.Context, id uuid.UUID) (*fundwatchlist.Watchlist, error) {
	var entry fundwatchlist.Watchlist

	query := `SELECT * FROM fund_watchlist WHERE id = $1`

	err := r.db.GetContext(ctx, &entry, query, id)
	if err == sql.ErrNoRows {
		return nil, errors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// GetBySymbol retrieves a fund watchlist entry by symbol and market type
func (r *FundWatchlistRepository) GetBySymbol(ctx context.Context, symbol, marketType string) (*fundwatchlist.Watchlist, error) {
	var entry fundwatchlist.Watchlist

	query := `SELECT * FROM fund_watchlist WHERE symbol = $1 AND market_type = $2`

	err := r.db.GetContext(ctx, &entry, query, symbol, marketType)
	if err == sql.ErrNoRows {
		return nil, errors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// GetActive retrieves all active fund watchlist entries
func (r *FundWatchlistRepository) GetActive(ctx context.Context) ([]*fundwatchlist.Watchlist, error) {
	var entries []*fundwatchlist.Watchlist

	query := `
		SELECT * FROM fund_watchlist
		WHERE is_active = true AND is_paused = false
		ORDER BY tier ASC, symbol ASC`

	err := r.db.SelectContext(ctx, &entries, query)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// GetAll retrieves all fund watchlist entries
func (r *FundWatchlistRepository) GetAll(ctx context.Context) ([]*fundwatchlist.Watchlist, error) {
	var entries []*fundwatchlist.Watchlist

	query := `SELECT * FROM fund_watchlist ORDER BY tier ASC, symbol ASC`

	err := r.db.SelectContext(ctx, &entries, query)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// Update updates a fund watchlist entry
func (r *FundWatchlistRepository) Update(ctx context.Context, entry *fundwatchlist.Watchlist) error {
	query := `
		UPDATE fund_watchlist SET
			symbol = $2,
			market_type = $3,
			category = $4,
			tier = $5,
			is_active = $6,
			is_paused = $7,
			paused_reason = $8,
			last_analyzed_at = $9,
			updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		entry.ID, entry.Symbol, entry.MarketType, entry.Category, entry.Tier,
		entry.IsActive, entry.IsPaused, entry.PausedReason,
		entry.LastAnalyzedAt,
	)

	return err
}

// Delete deletes a fund watchlist entry
func (r *FundWatchlistRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM fund_watchlist WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// IsActive checks if a symbol is actively monitored
func (r *FundWatchlistRepository) IsActive(ctx context.Context, symbol, marketType string) (bool, error) {
	var isActive bool

	query := `
		SELECT EXISTS(
			SELECT 1 FROM fund_watchlist
			WHERE symbol = $1 AND market_type = $2
			  AND is_active = true AND is_paused = false
		)`

	err := r.db.GetContext(ctx, &isActive, query, symbol, marketType)
	if err != nil {
		return false, err
	}

	return isActive, nil
}
