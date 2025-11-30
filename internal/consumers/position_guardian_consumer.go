package consumers

import (
	"context"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/events"
	eventspb "prometheus/internal/events/proto"
	"prometheus/internal/services/position"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// PositionGuardianConsumer handles position monitoring events with event-driven architecture
// Routes events by urgency: CRITICAL → algorithmic, HIGH/MEDIUM → LLM agent
type PositionGuardianConsumer struct {
	consumer        *kafka.Consumer
	criticalHandler *position.CriticalEventHandler
	agentHandler    *position.AgentEventHandler
	log             *logger.Logger
}

// NewPositionGuardianConsumer creates a new position guardian consumer
func NewPositionGuardianConsumer(
	consumer *kafka.Consumer,
	criticalHandler *position.CriticalEventHandler,
	agentHandler *position.AgentEventHandler,
	log *logger.Logger,
) *PositionGuardianConsumer {
	return &PositionGuardianConsumer{
		consumer:        consumer,
		criticalHandler: criticalHandler,
		agentHandler:    agentHandler,
		log:             log,
	}
}

// Start begins consuming position guardian events
func (pgc *PositionGuardianConsumer) Start(ctx context.Context) error {
	pgc.log.Info("Starting position guardian consumer (event-driven monitoring)...")

	// Ensure consumer is closed on exit
	defer func() {
		pgc.log.Info("Closing position guardian consumer...")
		if err := pgc.consumer.Close(); err != nil {
			pgc.log.Error("Failed to close position guardian consumer", "error", err)
		} else {
			pgc.log.Info("✓ Position guardian consumer closed")
		}
	}()

	// Subscribe to domain-level topics
	// Position guardian monitors events from multiple domains:
	// - TopicPositionEvents: stop_approaching, target_approaching, etc
	// - TopicTradingEvents: stop_loss_triggered, take_profit_hit
	// - TopicRiskEvents: circuit_breaker_tripped
	pgc.log.Infow("Subscribed to position guardian domains",
		"position_events", events.TopicPositionEvents,
		"trading_events", events.TopicTradingEvents,
		"risk_events", events.TopicRiskEvents,
	)

	// Consume messages (ReadMessage blocks until message or ctx cancelled)
	for {
		msg, err := pgc.consumer.ReadMessageWithShutdownCheck(ctx)
		if err != nil {
			// Check if error is due to context cancellation or reader closure
			if ctx.Err() != nil {
				pgc.log.Info("Position guardian consumer stopping (context cancelled)")
				return nil
			}
			// Reader might be closed during shutdown, log at debug level
			pgc.log.Debugw("Failed to read position guardian event", "error", err)
			continue
		}

		// Process message with timeout to prevent hanging during shutdown
		// Allow up to 45s to complete current message processing (30s agent timeout + 15s buffer)
		processCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		if err := pgc.handleMessage(processCtx, msg); err != nil {
			pgc.log.Errorw("Failed to handle position guardian event",
				"topic", msg.Topic,
				"error", err,
			)
		}
		cancel()

		// Check if we should stop AFTER processing current message
		if ctx.Err() != nil {
			pgc.log.Info("Position guardian consumer stopping after processing current message")
			return nil
		}
	}
}

// handleMessage processes a single position guardian event
// With domain-level topics, events come from multiple domains and are filtered by event.Base.Type
func (pgc *PositionGuardianConsumer) handleMessage(ctx context.Context, msg kafkago.Message) error {
	// First, unmarshal to get BaseEvent and determine type
	var baseEvent eventspb.BaseEvent
	if err := proto.Unmarshal(msg.Value, &baseEvent); err != nil {
		return errors.Wrap(err, "unmarshal base event")
	}

	pgc.log.Debugw("Processing position guardian event",
		"topic", msg.Topic,
		"event_type", baseEvent.Type,
		"size", len(msg.Value),
	)

	// Route by event type
	switch baseEvent.Type {
	// CRITICAL events - immediate algorithmic action
	case "trading.stop_loss_triggered":
		return pgc.handleStopLossTriggered(ctx, msg.Value)
	case "trading.take_profit_hit":
		return pgc.handleTakeProfitHit(ctx, msg.Value)
	case "risk.circuit_breaker_tripped":
		return pgc.handleCircuitBreakerTripped(ctx, msg.Value)

	// HIGH/MEDIUM events - LLM-based decision
	case "position.stop_approaching":
		return pgc.handleStopApproaching(ctx, msg.Value)
	case "position.target_approaching":
		return pgc.handleTargetApproaching(ctx, msg.Value)
	case "position.profit_milestone":
		return pgc.handleProfitMilestone(ctx, msg.Value)
	case "position.thesis_invalidation":
		return pgc.handleThesisInvalidation(ctx, msg.Value)
	case "position.time_decay":
		return pgc.handleTimeDecay(ctx, msg.Value)

	// LOW priority events - log for now (can implement batch processing later)
	case "position.correlation_spike":
		return pgc.handleCorrelationSpike(ctx, msg.Value)
	case "position.volatility_spike":
		return pgc.handleVolatilitySpike(ctx, msg.Value)

	default:
		pgc.log.Debug("Unhandled position guardian event type", "type", baseEvent.Type)
		return nil
	}
}

// CRITICAL event handlers - delegate to CriticalEventHandler

func (pgc *PositionGuardianConsumer) handleStopLossTriggered(ctx context.Context, data []byte) error {
	var event eventspb.PositionClosedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal stop_loss_triggered")
	}

	pgc.log.Warnw("Stop loss triggered - delegating to critical handler",
		"position_id", event.PositionId,
		"symbol", event.Symbol,
	)

	return pgc.criticalHandler.HandleStopLossHit(ctx, &event)
}

