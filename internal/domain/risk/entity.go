package risk

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CircuitBreakerState tracks user's risk state and circuit breaker status
type CircuitBreakerState struct {
	UserID uuid.UUID `db:"user_id"`

	// Current state
	IsTriggered   bool       `db:"is_triggered"`
	TriggeredAt   *time.Time `db:"triggered_at"`
	TriggerReason *string    `db:"trigger_reason"` // NULL when not triggered

	// Daily stats
	DailyPnL          decimal.Decimal `db:"daily_pnl"`
	DailyPnLPercent   decimal.Decimal `db:"daily_pnl_percent"`
	DailyTradeCount   int             `db:"daily_trade_count"`
	DailyWins         int             `db:"daily_wins"`
	DailyLosses       int             `db:"daily_losses"`
	ConsecutiveLosses int             `db:"consecutive_losses"`

	// Thresholds (from user settings)
	MaxDailyDrawdown   decimal.Decimal `db:"max_daily_drawdown"`
	MaxConsecutiveLoss int             `db:"max_consecutive_loss"`

	// Auto-reset
	ResetAt time.Time `db:"reset_at"` // Next day 00:00 UTC

	UpdatedAt time.Time `db:"updated_at"`
}

// RiskEvent represents a risk-related event
type RiskEvent struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	Timestamp time.Time `db:"timestamp"`

	EventType RiskEventType `db:"event_type"`
	Severity  string        `db:"severity"` // warning, critical
	Message   string        `db:"message"`
	Data      string        `db:"data"` // JSON with details

	Acknowledged bool `db:"acknowledged"`
}

// RiskEventType defines types of risk events
type RiskEventType string

const (
	RiskEventDrawdown        RiskEventType = "drawdown_warning"
	RiskEventConsecutiveLoss RiskEventType = "consecutive_loss"
	RiskEventCircuitBreaker  RiskEventType = "circuit_breaker_triggered"
	RiskEventMaxExposure     RiskEventType = "max_exposure_reached"
	RiskEventKillSwitch      RiskEventType = "kill_switch_activated"
	RiskEventAnomalyDetected RiskEventType = "anomaly_detected"
)

// Valid checks if risk event type is valid
func (r RiskEventType) Valid() bool {
	switch r {
	case RiskEventDrawdown, RiskEventConsecutiveLoss, RiskEventCircuitBreaker,
		RiskEventMaxExposure, RiskEventKillSwitch, RiskEventAnomalyDetected:
		return true
	}
	return false
}

// String returns string representation
func (r RiskEventType) String() string {
	return string(r)
}

// RedisClient defines interface for Redis operations needed by risk services
// This allows domain to not depend on concrete Redis implementation
type RedisClient interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
}
