package events

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"prometheus/internal/adapters/kafka"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// StrategyPublisher publishes strategy-related events to Kafka
type StrategyPublisher struct {
	producer *kafka.Producer
	log      *logger.Logger
}

// NewStrategyPublisher creates a new strategy event publisher
func NewStrategyPublisher(producer *kafka.Producer) *StrategyPublisher {
	return &StrategyPublisher{
		producer: producer,
		log:      logger.Get().With("component", "strategy_publisher"),
	}
}

// PublishStrategyCreated publishes a PortfolioInitializationJobEvent to trigger portfolio creation workflow
// This event is consumed by PortfolioInitializationConsumer which runs PortfolioArchitect workflow
func (sp *StrategyPublisher) PublishStrategyCreated(
	ctx context.Context,
	userID uuid.UUID,
	strategyID uuid.UUID,
	telegramID int64,
	capital float64,
	exchangeAccountID uuid.UUID,
	riskProfile string,
	marketType string,
) error {
	// Create event with all required fields
	event := &eventspb.PortfolioInitializationJobEvent{
		Base:              NewBaseEvent("strategy.created", "telegram_handler", userID.String()),
		UserId:            userID.String(),
		StrategyId:        strategyID.String(),
		TelegramId:        telegramID,
		Capital:           capital,
		ExchangeAccountId: exchangeAccountID.String(),
		RiskProfile:       riskProfile,
		MarketType:        marketType,
	}

	// Marshal protobuf
	data, err := proto.Marshal(event)
	if err != nil {
		sp.log.Errorw("Failed to marshal PortfolioInitializationJobEvent",
			"user_id", userID,
			"strategy_id", strategyID,
			"error", err,
		)
		return errors.Wrap(err, "marshal protobuf")
	}

	// Publish to system_events topic (same as other workflow triggers)
	if err := sp.producer.PublishBinary(ctx, TopicSystemEvents, []byte(userID.String()), data); err != nil {
		sp.log.Errorw("Failed to publish strategy.created event",
			"user_id", userID,
			"strategy_id", strategyID,
			"error", err,
		)
		return errors.Wrap(err, "publish to kafka")
	}

	sp.log.Infow("Published strategy.created event",
		"user_id", userID,
		"strategy_id", strategyID,
		"capital", capital,
		"risk_profile", riskProfile,
		"market_type", marketType,
	)

	return nil
}
