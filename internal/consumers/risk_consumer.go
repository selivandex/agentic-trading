package consumers

import (
	"context"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/events"
	eventspb "prometheus/internal/events/proto"
	riskservice "prometheus/internal/services/risk"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// RiskConsumer handles risk events and triggers automated actions
type RiskConsumer struct {
	consumer   *kafka.Consumer
	riskEngine *riskservice.RiskEngine
	log        *logger.Logger
}

// NewRiskConsumer creates a new risk event consumer
func NewRiskConsumer(
	consumer *kafka.Consumer,
	riskEngine *riskservice.RiskEngine,
	log *logger.Logger,
) *RiskConsumer {
	return &RiskConsumer{
		consumer:   consumer,
		riskEngine: riskEngine,
		log:        log,
	}
}

// Start begins consuming risk events
func (rc *RiskConsumer) Start(ctx context.Context) error {
	rc.log.Info("Starting risk consumer...")

	// Ensure consumer is closed on exit
	defer func() {
		rc.log.Info("Closing risk consumer...")
		if err := rc.consumer.Close(); err != nil {
			rc.log.Errorw("Failed to close risk consumer", "error", err)
		} else {
			rc.log.Info("✓ Risk consumer closed")
		}
	}()

	// Subscribe to domain-level risk topic (events filtered by type in handleMessage)
	rc.log.Infow("Subscribed to risk events", "topic", events.TopicRiskEvents)

	// Consume messages (ReadMessage blocks until message or ctx cancelled)
	for {
		msg, err := rc.consumer.ReadMessageWithShutdownCheck(ctx)
		if err != nil {
			// Check if error is due to context cancellation or reader closure
			if ctx.Err() != nil {
				rc.log.Info("Risk consumer stopping (context cancelled)")
				return nil
			}
			// Reader might be closed during shutdown, log at debug level
			rc.log.Debug("Failed to read risk event", "error", err)
			continue
		}

		// Process message with timeout to prevent hanging during shutdown
		// Allow up to 5s to complete current message processing
		processCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := rc.handleMessage(processCtx, msg); err != nil {
			rc.log.Error("Failed to handle risk event",
				"topic", msg.Topic,
				"error", err,
			)
		}
		cancel()

		// Check if we should stop AFTER processing current message
		if ctx.Err() != nil {
			rc.log.Info("Risk consumer stopping after processing current message")
			return nil
		}
	}
}

// handleMessage processes a single risk event
// With domain-level topics, all risk events come through TopicRiskEvents
// and are filtered by event.Base.Type
func (rc *RiskConsumer) handleMessage(ctx context.Context, msg kafkago.Message) error {
	// First, unmarshal to get BaseEvent and determine type
	var baseEvent eventspb.BaseEvent
	if err := proto.Unmarshal(msg.Value, &baseEvent); err != nil {
		return errors.Wrap(err, "unmarshal base event")
	}

	rc.log.Debug("Processing risk event",
		"topic", msg.Topic,
		"event_type", baseEvent.Type,
		"size", len(msg.Value),
	)

	// Route by event type
	switch baseEvent.Type {
	case "risk.circuit_breaker_tripped":
		return rc.handleCircuitBreakerTripped(ctx, msg.Value)
	case "risk.drawdown_alert":
		return rc.handleDrawdownAlert(ctx, msg.Value)
	case "risk.margin_call":
		// TODO: implement margin call handler
		rc.log.Warn("Margin call handler not implemented yet")
		return nil
	case "risk.limit_exceeded":
		// TODO: implement risk limit handler
		rc.log.Warn("Risk limit handler not implemented yet")
		return nil
	default:
		rc.log.Debug("Unhandled risk event type", "type", baseEvent.Type)
		return nil
	}
}

func (rc *RiskConsumer) handleCircuitBreakerTripped(ctx context.Context, data []byte) error {
	var event eventspb.CircuitBreakerTrippedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal circuit_breaker_tripped")
	}

	rc.log.Warn("Circuit breaker tripped - automated action triggered",
		"user_id", event.Base.UserId,
		"reason", event.Reason,
		"drawdown", event.Drawdown,
	)

	// Automated actions:
	// 1. Log to audit trail
	// 2. Update user preferences (pause trading)
	// 3. Send emergency notification
	// 4. Record in risk metrics

	// TODO: Implement automated actions when services are ready

	return nil
}

func (rc *RiskConsumer) handleDrawdownAlert(ctx context.Context, data []byte) error {
	// Similar to circuit breaker but less severe
	// Just log and notify, don't block trading

	rc.log.Warn("Drawdown alert received")
	return nil
}
