package clickhouse

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"prometheus/internal/domain/ai_usage"
	"prometheus/pkg/clickhouse"
	"prometheus/pkg/errors"
)

// AIUsageRepository implements ai_usage.Repository for ClickHouse
// Uses batch writer for efficient bulk inserts
type AIUsageRepository struct {
	conn        driver.Conn
	batchWriter *clickhouse.BatchWriter
}

// NewAIUsageRepository creates a new AI usage repository with batch writer
func NewAIUsageRepository(conn driver.Conn) *AIUsageRepository {
	repo := &AIUsageRepository{
		conn: conn,
	}

	// Create batch writer with flush function
	repo.batchWriter = clickhouse.NewBatchWriter(clickhouse.BatchWriterConfig{
		Conn:         conn,
		FlushFunc:    repo.flushBatch,
		TableName:    "ai_usage",
		MaxBatchSize: 500,             // Flush every 500 records
		MaxAge:       5 * time.Second, // Or every 5 seconds
	})

	return repo
}

// Start begins the background flush loop
func (r *AIUsageRepository) Start(ctx context.Context) {
	r.batchWriter.Start(ctx)
}

// Stop gracefully shuts down the batch writer
func (r *AIUsageRepository) Stop(ctx context.Context) error {
	return r.batchWriter.Stop(ctx)
}

// Store saves a usage log entry (buffered, not immediate)
func (r *AIUsageRepository) Store(ctx context.Context, log *ai_usage.UsageLog) error {
	return r.batchWriter.Add(ctx, log)
}

// flushBatch performs the actual batch insert to ClickHouse
func (r *AIUsageRepository) flushBatch(ctx context.Context, batch []interface{}) error {
	if len(batch) == 0 {
		return nil
	}

	query := `
		INSERT INTO ai_usage (
			timestamp, event_id, user_id, session_id,
			agent_name, agent_type,
			provider, model_id, model_family,
			prompt_tokens, completion_tokens, total_tokens,
			input_cost_usd, output_cost_usd, total_cost_usd,
			tool_calls_count, is_cached, cache_hit,
			latency_ms, reasoning_step, workflow_name, created_at
		) VALUES (
			?, ?, ?, ?,
			?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?, ?, ?
		)
	`

	// Prepare batch statement
	stmt, err := r.conn.PrepareBatch(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to prepare batch")
	}
	defer stmt.Close()

	// Append all items to batch
	for _, item := range batch {
		log, ok := item.(*ai_usage.UsageLog)
		if !ok {
			continue // Skip invalid items
		}

		err := stmt.Append(
			log.Timestamp, log.EventID, log.UserID, log.SessionID,
			log.AgentName, log.AgentType,
			log.Provider, log.ModelID, log.ModelFamily,
			log.PromptTokens, log.CompletionTokens, log.TotalTokens,
			log.InputCostUSD, log.OutputCostUSD, log.TotalCostUSD,
			log.ToolCallsCount, log.IsCached, log.CacheHit,
			log.LatencyMs, log.ReasoningStep, log.WorkflowName, log.CreatedAt,
		)

		if err != nil {
			return errors.Wrap(err, "failed to append to batch")
		}
	}

	// Execute batch
	if err := stmt.Send(); err != nil {
		return errors.Wrap(err, "failed to send batch")
	}

	return nil
}

// GetUserDailyCost returns total cost for a user on a specific day
func (r *AIUsageRepository) GetUserDailyCost(ctx context.Context, userID string, date time.Time) (float64, error) {
	query := `
		SELECT sum(total_cost_usd) as total_cost
		FROM ai_usage
		WHERE user_id = ? AND toDate(timestamp) = toDate(?)
	`

	var totalCost float64
	err := r.conn.QueryRow(ctx, query, userID, date).Scan(&totalCost)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get user daily cost")
	}

	return totalCost, nil
}

// GetUserMonthlyCost returns total cost for a user in a specific month
func (r *AIUsageRepository) GetUserMonthlyCost(ctx context.Context, userID string, year int, month int) (float64, error) {
	query := `
		SELECT sum(total_cost_usd) as total_cost
		FROM ai_usage
		WHERE user_id = ? 
		  AND toYear(timestamp) = ? 
		  AND toMonth(timestamp) = ?
	`

	var totalCost float64
	err := r.conn.QueryRow(ctx, query, userID, year, month).Scan(&totalCost)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get user monthly cost")
	}

	return totalCost, nil
}

// GetProviderCosts returns costs grouped by provider for a time range
func (r *AIUsageRepository) GetProviderCosts(ctx context.Context, from, to time.Time) (map[string]float64, error) {
	query := `
		SELECT provider, sum(total_cost_usd) as total_cost
		FROM ai_usage
		WHERE timestamp BETWEEN ? AND ?
		GROUP BY provider
		ORDER BY total_cost DESC
	`

	rows, err := r.conn.Query(ctx, query, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query provider costs")
	}
	defer rows.Close()

	costs := make(map[string]float64)
	for rows.Next() {
		var provider string
		var cost float64
		if err := rows.Scan(&provider, &cost); err != nil {
			return nil, errors.Wrap(err, "failed to scan provider cost")
		}
		costs[provider] = cost
	}

	return costs, nil
}

// GetAgentCosts returns costs grouped by agent type for a time range
func (r *AIUsageRepository) GetAgentCosts(ctx context.Context, from, to time.Time) (map[string]float64, error) {
	query := `
		SELECT agent_type, sum(total_cost_usd) as total_cost
		FROM ai_usage
		WHERE timestamp BETWEEN ? AND ?
		GROUP BY agent_type
		ORDER BY total_cost DESC
	`

	rows, err := r.conn.Query(ctx, query, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query agent costs")
	}
	defer rows.Close()

	costs := make(map[string]float64)
	for rows.Next() {
		var agentType string
		var cost float64
		if err := rows.Scan(&agentType, &cost); err != nil {
			return nil, errors.Wrap(err, "failed to scan agent cost")
		}
		costs[agentType] = cost
	}

	return costs, nil
}

// GetModelCosts returns costs grouped by model for a provider in a time range
func (r *AIUsageRepository) GetModelCosts(ctx context.Context, provider string, from, to time.Time) (map[string]float64, error) {
	query := `
		SELECT model_id, sum(total_cost_usd) as total_cost
		FROM ai_usage
		WHERE provider = ? AND timestamp BETWEEN ? AND ?
		GROUP BY model_id
		ORDER BY total_cost DESC
	`

	rows, err := r.conn.Query(ctx, query, provider, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query model costs")
	}
	defer rows.Close()

	costs := make(map[string]float64)
	for rows.Next() {
		var modelID string
		var cost float64
		if err := rows.Scan(&modelID, &cost); err != nil {
			return nil, errors.Wrap(err, "failed to scan model cost")
		}
		costs[modelID] = cost
	}

	return costs, nil
}
