package events

import (
	"fmt"
	"time"

	eventspb "prometheus/internal/events/proto"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// All Kafka topic names MUST be defined here
const (
	// AI/ML events
	TopicAIUsage = "ai.usage" // AI model usage tracking (tokens, costs, latency)

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
	TopicRiskEvents            = "risk_events" // General risk events stream

	// Agent events
	TopicAgentExecuted = "agent.executed"
	TopicAgentError    = "agent.error"
	TopicDecisionMade  = "agent.decision_made"

	// System events
	TopicHealthCheckFailed = "system.health_check_failed"
	TopicWorkerFailed      = "system.worker_failed"

	// Notification events
	TopicNotifications         = "notifications"          // General notification stream
	TopicTelegramNotifications = "telegram_notifications" // Telegram-specific notifications
	TopicAnalytics             = "analytics"              // Analytics events stream

	// Worker-specific events
	TopicPnLUpdated          = "user.pnl_updated"
	TopicJournalEntryCreated = "journal.entry_created"
	TopicDailyReport         = "report.daily"
	TopicStrategyDisabled    = "strategy.disabled"
	TopicStrategyWarning     = "strategy.warning"
	TopicPositionPnLUpdated  = "position.pnl_updated"

	// Position Guardian events (event-driven monitoring)
	TopicStopApproaching    = "position.stop_approaching"
	TopicTargetApproaching  = "position.target_approaching"
	TopicThesisInvalidation = "position.thesis_invalidation"
	TopicTimeDecay          = "position.time_decay"
	TopicProfitMilestone    = "position.profit_milestone"
	TopicCorrelationSpike   = "position.correlation_spike"
	TopicVolatilitySpike    = "position.volatility_spike"
	TopicPositionGuardian   = "position_guardian" // General position guardian stream

	// SMC (Smart Money Concepts) events
	TopicFVGDetected        = "smc.fvg_detected"
	TopicOrderBlockDetected = "smc.order_block_detected"

	// Internal worker events
	TopicAnalysisRequested = "market.analysis_requested"
	TopicScanComplete      = "market.scan_complete"
	TopicWhaleAlert        = "market.whale_alerts"
	TopicLiquidationAlert  = "market.liquidations"

	// WebSocket stream events (real-time market data)
	TopicWebSocketKline       = "websocket.kline"
	TopicWebSocketTicker      = "websocket.ticker"
	TopicWebSocketDepth       = "websocket.depth"
	TopicWebSocketTrade       = "websocket.trade"
	TopicWebSocketFundingRate = "websocket.funding_rate"
	TopicWebSocketMarkPrice   = "websocket.mark_price"
	TopicWebSocketLiquidation = "websocket.liquidation"
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
