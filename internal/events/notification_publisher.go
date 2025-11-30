package events

import (
	"context"

	"prometheus/internal/adapters/kafka"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"

	"google.golang.org/protobuf/proto"
)

// NotificationPublisher provides convenience methods for publishing Telegram notifications
// All notifications go to telegram_notifications topic and are routed by event type
type NotificationPublisher struct {
	kafka *kafka.Producer
}

// NewNotificationPublisher creates a new notification event publisher
func NewNotificationPublisher(kafka *kafka.Producer) *NotificationPublisher {
	return &NotificationPublisher{kafka: kafka}
}

// PublishExchangeDeactivated publishes an exchange deactivated notification
func (np *NotificationPublisher) PublishExchangeDeactivated(
	ctx context.Context,
	accountID, userID, exchange, label, reason, errorMsg string,
	isTestnet bool,
) error {
	event := &eventspb.ExchangeDeactivatedEvent{
		Base:         NewBaseEvent("exchange.deactivated", "exchange_service", userID),
		AccountId:    accountID,
		Exchange:     exchange,
		Label:        label,
		Reason:       reason,
		ErrorMessage: errorMsg,
		IsTestnet:    isTestnet,
	}

	return np.publishNotification(ctx, userID, event)
}

// publishNotification serializes and publishes a notification event to telegram_notifications topic
func (np *NotificationPublisher) publishNotification(ctx context.Context, key string, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "marshal protobuf")
	}

	if err := np.kafka.PublishBinary(ctx, TopicTelegramNotifications, []byte(key), data); err != nil {
		return errors.Wrap(err, "publish to kafka")
	}

	return nil
}
