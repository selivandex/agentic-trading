package events

import (
	"fmt"
	"time"

	eventspb "prometheus/internal/events/proto"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// Event topic constants
const (
	// Market events
	TopicOpportunityFound = "market.opportunity_found"
	TopicRegimeChanged    = "market.regime_changed"
	TopicSignalGenerated  = "market.signal_generated"
	TopicAnomalyDetected  = "market.anomaly_detected"

	// Trading events
	TopicOrderPlaced       = "trading.order_placed"
	TopicOrderFilled       = "trading.order_filled"
	TopicOrderCancelled    = "trading.order_cancelled"
	TopicPositionOpened    = "trading.position_opened"
	TopicPositionClosed    = "trading.position_closed"
	TopicStopLossTriggered = "trading.stop_loss_triggered"
	TopicTakeProfitHit     = "trading.take_profit_hit"

	// Risk events
	TopicCircuitBreakerTripped = "risk.circuit_breaker_tripped"
	TopicDrawdownAlert         = "risk.drawdown_alert"
	TopicMarginCall            = "risk.margin_call"
	TopicRiskLimitExceeded     = "risk.limit_exceeded"

	// Agent events
	TopicAgentExecuted = "agent.executed"
	TopicAgentError    = "agent.error"
	TopicDecisionMade  = "agent.decision_made"

	// System events
	TopicHealthCheckFailed = "system.health_check_failed"
	TopicWorkerFailed      = "system.worker_failed"
)

// NewBaseEvent creates a new base event with defaults
func NewBaseEvent(eventType, source, userID string) *eventspb.BaseEvent {
	return &eventspb.BaseEvent{
		Id:        generateEventID(),
		Type:      eventType,
		Timestamp: timestamppb.Now(),
		Source:    source,
		UserId:    userID,
		Version:   "1.0",
	}
}

// generateEventID generates a unique event ID
func generateEventID() string {
	// Format: timestamp_nanoseconds
	now := time.Now()
	return fmt.Sprintf("%d_%d", now.Unix(), now.Nanosecond())
}
