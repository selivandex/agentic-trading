package stats

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for tool usage statistics data access (ClickHouse)
type Repository interface {
	// Insert tool usage event
	InsertToolUsage(ctx context.Context, event *ToolUsageEvent) error
	InsertToolUsageBatch(ctx context.Context, events []ToolUsageEvent) error

	// Get aggregated stats from materialized view
	GetByUser(ctx context.Context, userID uuid.UUID, since time.Time) ([]ToolUsageAggregated, error)
	GetByAgent(ctx context.Context, agentID string, since time.Time) ([]ToolUsageAggregated, error)
	GetByTool(ctx context.Context, toolName string, since time.Time) ([]ToolUsageAggregated, error)
	GetTopTools(ctx context.Context, userID uuid.UUID, since time.Time, limit int) ([]ToolUsageAggregated, error)
}

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for tool usage statistics data access (ClickHouse)
type Repository interface {
	// Insert tool usage event
	InsertToolUsage(ctx context.Context, event *ToolUsageEvent) error
	InsertToolUsageBatch(ctx context.Context, events []ToolUsageEvent) error

	// Get aggregated stats from materialized view
	GetByUser(ctx context.Context, userID uuid.UUID, since time.Time) ([]ToolUsageAggregated, error)
	GetByAgent(ctx context.Context, agentID string, since time.Time) ([]ToolUsageAggregated, error)
	GetByTool(ctx context.Context, toolName string, since time.Time) ([]ToolUsageAggregated, error)
	GetTopTools(ctx context.Context, userID uuid.UUID, since time.Time, limit int) ([]ToolUsageAggregated, error)
}
