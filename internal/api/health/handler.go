package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"prometheus/pkg/logger"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

// Handler provides health check endpoints
type Handler struct {
	log         *logger.Logger
	postgres    *sqlx.DB
	clickhouse  driver.Conn
	redis       *redis.Client
	startTime   time.Time
	serviceName string
	version     string
}

// New creates a new health check handler
func New(
	log *logger.Logger,
	postgres *sqlx.DB,
	clickhouse driver.Conn,
	redis *redis.Client,
	serviceName string,
	version string,
) *Handler {
	return &Handler{
		log:         log,
		postgres:    postgres,
		clickhouse:  clickhouse,
		redis:       redis,
		startTime:   time.Now(),
		serviceName: serviceName,
		version:     version,
	}
}

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status      string                     `json:"status"` // "healthy", "degraded", "unhealthy"
	Service     string                     `json:"service"`
	Version     string                     `json:"version"`
	Uptime      string                     `json:"uptime"`
	Timestamp   string                     `json:"timestamp"`
	Checks      map[string]ComponentHealth `json:"checks"`
	ErrorDetail string                     `json:"error_detail,omitempty"`
}

// ComponentHealth represents health of a single component
type ComponentHealth struct {
	Status       string `json:"status"`
	ResponseTime string `json:"response_time,omitempty"`
	Error        string `json:"error,omitempty"`
}

// HandleLiveness returns 200 OK if service is running
// Used by Kubernetes liveness probe
func (h *Handler) HandleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "alive",
	})
}

// HandleReadiness checks if service is ready to accept traffic
// Used by Kubernetes readiness probe
func (h *Handler) HandleReadiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]ComponentHealth)
	allHealthy := true

	// Check PostgreSQL
	pgHealth := h.checkPostgres(ctx)
	checks["postgres"] = pgHealth
	if pgHealth.Status != "healthy" {
		allHealthy = false
	}

	// Check ClickHouse
	chHealth := h.checkClickHouse(ctx)
	checks["clickhouse"] = chHealth
	if chHealth.Status != "healthy" {
		allHealthy = false
	}

	// Check Redis
	redisHealth := h.checkRedis(ctx)
	checks["redis"] = redisHealth
	if redisHealth.Status != "healthy" {
		allHealthy = false
	}

	status := HealthStatus{
		Status:    "healthy",
		Service:   h.serviceName,
		Version:   h.version,
		Uptime:    time.Since(h.startTime).String(),
		Timestamp: time.Now().Format(time.RFC3339),
		Checks:    checks,
	}

	statusCode := http.StatusOK
	if !allHealthy {
		status.Status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
		h.log.Warn("Readiness check failed", "checks", checks)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(status)
}

// HandleHealth returns detailed health status (includes all checks)
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	checks := make(map[string]ComponentHealth)
	healthyCount := 0
	totalCount := 0

	// Check PostgreSQL
	totalCount++
	pgHealth := h.checkPostgres(ctx)
	checks["postgres"] = pgHealth
	if pgHealth.Status == "healthy" {
		healthyCount++
	}

	// Check ClickHouse
	totalCount++
	chHealth := h.checkClickHouse(ctx)
	checks["clickhouse"] = chHealth
	if chHealth.Status == "healthy" {
		healthyCount++
	}

	// Check Redis
	totalCount++
	redisHealth := h.checkRedis(ctx)
	checks["redis"] = redisHealth
	if redisHealth.Status == "healthy" {
		healthyCount++
	}

	// Determine overall status
	status := HealthStatus{
		Status:    "healthy",
		Service:   h.serviceName,
		Version:   h.version,
		Uptime:    time.Since(h.startTime).String(),
		Timestamp: time.Now().Format(time.RFC3339),
		Checks:    checks,
	}

	statusCode := http.StatusOK

	if healthyCount == 0 {
		status.Status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	} else if healthyCount < totalCount {
		status.Status = "degraded"
		statusCode = http.StatusOK // Still return 200 for degraded
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(status)
}

// checkPostgres verifies PostgreSQL connectivity
func (h *Handler) checkPostgres(ctx context.Context) ComponentHealth {
	start := time.Now()
	err := h.postgres.PingContext(ctx)
	elapsed := time.Since(start)

	if err != nil {
		h.log.Error("Postgres health check failed", "error", err, "elapsed", elapsed)
		return ComponentHealth{
			Status:       "unhealthy",
			ResponseTime: elapsed.String(),
			Error:        err.Error(),
		}
	}

	return ComponentHealth{
		Status:       "healthy",
		ResponseTime: elapsed.String(),
	}
}

// checkClickHouse verifies ClickHouse connectivity
func (h *Handler) checkClickHouse(ctx context.Context) ComponentHealth {
	start := time.Now()
	err := h.clickhouse.Ping(ctx)
	elapsed := time.Since(start)

	if err != nil {
		h.log.Error("ClickHouse health check failed", "error", err, "elapsed", elapsed)
		return ComponentHealth{
			Status:       "unhealthy",
			ResponseTime: elapsed.String(),
			Error:        err.Error(),
		}
	}

	return ComponentHealth{
		Status:       "healthy",
		ResponseTime: elapsed.String(),
	}
}

// checkRedis verifies Redis connectivity
func (h *Handler) checkRedis(ctx context.Context) ComponentHealth {
	start := time.Now()
	err := h.redis.Ping(ctx).Err()
	elapsed := time.Since(start)

	if err != nil {
		h.log.Error("Redis health check failed", "error", err, "elapsed", elapsed)
		return ComponentHealth{
			Status:       "unhealthy",
			ResponseTime: elapsed.String(),
			Error:        err.Error(),
		}
	}

	return ComponentHealth{
		Status:       "healthy",
		ResponseTime: elapsed.String(),
	}
}
