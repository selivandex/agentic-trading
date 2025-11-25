package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"prometheus/internal/domain/reasoning"
)

// Compile-time check
var _ reasoning.Repository = (*ReasoningRepository)(nil)

// ReasoningRepository implements reasoning.Repository using sqlx
type ReasoningRepository struct {
	db *sqlx.DB
}

// NewReasoningRepository creates a new reasoning repository
func NewReasoningRepository(db *sqlx.DB) *ReasoningRepository {
	return &ReasoningRepository{db: db}
}

// Create inserts a new reasoning log entry
func (r *ReasoningRepository) Create(ctx context.Context, entry *reasoning.LogEntry) error {
	query := `
		INSERT INTO agent_reasoning_logs (
			id, user_id, agent_id, session_id, symbol, trading_pair_id,
			reasoning_steps, decision, confidence,
			tokens_used, cost_usd, duration_ms, tool_calls_count,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)`

	_, err := r.db.ExecContext(ctx, query,
		entry.ID, entry.UserID, entry.AgentID, entry.SessionID, entry.Symbol, entry.TradingPairID,
		entry.ReasoningSteps, entry.Decision, entry.Confidence,
		entry.TokensUsed, entry.CostUSD, entry.DurationMs, entry.ToolCallsCount,
		entry.CreatedAt,
	)

	return err
}

// GetByID retrieves a reasoning log by ID
func (r *ReasoningRepository) GetByID(ctx context.Context, id uuid.UUID) (*reasoning.LogEntry, error) {
	var entry reasoning.LogEntry

	query := `SELECT * FROM agent_reasoning_logs WHERE id = $1`

	err := r.db.GetContext(ctx, &entry, query, id)
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// GetBySession retrieves all reasoning logs for a session
func (r *ReasoningRepository) GetBySession(ctx context.Context, sessionID string) ([]*reasoning.LogEntry, error) {
	var entries []*reasoning.LogEntry

	query := `
		SELECT * FROM agent_reasoning_logs
		WHERE session_id = $1
		ORDER BY created_at ASC`

	err := r.db.SelectContext(ctx, &entries, query, sessionID)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// GetByUser retrieves reasoning logs for a user
func (r *ReasoningRepository) GetByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*reasoning.LogEntry, error) {
	var entries []*reasoning.LogEntry

	query := `
		SELECT * FROM agent_reasoning_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	err := r.db.SelectContext(ctx, &entries, query, userID, limit)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// GetByAgent retrieves reasoning logs for an agent
func (r *ReasoningRepository) GetByAgent(ctx context.Context, agentID string, limit int) ([]*reasoning.LogEntry, error) {
	var entries []*reasoning.LogEntry

	query := `
		SELECT * FROM agent_reasoning_logs
		WHERE agent_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	err := r.db.SelectContext(ctx, &entries, query, agentID, limit)
	if err != nil {
		return nil, err
	}

	return entries, nil
}
