package consumers

import (
	"context"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"

	"prometheus/internal/adapters/kafka"
	telegram "prometheus/internal/adapters/telegram"
	"prometheus/internal/domain/user"
	"prometheus/internal/events"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// TelegramNotificationConsumer sends Telegram notifications for trading events
type TelegramNotificationConsumer struct {
	consumer     *kafka.Consumer
	bot          *telegram.Bot
	notifService *telegram.NotificationService
	userRepo     user.Repository
	log          *logger.Logger
}

// NewTelegramNotificationConsumer creates a new Telegram notification consumer
func NewTelegramNotificationConsumer(
	consumer *kafka.Consumer,
	bot *telegram.Bot,
	notifService *telegram.NotificationService,
	userRepo user.Repository,
	log *logger.Logger,
) *TelegramNotificationConsumer {
	return &TelegramNotificationConsumer{
		consumer:     consumer,
		bot:          bot,
		notifService: notifService,
		userRepo:     userRepo,
		log:          log,
	}
}

// Start begins consuming notification events
func (tnc *TelegramNotificationConsumer) Start(ctx context.Context) error {
	tnc.log.Info("Starting Telegram notification consumer...")

	// Ensure consumer is closed on exit
	defer func() {
		tnc.log.Info("Closing Telegram notification consumer...")
		if err := tnc.consumer.Close(); err != nil {
			tnc.log.Error("Failed to close consumer", "error", err)
		} else {
			tnc.log.Info("✓ Telegram notification consumer closed")
		}
	}()

	// Consume messages (blocks until message or ctx cancelled)
	for {
		msg, err := tnc.consumer.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				tnc.log.Info("Telegram notification consumer stopping (context cancelled)")
				return nil
			}
			tnc.log.Debug("Failed to read notification event", "error", err)
			continue
		}

		// Process message with timeout
		processCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := tnc.handleMessage(processCtx, msg); err != nil {
			tnc.log.Error("Failed to handle notification",
				"topic", msg.Topic,
				"error", err,
			)
		}
		cancel()

		// Check if we should stop
		if ctx.Err() != nil {
			tnc.log.Info("Telegram notification consumer stopping after processing current message")
			return nil
		}
	}
}

// handleMessage routes message to appropriate handler based on topic
func (tnc *TelegramNotificationConsumer) handleMessage(ctx context.Context, msg kafkago.Message) error {
	tnc.log.Debug("Processing notification event", "topic", msg.Topic)

	switch msg.Topic {
	case events.TopicPositionOpened:
		return tnc.handlePositionOpened(ctx, msg.Value)

	case events.TopicPositionClosed:
		return tnc.handlePositionClosed(ctx, msg.Value)

	case events.TopicStopLossTriggered:
		return tnc.handleStopLossTriggered(ctx, msg.Value)

	case events.TopicTakeProfitHit:
		return tnc.handleTakeProfitHit(ctx, msg.Value)

	case events.TopicCircuitBreakerTripped:
		return tnc.handleCircuitBreakerTripped(ctx, msg.Value)

	case events.TopicDailyReport:
		return tnc.handleDailyReport(ctx, msg.Value)

	case events.TopicOpportunityFound:
		return tnc.handleOpportunityFound(ctx, msg.Value)

	default:
		tnc.log.Debug("Ignoring unknown topic", "topic", msg.Topic)
		return nil
	}
}

// handlePositionOpened handles position opened events
func (tnc *TelegramNotificationConsumer) handlePositionOpened(ctx context.Context, data []byte) error {
	var event eventspb.PositionOpenedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal position_opened")
	}

	chatID, err := tnc.getChatID(ctx, event.Base.UserId)
	if err != nil {
		return err
	}

	notifData := telegram.PositionOpenedData{
		Symbol:     event.Symbol,
		Side:       event.Side,
		EntryPrice: event.EntryPrice,
		Amount:     event.Amount,
		StopLoss:   event.StopLoss,
		TakeProfit: event.TakeProfit,
		Exchange:   event.Exchange,
		Reasoning:  "", // Not in protobuf schema
		Timestamp:  event.Base.Timestamp.AsTime(),
	}

	return tnc.notifService.NotifyPositionOpened(chatID, notifData)
}

// handlePositionClosed handles position closed events
func (tnc *TelegramNotificationConsumer) handlePositionClosed(ctx context.Context, data []byte) error {
	var event eventspb.PositionClosedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal position_closed")
	}

	chatID, err := tnc.getChatID(ctx, event.Base.UserId)
	if err != nil {
		return err
	}

	notifData := telegram.PositionClosedData{
		Symbol:      event.Symbol,
		Side:        event.Side,
		EntryPrice:  event.EntryPrice,
		ExitPrice:   event.ExitPrice,
		PnL:         event.Pnl,        // Lowercase 'n' in protobuf
		PnLPercent:  event.PnlPercent, // Lowercase 'n' in protobuf
		Duration:    time.Duration(event.DurationSeconds) * time.Second,
		CloseReason: event.CloseReason,
		Timestamp:   event.Base.Timestamp.AsTime(),
	}

	return tnc.notifService.NotifyPositionClosed(chatID, notifData)
}

