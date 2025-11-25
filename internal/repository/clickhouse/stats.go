package clickhouse

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"

	"prometheus/internal/domain/stats"
)

// Compile-time check
var _ stats.Repository = (*StatsRepository)(nil)

// StatsRepository implements stats.Repository using ClickHouse
type StatsRepository struct {
	conn driver.Conn
}

// NewStatsRepository creates a new stats repository
func NewStatsRepository(conn driver.Conn) *StatsRepository {
	return &StatsRepository{conn: conn}
}

// InsertToolUsage inserts a single tool usage event
func (r *StatsRepository) InsertToolUsage(ctx context.Context, event *stats.ToolUsageEvent) error {
	query := `
		INSERT INTO tool_usage_stats (
			user_id, agent_id, tool_name, timestamp,
			duration_ms, success, session_id, symbol
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)`

	return r.conn.Exec(ctx, query,
		event.UserID, event.AgentID, event.ToolName, event.Timestamp,
		event.DurationMs, event.Success, event.SessionID, event.Symbol,
	)
}

// InsertToolUsageBatch inserts multiple tool usage events
func (r *StatsRepository) InsertToolUsageBatch(ctx context.Context, events []stats.ToolUsageEvent) error {
	if len(events) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO tool_usage_stats (
			user_id, agent_id, tool_name, timestamp,
			duration_ms, success, session_id, symbol
		)
	`)
	if err != nil {
		return err
	}

	for _, event := range events {
		err := batch.Append(
			event.UserID, event.AgentID, event.ToolName, event.Timestamp,
			event.DurationMs, event.Success, event.SessionID, event.Symbol,
		)
		if err != nil {
			return err
		}
	}

	return batch.Send()
}

// GetByUser retrieves aggregated stats for a user from materialized view
func (r *StatsRepository) GetByUser(ctx context.Context, userID uuid.UUID, since time.Time) ([]stats.ToolUsageAggregated, error) {
	var usage []stats.ToolUsageAggregated

	query := `
		SELECT * FROM tool_usage_hourly_mv
		WHERE user_id = $1 AND hour >= $2
		ORDER BY hour DESC`

	err := r.conn.Select(ctx, &usage, query, userID, since)
	return usage, err
}

// GetByAgent retrieves aggregated stats for an agent
func (r *StatsRepository) GetByAgent(ctx context.Context, agentID string, since time.Time) ([]stats.ToolUsageAggregated, error) {
	var usage []stats.ToolUsageAggregated

	query := `
		SELECT * FROM tool_usage_hourly_mv
		WHERE agent_id = $1 AND hour >= $2
		ORDER BY hour DESC`

	err := r.conn.Select(ctx, &usage, query, agentID, since)
	return usage, err
}

// GetByTool retrieves aggregated stats for a specific tool
func (r *StatsRepository) GetByTool(ctx context.Context, toolName string, since time.Time) ([]stats.ToolUsageAggregated, error) {
	var usage []stats.ToolUsageAggregated

	query := `
		SELECT * FROM tool_usage_hourly_mv
		WHERE tool_name = $1 AND hour >= $2
		ORDER BY hour DESC`

	err := r.conn.Select(ctx, &usage, query, toolName, since)
	return usage, err
}

// GetTopTools retrieves most used tools for a user
func (r *StatsRepository) GetTopTools(ctx context.Context, userID uuid.UUID, since time.Time, limit int) ([]stats.ToolUsageAggregated, error) {
	var usage []stats.ToolUsageAggregated

	query := `
		SELECT 
			user_id,
			agent_id,
			tool_name,
			max(hour) as hour,
			sum(call_count) as call_count,
			sum(total_duration_ms) as total_duration_ms,
			sum(success_count) as success_count,
			sum(error_count) as error_count,
			avg(avg_duration_ms) as avg_duration_ms
		FROM tool_usage_hourly_mv
		WHERE user_id = $1 AND hour >= $2
		GROUP BY user_id, agent_id, tool_name
		ORDER BY call_count DESC
		LIMIT $3`

	err := r.conn.Select(ctx, &usage, query, userID, since, limit)
	return usage, err
}
