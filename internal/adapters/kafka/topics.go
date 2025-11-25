package kafka

// Topic definitions for Kafka event streaming
const (
	// Trading events
	TopicTradeOpened   = "trades.opened"
	TopicTradeClosed   = "trades.closed"
	TopicOrderPlaced   = "orders.placed"
	TopicOrderFilled   = "orders.filled"
	TopicOrderCanceled = "orders.canceled"

	// Risk events
	TopicRiskAlert      = "risk.alerts"
	TopicCircuitBreaker = "risk.circuit_breaker"
	TopicKillSwitch     = "risk.kill_switch"

	// Market data events
	TopicPriceUpdate = "market.prices"
	TopicLiquidation = "market.liquidations"
	TopicWhaleAlert  = "market.whale_alerts"

	// Agent events
	TopicAgentDecision = "agents.decisions"
	TopicAgentSignal   = "agents.signals"

	// Notifications
	TopicNotifications = "notifications.telegram"
)
