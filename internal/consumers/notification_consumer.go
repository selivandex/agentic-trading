package consumers

import (
	"context"
	"fmt"
	"time"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/events"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"

	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

// NotificationConsumer consumes events and sends Telegram notifications
type NotificationConsumer struct {
	consumer *kafka.Consumer
	log      *logger.Logger
}

// NewNotificationConsumer creates a new notification consumer
func NewNotificationConsumer(
	consumer *kafka.Consumer,
	log *logger.Logger,
) *NotificationConsumer {
	return &NotificationConsumer{
		consumer: consumer,
		log:      log,
	}
}

// Start begins consuming events and sending notifications
func (nc *NotificationConsumer) Start(ctx context.Context) error {
	nc.log.Info("Starting notification consumer...")

	// Ensure consumer is closed on exit
	defer func() {
		nc.log.Info("Closing notification consumer...")
		if err := nc.consumer.Close(); err != nil {
			nc.log.Error("Failed to close notification consumer", "error", err)
		} else {
			nc.log.Info("✓ Notification consumer closed")
		}
	}()

	// Subscribe to all notification-worthy topics
	topics := []string{
		events.TopicOrderPlaced,
		events.TopicOrderFilled,
		events.TopicPositionOpened,
		events.TopicPositionClosed,
		events.TopicStopLossTriggered,
		events.TopicTakeProfitHit,
		events.TopicCircuitBreakerTripped,
		events.TopicDrawdownAlert,
	}

	for _, topic := range topics {
		nc.log.Info("Subscribed to topic", "topic", topic)
	}

	// Consume messages (ReadMessage blocks until message or ctx cancelled)
	for {
		msg, err := nc.consumer.ReadMessage(ctx)
		if err != nil {
			// Check if error is due to context cancellation or reader closure
			if ctx.Err() != nil {
				nc.log.Info("Notification consumer stopping (context cancelled)")
				return nil
			}
			// Reader might be closed during shutdown, log at debug level
			nc.log.Debug("Failed to read message", "error", err)
			continue
		}

		// Process message with timeout to prevent hanging during shutdown
		// Allow up to 5s to complete current message processing
		processCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := nc.handleMessage(processCtx, msg); err != nil {
			nc.log.Error("Failed to handle message",
				"topic", msg.Topic,
				"error", err,
			)
		}
		cancel()

		// Check if we should stop AFTER processing current message
		if ctx.Err() != nil {
			nc.log.Info("Notification consumer stopping after processing current message")
			return nil
		}
	}
}

// handleMessage processes a single event message
func (nc *NotificationConsumer) handleMessage(ctx context.Context, msg kafkago.Message) error {
	nc.log.Debug("Processing notification event",
		"topic", msg.Topic,
		"size", len(msg.Value),
	)

	switch msg.Topic {
	case events.TopicOrderPlaced:
		return nc.handleOrderPlaced(ctx, msg.Value)
	case events.TopicOrderFilled:
		return nc.handleOrderFilled(ctx, msg.Value)
	case events.TopicPositionOpened:
		return nc.handlePositionOpened(ctx, msg.Value)
	case events.TopicPositionClosed:
		return nc.handlePositionClosed(ctx, msg.Value)
	case events.TopicCircuitBreakerTripped:
		return nc.handleCircuitBreakerTripped(ctx, msg.Value)
	default:
		nc.log.Warn("Unknown topic", "topic", msg.Topic)
		return nil
	}
}

func (nc *NotificationConsumer) handleOrderPlaced(ctx context.Context, data []byte) error {
	var event eventspb.OrderPlacedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal order_placed event")
	}

	// TODO: Send Telegram notification when bot is implemented
	nc.log.Info("Order placed notification",
		"user_id", event.Base.UserId,
		"symbol", event.Symbol,
		"side", event.Side,
		"price", event.Price,
	)

	return nil
}

func (nc *NotificationConsumer) handleOrderFilled(ctx context.Context, data []byte) error {
	var event eventspb.OrderFilledEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal order_filled event")
	}

	nc.log.Info("Order filled notification",
		"user_id", event.Base.UserId,
		"symbol", event.Symbol,
		"filled_price", event.FilledPrice,
		"filled_amount", event.FilledAmount,
	)

	return nil
}

func (nc *NotificationConsumer) handlePositionOpened(ctx context.Context, data []byte) error {
	var event eventspb.PositionOpenedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal position_opened event")
	}

	nc.log.Info("Position opened notification",
		"user_id", event.Base.UserId,
		"symbol", event.Symbol,
		"side", event.Side,
		"entry_price", event.EntryPrice,
		"amount", event.Amount,
	)

	return nil
}

func (nc *NotificationConsumer) handlePositionClosed(ctx context.Context, data []byte) error {
	var event eventspb.PositionClosedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal position_closed event")
	}

	pnlSign := "📈"
	if event.Pnl < 0 {
		pnlSign = "📉"
	}

	nc.log.Info(fmt.Sprintf("%s Position closed notification", pnlSign),
		"user_id", event.Base.UserId,
		"symbol", event.Symbol,
		"pnl", event.Pnl,
		"pnl_percent", event.PnlPercent,
		"reason", event.CloseReason,
	)

	return nil
}

func (nc *NotificationConsumer) handleCircuitBreakerTripped(ctx context.Context, data []byte) error {
	var event eventspb.CircuitBreakerTrippedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal circuit_breaker_tripped event")
	}

	nc.log.Warn("🚨 Circuit breaker tripped",
		"user_id", event.Base.UserId,
		"reason", event.Reason,
		"current_loss", event.CurrentLoss,
		"threshold", event.Threshold,
		"drawdown", event.Drawdown,
	)

	return nil
}
