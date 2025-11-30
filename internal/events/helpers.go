package events

import (
	"fmt"
	"strings"
	"time"

	eventspb "prometheus/internal/events/proto"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// All Kafka topic names MUST be defined here
// Architecture: Domain-level topics with event type discrimination in message payload
//
// Event Type Pattern:
// - Publishers use domain-level topics (e.g., TopicMarketEvents)
// - Event type is stored in BaseEvent.Type field (e.g., "market.opportunity_found")
// - Consumers subscribe to domain topic and filter by event type if needed
//
// Benefits:
// - Reduced operational overhead (~10 topics instead of ~40)
// - Easier scaling (fewer partitions to manage)
// - Simpler consumer setup (subscribe to one domain, filter by type)
// - Better resource utilization (no empty low-volume topics)
const (
	// AI events - AI/ML usage tracking, model inference, costs
	TopicAIEvents = "ai.events"

	// Market events - opportunities, regime changes, signals, anomalies, SMC patterns
	TopicMarketEvents = "market.events"

	// Trading events - orders, positions, executions (high volume)
	TopicTradingEvents = "trading.events"

	// Risk events - circuit breakers, drawdowns, margin calls, limit violations
	TopicRiskEvents = "risk.events"

	// Position events - position monitoring, PnL updates, guardian alerts
	TopicPositionEvents = "position.events"

	// Agent events - agent execution, decisions, errors
	TopicAgentEvents = "agent.events"

	// System events - health checks, worker failures, infrastructure issues
	TopicSystemEvents = "system.events"

	// Notification events - user-facing notifications (Telegram, email)
	TopicNotifications = "notifications"

	// WebSocket events - real-time market data (kline, ticker, depth, trades, liquidations)
	TopicWebSocketEvents = "websocket.events"

	// User Data events - real-time order, position, balance, margin updates from exchange WebSocket
	TopicUserDataEvents = "user-data.events"

	// Analytics events - metrics, statistics, performance, reports
	TopicAnalytics = "analytics"
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

// SanitizeUTF8 removes invalid UTF-8 sequences from string
// This prevents protobuf unmarshaling errors: "string field contains invalid UTF-8"
// Use this for any user-generated content or external API responses before publishing to Kafka
func SanitizeUTF8(s string) string {
	// Replace invalid UTF-8 sequences with empty string
	// This is more robust than string([]rune(s)) and safer than using replacement character
	return strings.ToValidUTF8(s, "")
}