func (pgc *PositionGuardianConsumer) handleTakeProfitHit(ctx context.Context, data []byte) error {
	var event eventspb.PositionClosedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal take_profit_hit")
	}

	pgc.log.Infow("Take profit hit - delegating to critical handler",
		"position_id", event.PositionId,
		"symbol", event.Symbol,
	)

	return pgc.criticalHandler.HandleTakeProfitHit(ctx, &event)
}

func (pgc *PositionGuardianConsumer) handleCircuitBreakerTripped(ctx context.Context, data []byte) error {
	var event eventspb.CircuitBreakerTrippedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal circuit_breaker_tripped")
	}

	pgc.log.Errorw("Circuit breaker tripped - delegating to critical handler",
		"user_id", event.Base.UserId,
		"reason", event.Reason,
	)

	return pgc.criticalHandler.HandleCircuitBreakerTripped(ctx, &event)
}

// HIGH/MEDIUM event handlers - delegate to AgentEventHandler

func (pgc *PositionGuardianConsumer) handleStopApproaching(ctx context.Context, data []byte) error {
	var event eventspb.StopApproachingEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal stop_approaching")
	}

	pgc.log.Infow("Stop approaching - delegating to agent handler",
		"position_id", event.PositionId,
		"symbol", event.Symbol,
		"distance_pct", event.DistancePercent,
	)

	return pgc.agentHandler.HandleStopApproaching(ctx, &event)
}

func (pgc *PositionGuardianConsumer) handleTargetApproaching(ctx context.Context, data []byte) error {
	var event eventspb.TargetApproachingEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal target_approaching")
	}

	pgc.log.Infow("Target approaching - delegating to agent handler",
		"position_id", event.PositionId,
		"symbol", event.Symbol,
		"distance_pct", event.DistancePercent,
	)

	return pgc.agentHandler.HandleTargetApproaching(ctx, &event)
}

func (pgc *PositionGuardianConsumer) handleProfitMilestone(ctx context.Context, data []byte) error {
	var event eventspb.ProfitMilestoneEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal profit_milestone")
	}

	pgc.log.Infow("Profit milestone reached - delegating to agent handler",
		"position_id", event.PositionId,
		"symbol", event.Symbol,
		"milestone", event.Milestone,
	)

	return pgc.agentHandler.HandleProfitMilestone(ctx, &event)
}

func (pgc *PositionGuardianConsumer) handleThesisInvalidation(ctx context.Context, data []byte) error {
	var event eventspb.ThesisInvalidationEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal thesis_invalidation")
	}

	pgc.log.Warnw("Thesis invalidation detected - delegating to agent handler",
		"position_id", event.PositionId,
		"symbol", event.Symbol,
		"reason", event.InvalidationReason,
	)

	return pgc.agentHandler.HandleThesisInvalidation(ctx, &event)
}

func (pgc *PositionGuardianConsumer) handleTimeDecay(ctx context.Context, data []byte) error {
	var event eventspb.TimeDecayEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal time_decay")
	}

	pgc.log.Infow("Time decay detected - delegating to agent handler",
		"position_id", event.PositionId,
		"symbol", event.Symbol,
		"duration_hours", event.DurationHours,
	)

	return pgc.agentHandler.HandleTimeDecay(ctx, &event)
}

// LOW priority event handlers - log and track for now

func (pgc *PositionGuardianConsumer) handleCorrelationSpike(ctx context.Context, data []byte) error {
	var event eventspb.CorrelationSpikeEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal correlation_spike")
	}

	pgc.log.Warnw("Correlation spike detected",
		"user_id", event.Base.UserId,
		"symbols", event.Symbols,
		"correlation", event.Correlation,
		"exposure_pct", event.TotalExposurePercent,
	)

	// TODO: Implement batch processing or portfolio-level analysis
	// For now, just log the event
	return nil
}

func (pgc *PositionGuardianConsumer) handleVolatilitySpike(ctx context.Context, data []byte) error {
	var event eventspb.VolatilitySpikeEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal volatility_spike")
	}

	pgc.log.Warnw("Volatility spike detected",
		"symbol", event.Symbol,
		"current_volatility", event.CurrentVolatility,
		"spike_ratio", event.SpikeRatio,
		"affected_positions", len(event.AffectedPositions),
	)

	// TODO: Implement volatility-based position adjustments
	// For now, just log the event
	return nil
}
