package metrics

import (
	"context"
	"time"

	"prometheus/pkg/logger"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

// CustomCollector collects custom metrics from databases
type CustomCollector struct {
	log        *logger.Logger
	postgres   *sqlx.DB
	clickhouse driver.Conn
	redis      *redis.Client

	// Descriptors
	totalUsers          *prometheus.Desc
	totalPositions      *prometheus.Desc
	totalOrders         *prometheus.Desc
	totalTrades         *prometheus.Desc
	circuitBreakerState *prometheus.Desc
	memoryUsage         *prometheus.Desc
}

// NewCustomCollector creates a new custom metrics collector
func NewCustomCollector(log *logger.Logger, postgres *sqlx.DB, clickhouse driver.Conn, redis *redis.Client) *CustomCollector {
	return &CustomCollector{
		log:        log,
		postgres:   postgres,
		clickhouse: clickhouse,
		redis:      redis,

		totalUsers: prometheus.NewDesc(
			"prometheus_total_users",
			"Total number of registered users",
			nil, nil,
		),
		totalPositions: prometheus.NewDesc(
			"prometheus_total_positions",
			"Total number of positions by status",
			[]string{"status"}, nil,
		),
		totalOrders: prometheus.NewDesc(
			"prometheus_total_orders",
			"Total number of orders by status",
			[]string{"status"}, nil,
		),
		totalTrades: prometheus.NewDesc(
			"prometheus_total_trades_24h",
			"Total trades executed in last 24h",
			nil, nil,
		),
		circuitBreakerState: prometheus.NewDesc(
			"prometheus_circuit_breaker_state",
			"Circuit breaker state (0=closed, 1=open)",
			[]string{"user_id"}, nil,
		),
		memoryUsage: prometheus.NewDesc(
			"prometheus_memory_usage_mb",
			"Memory usage in megabytes",
			[]string{"type"}, // type: user_memory|collective_memory
			nil,
		),
	}
}

// Describe implements prometheus.Collector
func (c *CustomCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalUsers
	ch <- c.totalPositions
	ch <- c.totalOrders
	ch <- c.totalTrades
	ch <- c.circuitBreakerState
	ch <- c.memoryUsage
}

// Collect implements prometheus.Collector
func (c *CustomCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Collect user count
	c.collectUserCount(ctx, ch)

	// Collect position stats
	c.collectPositionStats(ctx, ch)

	// Collect order stats
	c.collectOrderStats(ctx, ch)

	// Collect trade stats
	c.collectTradeStats(ctx, ch)

	// Collect circuit breaker states
	c.collectCircuitBreakerStates(ctx, ch)

	// Collect memory usage
	c.collectMemoryUsage(ctx, ch)
}

func (c *CustomCollector) collectUserCount(ctx context.Context, ch chan<- prometheus.Metric) {
	var count int
	err := c.postgres.GetContext(ctx, &count, "SELECT COUNT(*) FROM users")
	if err != nil {
		c.log.Error("Failed to collect user count metric", "error", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		c.totalUsers,
		prometheus.GaugeValue,
		float64(count),
	)
}

func (c *CustomCollector) collectPositionStats(ctx context.Context, ch chan<- prometheus.Metric) {
	type PositionStat struct {
		Status string `db:"status"`
		Count  int    `db:"count"`
	}

	var stats []PositionStat
	err := c.postgres.SelectContext(ctx, &stats, `
		SELECT status, COUNT(*) as count
		FROM positions
		GROUP BY status
	`)
	if err != nil {
		c.log.Error("Failed to collect position stats", "error", err)
		return
	}

	for _, stat := range stats {
		ch <- prometheus.MustNewConstMetric(
			c.totalPositions,
			prometheus.GaugeValue,
			float64(stat.Count),
			stat.Status,
		)
	}
}

func (c *CustomCollector) collectOrderStats(ctx context.Context, ch chan<- prometheus.Metric) {
	type OrderStat struct {
		Status string `db:"status"`
		Count  int    `db:"count"`
	}

	var stats []OrderStat
	err := c.postgres.SelectContext(ctx, &stats, `
		SELECT status, COUNT(*) as count
		FROM orders
		GROUP BY status
	`)
	if err != nil {
		c.log.Error("Failed to collect order stats", "error", err)
		return
	}

	for _, stat := range stats {
		ch <- prometheus.MustNewConstMetric(
			c.totalOrders,
			prometheus.GaugeValue,
			float64(stat.Count),
			stat.Status,
		)
	}
}

func (c *CustomCollector) collectTradeStats(ctx context.Context, ch chan<- prometheus.Metric) {
	var count int
	err := c.postgres.GetContext(ctx, &count, `
		SELECT COUNT(*)
		FROM orders
		WHERE status = 'filled'
		AND updated_at > NOW() - INTERVAL '24 hours'
	`)
	if err != nil {
		c.log.Error("Failed to collect trade stats", "error", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		c.totalTrades,
		prometheus.CounterValue,
		float64(count),
	)
}

func (c *CustomCollector) collectCircuitBreakerStates(ctx context.Context, ch chan<- prometheus.Metric) {
	type CircuitBreakerState struct {
		UserID string `db:"user_id"`
		IsOpen bool   `db:"is_open"`
	}

	var states []CircuitBreakerState
	err := c.postgres.SelectContext(ctx, &states, `
		SELECT user_id, is_open
		FROM circuit_breaker_state
		WHERE is_open = true
	`)
	if err != nil {
		c.log.Error("Failed to collect circuit breaker states", "error", err)
		return
	}

	for _, state := range states {
		value := 0.0
		if state.IsOpen {
			value = 1.0
		}
		ch <- prometheus.MustNewConstMetric(
			c.circuitBreakerState,
			prometheus.GaugeValue,
			value,
			state.UserID,
		)
	}
}

func (c *CustomCollector) collectMemoryUsage(ctx context.Context, ch chan<- prometheus.Metric) {
	// Collect user memory count
	var userMemCount int
	err := c.postgres.GetContext(ctx, &userMemCount, "SELECT COUNT(*) FROM memories WHERE user_id IS NOT NULL")
	if err == nil {
		ch <- prometheus.MustNewConstMetric(
			c.memoryUsage,
			prometheus.GaugeValue,
			float64(userMemCount),
			"user_memory",
		)
	}

	// Collect collective memory count
	var collectiveMemCount int
	err = c.postgres.GetContext(ctx, &collectiveMemCount, "SELECT COUNT(*) FROM collective_memory")
	if err == nil {
		ch <- prometheus.MustNewConstMetric(
			c.memoryUsage,
			prometheus.GaugeValue,
			float64(collectiveMemCount),
			"collective_memory",
		)
	}
}

// RegisterCustomCollector registers the custom collector
func RegisterCustomCollector(collector *CustomCollector) {
	prometheus.MustRegister(collector)
}