// handleStopLossTriggered handles stop loss triggered events
func (tnc *TelegramNotificationConsumer) handleStopLossTriggered(ctx context.Context, data []byte) error {
	var event eventspb.PositionClosedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal stop_loss_triggered")
	}

	chatID, err := tnc.getChatID(ctx, event.Base.UserId)
	if err != nil {
		return err
	}

	notifData := telegram.StopLossHitData{
		Symbol:      event.Symbol,
		Side:        event.Side,
		EntryPrice:  event.EntryPrice,
		StopPrice:   event.ExitPrice,   // ExitPrice is the stop price when SL triggered
		Loss:        -event.Pnl,        // Make it positive for display
		LossPercent: -event.PnlPercent, // Make it positive for display
		Timestamp:   event.Base.Timestamp.AsTime(),
	}

	return tnc.notifService.NotifyStopLossHit(chatID, notifData)
}

// handleTakeProfitHit handles take profit hit events
func (tnc *TelegramNotificationConsumer) handleTakeProfitHit(ctx context.Context, data []byte) error {
	var event eventspb.PositionClosedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal take_profit_hit")
	}

	chatID, err := tnc.getChatID(ctx, event.Base.UserId)
	if err != nil {
		return err
	}

	notifData := telegram.TakeProfitHitData{
		Symbol:        event.Symbol,
		Side:          event.Side,
		EntryPrice:    event.EntryPrice,
		TargetPrice:   event.ExitPrice,  // ExitPrice is the target price when TP hit
		Profit:        event.Pnl,        // Lowercase 'n'
		ProfitPercent: event.PnlPercent, // Lowercase 'n'
		Timestamp:     event.Base.Timestamp.AsTime(),
	}

	return tnc.notifService.NotifyTakeProfitHit(chatID, notifData)
}

// handleCircuitBreakerTripped handles circuit breaker events
func (tnc *TelegramNotificationConsumer) handleCircuitBreakerTripped(ctx context.Context, data []byte) error {
	var event eventspb.CircuitBreakerTrippedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal circuit_breaker_tripped")
	}

	chatID, err := tnc.getChatID(ctx, event.Base.UserId)
	if err != nil {
		return err
	}

	notifData := telegram.CircuitBreakerData{
		Reason:          event.Reason,
		DailyLoss:       event.CurrentLoss,
		LossPercent:     event.DailyLossPercent,
		ConsecutiveLoss: int(event.ConsecutiveLosses),
		Timestamp:       event.Base.Timestamp.AsTime(),
		Action:          "All trading paused",
	}

	return tnc.notifService.NotifyCircuitBreaker(chatID, notifData)
}

// handleDailyReport handles daily report events
func (tnc *TelegramNotificationConsumer) handleDailyReport(ctx context.Context, data []byte) error {
	// TODO: Define DailyReportEvent protobuf schema
	// For now, skip
	tnc.log.Debug("Daily report event received (not yet implemented)")
	return nil
}

// handleOpportunityFound handles opportunity found events (optional notification)
func (tnc *TelegramNotificationConsumer) handleOpportunityFound(ctx context.Context, data []byte) error {
	var event eventspb.OpportunityFoundEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal opportunity_found")
	}

	// This is a global event - we could notify all users monitoring this symbol
	// For now, skip to avoid spam (users will get notified when their personal workflow executes)
	tnc.log.Debug("Opportunity found event received",
		"symbol", event.Symbol,
		"confidence", event.Confidence,
	)

	return nil
}

// getChatID gets Telegram chat ID for a user
func (tnc *TelegramNotificationConsumer) getChatID(ctx context.Context, userIDStr string) (int64, error) {
	// Parse user ID
	userID, err := parseUserID(userIDStr)
	if err != nil {
		return 0, errors.Wrap(err, "invalid user_id")
	}

	// Get user from repository
	usr, err := tnc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get user")
	}

	// Check if notifications are enabled
	if !usr.Settings.NotificationsOn {
		tnc.log.Debug("Notifications disabled for user", "user_id", userID)
		return 0, errors.New("notifications disabled")
	}

	return usr.TelegramID, nil
}

// parseUserID parses user ID string to UUID
func parseUserID(userIDStr string) (uuid.UUID, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, errors.Wrap(err, "invalid user_id format")
	}
	return userID, nil
}
