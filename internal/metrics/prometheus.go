package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Worker metrics
	WorkerExecutions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prometheus_worker_executions_total",
			Help: "Total number of worker executions",
		},
		[]string{"worker", "status"}, // status: success|error
	)

	WorkerDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "prometheus_worker_duration_seconds",
			Help:    "Worker execution duration in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
		},
		[]string{"worker"},
	)

	WorkerLastRun = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "prometheus_worker_last_run_timestamp",
			Help: "Unix timestamp of last worker execution",
		},
		[]string{"worker"},
	)

	// Agent metrics
	AgentCalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prometheus_agent_calls_total",
			Help: "Total number of agent calls",
		},
		[]string{"agent", "model", "status"}, // status: success|error|rate_limited
	)

	AgentCost = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prometheus_agent_cost_usd",
			Help: "Total AI cost in USD",
		},
		[]string{"agent", "user_id", "model"},
	)

	AgentLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "prometheus_agent_latency_seconds",
			Help:    "Agent execution latency in seconds",
			Buckets: []float64{0.5, 1, 2, 5, 10, 20, 30, 60},
		},
		[]string{"agent", "model"},
	)

	AgentTokens = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prometheus_agent_tokens_total",
			Help: "Total tokens used by agents",
		},
		[]string{"agent", "model", "type"}, // type: input|output
	)

	// Exchange metrics
	ExchangeAPICalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prometheus_exchange_api_calls_total",
			Help: "Total number of exchange API calls",
		},
		[]string{"exchange", "endpoint", "status"}, // status: success|error|rate_limited
	)

	ExchangeAPIErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prometheus_exchange_api_errors_total",
			Help: "Total number of exchange API errors",
		},
		[]string{"exchange", "error_type"},
	)

	ExchangeAPILatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "prometheus_exchange_api_latency_seconds",
			Help:    "Exchange API latency in seconds",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		},
		[]string{"exchange", "endpoint"},
	)

	// Risk metrics
	CircuitBreakerTrips = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prometheus_circuit_breaker_trips_total",
			Help: "Total number of circuit breaker activations",
		},
		[]string{"user_id", "reason"},
	)

	PositionsOpen = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "prometheus_positions_open_count",
			Help: "Current number of open positions",
		},
		[]string{"user_id", "exchange", "symbol"},
	)

	OrdersActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "prometheus_orders_active_count",
			Help: "Current number of active orders",
		},
		[]string{"user_id", "exchange"},
	)

	UserDrawdown = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "prometheus_user_drawdown_percent",
			Help: "Current drawdown percentage per user",
		},
		[]string{"user_id", "exchange"},
	)

	// Tool metrics
	ToolExecutions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prometheus_tool_executions_total",
			Help: "Total number of tool executions",
		},
		[]string{"tool", "status"}, // status: success|error
	)

	ToolLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "prometheus_tool_latency_seconds",
			Help:    "Tool execution latency in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10},
		},
		[]string{"tool"},
	)

	// Database metrics
	DBQueries = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prometheus_db_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"database", "operation", "status"}, // database: postgres|clickhouse|redis
	)

	DBQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "prometheus_db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2},
		},
		[]string{"database", "operation"},
	)

	// System metrics
	KafkaMessages = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prometheus_kafka_messages_total",
			Help: "Total Kafka messages produced/consumed",
		},
		[]string{"topic", "direction", "status"}, // direction: produced|consumed
	)

	WebSocketConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "prometheus_websocket_connections",
			Help: "Current number of active WebSocket connections",
		},
		[]string{"exchange", "channel"},
	)

	// User Data WebSocket metrics (per-user connections)
	UserDataConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "prometheus_userdata_connections",
			Help: "Current number of active User Data WebSocket connections",
		},
		[]string{"exchange"},
	)

	UserDataReconnects = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prometheus_userdata_reconnects_total",
			Help: "Total number of User Data WebSocket reconnections",
		},
		[]string{"exchange", "reason"},
	)

	UserDataReconciliations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prometheus_userdata_reconciliations_total",
			Help: "Total number of reconciliation cycles",
		},
		[]string{"status"}, // status: success|error
	)

	UserDataHotReload = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prometheus_userdata_hotreload_operations_total",
			Help: "Total number of hot reload operations (add/remove accounts)",
		},
		[]string{"operation", "exchange"}, // operation: add|remove
	)

	UserDataListenKeyRenewals = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prometheus_userdata_listenkey_renewals_total",
			Help: "Total number of listenKey renewal operations",
		},
		[]string{"exchange", "status"}, // status: success|error
	)

	// Market Data WebSocket metrics (centralized market data streams)
	MarketDataReconnects = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prometheus_marketdata_reconnects_total",
			Help: "Total number of Market Data WebSocket reconnect attempts",
		},
		[]string{"status"}, // status: success|failed
	)
)

