package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"prometheus/internal/domain/journal"
)

// Compile-time check
var _ journal.Repository = (*JournalRepository)(nil)

// JournalRepository implements journal.Repository using sqlx
type JournalRepository struct {
	db *sqlx.DB
}

// NewJournalRepository creates a new journal repository
func NewJournalRepository(db *sqlx.DB) *JournalRepository {
	return &JournalRepository{db: db}
}

// Create inserts a new journal entry
func (r *JournalRepository) Create(ctx context.Context, entry *journal.JournalEntry) error {
	query := `
		INSERT INTO journal_entries (
			id, user_id, trade_id, symbol, side,
			entry_price, exit_price, size, pnl, pnl_percent,
			strategy_used, timeframe, setup_type,
			market_regime, entry_reasoning, exit_reasoning, confidence_score,
			rsi_at_entry, atr_at_entry, volume_at_entry,
			was_correct_entry, was_correct_exit, max_drawdown, max_profit, hold_duration,
			lessons_learned, improvement_tips, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28
		)`

	_, err := r.db.ExecContext(ctx, query,
		entry.ID, entry.UserID, entry.TradeID, entry.Symbol, entry.Side,
		entry.EntryPrice, entry.ExitPrice, entry.Size, entry.PnL, entry.PnLPercent,
		entry.StrategyUsed, entry.Timeframe, entry.SetupType,
		entry.MarketRegime, entry.EntryReasoning, entry.ExitReasoning, entry.ConfidenceScore,
		entry.RSIAtEntry, entry.ATRAtEntry, entry.VolumeAtEntry,
		entry.WasCorrectEntry, entry.WasCorrectExit, entry.MaxDrawdown, entry.MaxProfit, entry.HoldDuration,
		entry.LessonsLearned, entry.ImprovementTips, entry.CreatedAt,
	)

	return err
}

// GetByID retrieves a journal entry by ID
func (r *JournalRepository) GetByID(ctx context.Context, id uuid.UUID) (*journal.JournalEntry, error) {
	var entry journal.JournalEntry

	query := `SELECT * FROM journal_entries WHERE id = $1`

	err := r.db.GetContext(ctx, &entry, query, id)
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// GetEntriesSince retrieves journal entries since a specific time
func (r *JournalRepository) GetEntriesSince(ctx context.Context, userID uuid.UUID, since time.Time) ([]journal.JournalEntry, error) {
	var entries []journal.JournalEntry

	query := `
		SELECT * FROM journal_entries
		WHERE user_id = $1 AND created_at >= $2
		ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &entries, query, userID, since)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// GetStrategyStats calculates aggregated strategy performance
func (r *JournalRepository) GetStrategyStats(ctx context.Context, userID uuid.UUID, since time.Time) ([]journal.StrategyStats, error) {
	var stats []journal.StrategyStats

	query := `
		SELECT
			strategy_used as strategy_name,
			COUNT(*) as total_trades,
			SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END) as winning_trades,
			SUM(CASE WHEN pnl <= 0 THEN 1 ELSE 0 END) as losing_trades,
			ROUND(100.0 * SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END)::DECIMAL / COUNT(*), 2) as win_rate,
			COALESCE(AVG(CASE WHEN pnl > 0 THEN pnl END), 0) as avg_win,
			COALESCE(AVG(CASE WHEN pnl < 0 THEN ABS(pnl) END), 0) as avg_loss,
			CASE
				WHEN SUM(CASE WHEN pnl < 0 THEN ABS(pnl) END) > 0
				THEN SUM(CASE WHEN pnl > 0 THEN pnl END) / SUM(CASE WHEN pnl < 0 THEN ABS(pnl) END)
				ELSE 0
			END as profit_factor,
			(
				(SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END)::DECIMAL / COUNT(*)) * AVG(CASE WHEN pnl > 0 THEN pnl END) -
				(SUM(CASE WHEN pnl <= 0 THEN 1 ELSE 0 END)::DECIMAL / COUNT(*)) * AVG(CASE WHEN pnl < 0 THEN ABS(pnl) END)
			) as expected_value,
			true as is_active,
			MAX(created_at) as last_updated
		FROM journal_entries
		WHERE user_id = $1 AND created_at >= $2
		GROUP BY strategy_used
		ORDER BY total_trades DESC`

	err := r.db.SelectContext(ctx, &stats, query, userID, since)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// GetByStrategy retrieves entries for a specific strategy
func (r *JournalRepository) GetByStrategy(ctx context.Context, userID uuid.UUID, strategy string, limit int) ([]journal.JournalEntry, error) {
	var entries []journal.JournalEntry

	query := `
		SELECT * FROM journal_entries
		WHERE user_id = $1 AND strategy_used = $2
		ORDER BY created_at DESC
		LIMIT $3`

	err := r.db.SelectContext(ctx, &entries, query, userID, strategy, limit)
	if err != nil {
		return nil, err
	}

	return entries, nil
}
