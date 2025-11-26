package consumers

import (
	"context"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/events"
	eventspb "prometheus/internal/events/proto"
	"prometheus/internal/metrics"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"

	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

// AnalyticsConsumer processes events for real-time analytics and metrics
type AnalyticsConsumer struct {
	consumer *kafka.Consumer
	log      *logger.Logger
}

// NewAnalyticsConsumer creates a new analytics consumer
func NewAnalyticsConsumer(
	consumer *kafka.Consumer,
	log *logger.Logger,
) *AnalyticsConsumer {
	return &AnalyticsConsumer{
		consumer: consumer,
		log:      log,
	}
}

// Start begins consuming events for analytics
func (ac *AnalyticsConsumer) Start(ctx context.Context) error {
	ac.log.Info("Starting analytics consumer...")

	// Subscribe to analytics-worthy topics
	topics := []string{
		events.TopicOpportunityFound,
		events.TopicRegimeChanged,
		events.TopicAgentExecuted,
		events.TopicDecisionMade,
		events.TopicWorkerFailed,
	}

	for _, topic := range topics {
		ac.log.Info("Subscribed to analytics topic", "topic", topic)
	}

	// Consume messages (ReadMessage blocks until message or ctx cancelled)
	for {
		msg, err := ac.consumer.ReadMessage(ctx)
		if err != nil {
			// Check if error is due to context cancellation
			if ctx.Err() != nil {
				ac.log.Info("Analytics consumer stopping (context cancelled)")
				return nil
			}
			ac.log.Error("Failed to read analytics event", "error", err)
			continue
		}

		// Process message
		if err := ac.handleMessage(ctx, msg); err != nil {
			ac.log.Error("Failed to handle analytics event",
				"topic", msg.Topic,
				"error", err,
			)
		}
	}
}

// handleMessage processes a single analytics event
func (ac *AnalyticsConsumer) handleMessage(ctx context.Context, msg kafkago.Message) error {
	ac.log.Debug("Processing analytics event",
		"topic", msg.Topic,
		"size", len(msg.Value),
	)

	switch msg.Topic {
	case events.TopicOpportunityFound:
		return ac.handleOpportunityFound(ctx, msg.Value)
	case events.TopicRegimeChanged:
		return ac.handleRegimeChanged(ctx, msg.Value)
	case events.TopicAgentExecuted:
		return ac.handleAgentExecuted(ctx, msg.Value)
	case events.TopicDecisionMade:
		return ac.handleDecisionMade(ctx, msg.Value)
	case events.TopicWorkerFailed:
		return ac.handleWorkerFailed(ctx, msg.Value)
	default:
		ac.log.Warn("Unknown analytics topic", "topic", msg.Topic)
		return nil
	}
}

func (ac *AnalyticsConsumer) handleOpportunityFound(ctx context.Context, data []byte) error {
	var event eventspb.OpportunityFoundEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal opportunity_found")
	}

	ac.log.Info("Opportunity detected",
		"symbol", event.Symbol,
		"direction", event.Direction,
		"confidence", event.Confidence,
		"strategy", event.Strategy,
	)

	// TODO: Track opportunity metrics
	// - Count by strategy type
	// - Average confidence
	// - Conversion rate (opportunity → trade)

	return nil
}

func (ac *AnalyticsConsumer) handleRegimeChanged(ctx context.Context, data []byte) error {
	var event eventspb.RegimeChangedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal regime_changed")
	}

	ac.log.Info("Market regime changed",
		"symbol", event.Symbol,
		"old_regime", event.OldRegime,
		"new_regime", event.NewRegime,
		"trend", event.Trend,
	)

	// TODO: Update regime metrics in Prometheus
	// - Track regime durations
	// - Regime transition patterns
	// - Performance by regime

	return nil
}

func (ac *AnalyticsConsumer) handleAgentExecuted(ctx context.Context, data []byte) error {
	var event eventspb.AgentExecutedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal agent_executed")
	}

	// Update Prometheus metrics
	if event.Success {
		metrics.AgentCalls.WithLabelValues(event.AgentType, event.Model, "success").Inc()
	} else {
		metrics.AgentCalls.WithLabelValues(event.AgentType, event.Model, "error").Inc()
	}

	if event.CostUsd > 0 {
		metrics.AgentCost.WithLabelValues(event.AgentType, event.Base.UserId, event.Model).Add(event.CostUsd)
	}

	if event.TokensUsed > 0 {
		metrics.AgentTokens.WithLabelValues(event.AgentType, event.Model, "total").Add(float64(event.TokensUsed))
	}

	ac.log.Debug("Agent execution tracked",
		"agent", event.AgentType,
		"duration_ms", event.DurationMs,
		"cost", event.CostUsd,
		"tokens", event.TokensUsed,
	)

	return nil
}

func (ac *AnalyticsConsumer) handleDecisionMade(ctx context.Context, data []byte) error {
	var event eventspb.DecisionMadeEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal decision_made")
	}

	ac.log.Info("Agent decision",
		"agent", event.AgentType,
		"action", event.Action,
		"symbol", event.Symbol,
		"confidence", event.Confidence,
	)

	// TODO: Track decision quality over time
	// - Decision → outcome correlation
	// - Confidence vs actual performance
	// - Agent accuracy by market regime

	return nil
}

func (ac *AnalyticsConsumer) handleWorkerFailed(ctx context.Context, data []byte) error {
	var event eventspb.WorkerFailedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal worker_failed")
	}

	ac.log.Error("Worker failure detected",
		"worker", event.WorkerName,
		"error", event.Error,
		"fail_count", event.FailCount,
	)

	// TODO: Automated recovery actions
	// - If fail_count > 3: disable worker
	// - If critical worker: send alert
	// - Update health check status

	return nil
}
