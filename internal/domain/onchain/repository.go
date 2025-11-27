package onchain

import (
	"context"
	"time"
)

// Repository defines the interface for on-chain data access (ClickHouse)
type Repository interface {
	// Whale movement operations
	InsertWhaleMovement(ctx context.Context, movement *WhaleMovement) error
	InsertWhaleMovementBatch(ctx context.Context, movements []WhaleMovement) error
	GetRecentWhaleMovements(ctx context.Context, minAmount float64, limit int) ([]WhaleMovement, error)
	GetWhaleMovementsByBlockchain(ctx context.Context, blockchain string, since time.Time) ([]WhaleMovement, error)
	GetWhaleMovementsByToken(ctx context.Context, token string, minAmount float64, limit int) ([]WhaleMovement, error)

	// Exchange flow operations
	InsertExchangeFlow(ctx context.Context, flow *ExchangeFlow) error
	InsertExchangeFlowBatch(ctx context.Context, flows []ExchangeFlow) error
	GetExchangeFlows(ctx context.Context, exchange string, since time.Time) ([]ExchangeFlow, error)
	GetExchangeFlowsByToken(ctx context.Context, token string, since time.Time) ([]ExchangeFlow, error)
	GetNetFlowSummary(ctx context.Context, since time.Time) ([]ExchangeFlow, error)

	// Network metrics operations
	InsertNetworkMetrics(ctx context.Context, metrics *NetworkMetrics) error
	GetLatestNetworkMetrics(ctx context.Context, blockchain string) (*NetworkMetrics, error)
	GetNetworkMetricsHistory(ctx context.Context, blockchain string, since time.Time) ([]NetworkMetrics, error)

	// Miner metrics operations
	InsertMinerMetrics(ctx context.Context, metrics *MinerMetrics) error
	InsertMinerMetricsBatch(ctx context.Context, metrics []MinerMetrics) error
	GetMinerActivity(ctx context.Context, since time.Time) ([]MinerMetrics, error)
	GetMinerMetricsByPool(ctx context.Context, poolName string, since time.Time) ([]MinerMetrics, error)
	GetLatestMinerMetrics(ctx context.Context) ([]MinerMetrics, error)
}


