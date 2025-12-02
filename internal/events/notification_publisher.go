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

// PublishInvestmentAccepted publishes an investment accepted notification
func (np *NotificationPublisher) PublishInvestmentAccepted(
	ctx context.Context,
	userID string,
	telegramID int64,
	capital float64,
	riskProfile, exchange string,
) error {
	event := &eventspb.InvestmentAcceptedEvent{
		Base:        NewBaseEvent("investment.accepted", "system_events_consumer", userID),
		TelegramId:  telegramID,
		Capital:     capital,
		RiskProfile: riskProfile,
		Exchange:    exchange,
	}

	return np.publishNotification(ctx, userID, event)
}

// PublishPortfolioCreated publishes a portfolio created notification
func (np *NotificationPublisher) PublishPortfolioCreated(
	ctx context.Context,
	userID, strategyID, strategyName string,
	telegramID int64,
	invested float64,
	positionsCount int32,
) error {
	event := &eventspb.PortfolioCreatedEvent{
		Base:           NewBaseEvent("portfolio.created", "system_events_consumer", userID),
		TelegramId:     telegramID,
		StrategyId:     strategyID,
		StrategyName:   strategyName,
		Invested:       invested,
		PositionsCount: positionsCount,
	}

	return np.publishNotification(ctx, userID, event)
}

// PublishExchangeDeactivated publishes an exchange deactivated notification
func (np *NotificationPublisher) PublishExchangeDeactivated(
	ctx context.Context,
	accountID, userID, exchange, label, reason, errorMsg string,
	isTestnet bool,
) error {
	// Sanitize ALL string fields to ensure valid UTF-8
	// This prevents protobuf unmarshaling errors: "string field contains invalid UTF-8"
	// External API responses (especially error messages) may contain invalid UTF-8
	label = SanitizeUTF8(label)
	reason = SanitizeUTF8(reason)
	errorMsg = SanitizeUTF8(errorMsg)

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

	if err := np.kafka.PublishBinary(ctx, TopicNotifications, []byte(key), data); err != nil {
		return errors.Wrap(err, "publish to kafka")
	}

	return nil
}
