package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"prometheus/internal/domain/trading_pair"
)

// Compile-time check
var _ trading_pair.Repository = (*TradingPairRepository)(nil)

// TradingPairRepository implements trading_pair.Repository using sqlx
type TradingPairRepository struct {
	db *sqlx.DB
}

// NewTradingPairRepository creates a new trading pair repository
func NewTradingPairRepository(db *sqlx.DB) *TradingPairRepository {
	return &TradingPairRepository{db: db}
}

// Create inserts a new trading pair
func (r *TradingPairRepository) Create(ctx context.Context, pair *trading_pair.TradingPair) error {
	query := `
		INSERT INTO trading_pairs (
			id, user_id, exchange_account_id, symbol, market_type,
			budget, max_position_size, max_leverage, stop_loss_percent, take_profit_percent,
			ai_provider, strategy_mode, timeframes,
			is_active, is_paused, paused_reason, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)`

	_, err := r.db.ExecContext(ctx, query,
		pair.ID, pair.UserID, pair.ExchangeAccountID, pair.Symbol, pair.MarketType,
		pair.Budget, pair.MaxPositionSize, pair.MaxLeverage, pair.StopLossPercent, pair.TakeProfitPercent,
		pair.AIProvider, pair.StrategyMode, pq.Array(pair.Timeframes),
		pair.IsActive, pair.IsPaused, pair.PausedReason, pair.CreatedAt, pair.UpdatedAt,
	)

	return err
}

// GetByID retrieves a trading pair by ID
func (r *TradingPairRepository) GetByID(ctx context.Context, id uuid.UUID) (*trading_pair.TradingPair, error) {
	var pair trading_pair.TradingPair

	query := `SELECT * FROM trading_pairs WHERE id = $1`

	err := r.db.GetContext(ctx, &pair, query, id)
	if err != nil {
		return nil, err
	}

	return &pair, nil
}

// GetByUser retrieves all trading pairs for a user
func (r *TradingPairRepository) GetByUser(ctx context.Context, userID uuid.UUID) ([]*trading_pair.TradingPair, error) {
	var pairs []*trading_pair.TradingPair

	query := `SELECT * FROM trading_pairs WHERE user_id = $1 ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &pairs, query, userID)
	if err != nil {
		return nil, err
	}

	return pairs, nil
}

// GetActiveByUser retrieves active trading pairs for a user
func (r *TradingPairRepository) GetActiveByUser(ctx context.Context, userID uuid.UUID) ([]*trading_pair.TradingPair, error) {
	var pairs []*trading_pair.TradingPair

	query := `
		SELECT * FROM trading_pairs 
		WHERE user_id = $1 AND is_active = true AND is_paused = false
		ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &pairs, query, userID)
	if err != nil {
		return nil, err
	}

	return pairs, nil
}

// GetActiveBySymbol retrieves active trading pairs for a specific symbol across all users
// This is used for event-driven analysis when an opportunity is detected on a symbol
func (r *TradingPairRepository) GetActiveBySymbol(ctx context.Context, symbol string) ([]*trading_pair.TradingPair, error) {
	var pairs []*trading_pair.TradingPair

	query := `
		SELECT * FROM trading_pairs 
		WHERE symbol = $1 AND is_active = true AND is_paused = false
		ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &pairs, query, symbol)
	if err != nil {
		return nil, err
	}

	return pairs, nil
}

// FindAllActive retrieves ALL active trading pairs from ALL users
func (r *TradingPairRepository) FindAllActive(ctx context.Context) ([]*trading_pair.TradingPair, error) {
	var pairs []*trading_pair.TradingPair

	query := `
		SELECT * FROM trading_pairs 
		WHERE is_active = true AND is_paused = false
		ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &pairs, query)
	if err != nil {
		return nil, err
	}

	return pairs, nil
}

// Update updates a trading pair
func (r *TradingPairRepository) Update(ctx context.Context, pair *trading_pair.TradingPair) error {
	query := `
		UPDATE trading_pairs SET
			budget = $2,
			max_position_size = $3,
			max_leverage = $4,
			stop_loss_percent = $5,
			take_profit_percent = $6,
			ai_provider = $7,
			strategy_mode = $8,
			timeframes = $9,
			is_active = $10,
			is_paused = $11,
			paused_reason = $12,
			updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		pair.ID, pair.Budget, pair.MaxPositionSize, pair.MaxLeverage,
		pair.StopLossPercent, pair.TakeProfitPercent,
		pair.AIProvider, pair.StrategyMode, pq.Array(pair.Timeframes),
		pair.IsActive, pair.IsPaused, pair.PausedReason,
	)

	return err
}

// Pause pauses a trading pair
func (r *TradingPairRepository) Pause(ctx context.Context, id uuid.UUID, reason string) error {
	query := `UPDATE trading_pairs SET is_paused = true, paused_reason = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, reason)
	return err
}

// Resume resumes a trading pair
func (r *TradingPairRepository) Resume(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE trading_pairs SET is_paused = false, paused_reason = NULL, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// Disable disables a trading pair (sets is_active to false)
func (r *TradingPairRepository) Disable(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE trading_pairs SET is_active = false, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// Delete deletes a trading pair
func (r *TradingPairRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM trading_pairs WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