// Init registers all metrics with Prometheus
func Init() {
	// Worker metrics
	prometheus.MustRegister(WorkerExecutions)
	prometheus.MustRegister(WorkerDuration)
	prometheus.MustRegister(WorkerLastRun)

	// Agent metrics
	prometheus.MustRegister(AgentCalls)
	prometheus.MustRegister(AgentCost)
	prometheus.MustRegister(AgentLatency)
	prometheus.MustRegister(AgentTokens)

	// Exchange metrics
	prometheus.MustRegister(ExchangeAPICalls)
	prometheus.MustRegister(ExchangeAPIErrors)
	prometheus.MustRegister(ExchangeAPILatency)

	// Risk metrics
	prometheus.MustRegister(CircuitBreakerTrips)
	prometheus.MustRegister(PositionsOpen)
	prometheus.MustRegister(OrdersActive)
	prometheus.MustRegister(UserDrawdown)

	// Tool metrics
	prometheus.MustRegister(ToolExecutions)
	prometheus.MustRegister(ToolLatency)

	// Database metrics
	prometheus.MustRegister(DBQueries)
	prometheus.MustRegister(DBQueryDuration)

	// System metrics
	prometheus.MustRegister(KafkaMessages)
	prometheus.MustRegister(WebSocketConnections)

	// User Data WebSocket metrics
	prometheus.MustRegister(UserDataConnections)
	prometheus.MustRegister(UserDataReconnects)
	prometheus.MustRegister(UserDataReconciliations)
	prometheus.MustRegister(UserDataHotReload)
	prometheus.MustRegister(UserDataListenKeyRenewals)

	// Market Data WebSocket metrics
	prometheus.MustRegister(MarketDataReconnects)
}

// Handler returns Prometheus HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// RecordWorkerExecution records a worker execution
func RecordWorkerExecution(worker string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	WorkerExecutions.WithLabelValues(worker, status).Inc()
	WorkerDuration.WithLabelValues(worker).Observe(duration.Seconds())
	WorkerLastRun.WithLabelValues(worker).SetToCurrentTime()
}

// RecordAgentCall records an agent invocation
func RecordAgentCall(agent, model string, latency time.Duration, cost float64, tokens int, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	AgentCalls.WithLabelValues(agent, model, status).Inc()
	AgentLatency.WithLabelValues(agent, model).Observe(latency.Seconds())

	if cost > 0 {
		AgentCost.WithLabelValues(agent, "", model).Add(cost)
	}

	if tokens > 0 {
		AgentTokens.WithLabelValues(agent, model, "total").Add(float64(tokens))
	}
}

// RecordExchangeAPICall records an exchange API call
func RecordExchangeAPICall(exchange, endpoint string, latency time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	ExchangeAPICalls.WithLabelValues(exchange, endpoint, status).Inc()
	ExchangeAPILatency.WithLabelValues(exchange, endpoint).Observe(latency.Seconds())

	if err != nil {
		ExchangeAPIErrors.WithLabelValues(exchange, "unknown").Inc()
	}
}

// RecordToolExecution records a tool execution
func RecordToolExecution(tool string, latency time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	ToolExecutions.WithLabelValues(tool, status).Inc()
	ToolLatency.WithLabelValues(tool).Observe(latency.Seconds())
}

// RecordDBQuery records a database query
func RecordDBQuery(database, operation string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	DBQueries.WithLabelValues(database, operation, status).Inc()
	DBQueryDuration.WithLabelValues(database, operation).Observe(duration.Seconds())
}
