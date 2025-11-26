package events

import (
	"context"

	"prometheus/internal/adapters/kafka"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"

	"google.golang.org/protobuf/proto"
)

// Publisher publishes events to Kafka
type Publisher struct {
	producer *kafka.Producer
	log      *logger.Logger
}

// NewPublisher creates a new event publisher
func NewPublisher(producer *kafka.Producer, log *logger.Logger) *Publisher {
	return &Publisher{
		producer: producer,
		log:      log,
	}
}

// PublishOpportunity publishes an opportunity found event
func (p *Publisher) PublishOpportunity(ctx context.Context, event *eventspb.OpportunityFoundEvent) error {
	return p.publish(ctx, TopicOpportunityFound, event)
}

// PublishRegimeChange publishes a regime changed event
func (p *Publisher) PublishRegimeChange(ctx context.Context, event *eventspb.RegimeChangedEvent) error {
	return p.publish(ctx, TopicRegimeChanged, event)
}

// PublishOrderPlaced publishes an order placed event
func (p *Publisher) PublishOrderPlaced(ctx context.Context, event *eventspb.OrderPlacedEvent) error {
	return p.publish(ctx, TopicOrderPlaced, event)
}

// PublishOrderFilled publishes an order filled event
func (p *Publisher) PublishOrderFilled(ctx context.Context, event *eventspb.OrderFilledEvent) error {
	return p.publish(ctx, TopicOrderFilled, event)
}

// PublishPositionOpened publishes a position opened event
func (p *Publisher) PublishPositionOpened(ctx context.Context, event *eventspb.PositionOpenedEvent) error {
	return p.publish(ctx, TopicPositionOpened, event)
}

// PublishPositionClosed publishes a position closed event
func (p *Publisher) PublishPositionClosed(ctx context.Context, event *eventspb.PositionClosedEvent) error {
	return p.publish(ctx, TopicPositionClosed, event)
}

// PublishCircuitBreakerTripped publishes a circuit breaker event
func (p *Publisher) PublishCircuitBreakerTripped(ctx context.Context, event *eventspb.CircuitBreakerTrippedEvent) error {
	return p.publish(ctx, TopicCircuitBreakerTripped, event)
}

// PublishAgentExecuted publishes an agent execution event
func (p *Publisher) PublishAgentExecuted(ctx context.Context, event *eventspb.AgentExecutedEvent) error {
	return p.publish(ctx, TopicAgentExecuted, event)
}

// PublishDecisionMade publishes an agent decision event
func (p *Publisher) PublishDecisionMade(ctx context.Context, event *eventspb.DecisionMadeEvent) error {
	return p.publish(ctx, TopicDecisionMade, event)
}

// PublishWorkerFailed publishes a worker failure event
func (p *Publisher) PublishWorkerFailed(ctx context.Context, event *eventspb.WorkerFailedEvent) error {
	return p.publish(ctx, TopicWorkerFailed, event)
}

// publish is the generic publish method using protobuf serialization
func (p *Publisher) publish(ctx context.Context, topic string, event proto.Message) error {
	// Serialize to protobuf binary format
	data, err := proto.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "marshal protobuf")
	}

	// Publish to Kafka using binary method (no JSON serialization)
	if err := p.producer.PublishBinary(ctx, topic, nil, data); err != nil {
		p.log.Error("Failed to publish event",
			"topic", topic,
			"error", err,
		)
		return errors.Wrap(err, "send to kafka")
	}

	p.log.Debug("Event published",
		"topic", topic,
		"size_bytes", len(data),
	)

	return nil
}
