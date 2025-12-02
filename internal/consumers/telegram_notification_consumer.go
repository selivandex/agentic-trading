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
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// TelegramNotificationConsumer sends Telegram notifications for trading events
type TelegramNotificationConsumer struct {
	consumer     *kafka.Consumer
	notifService *telegram.NotificationService // Uses new framework service
	userService  *user.Service                 // Domain service (no side effects needed)
	log          *logger.Logger
}

// NewTelegramNotificationConsumer creates a new Telegram notification consumer
func NewTelegramNotificationConsumer(
	consumer *kafka.Consumer,
	notifService *telegram.NotificationService,
	userService *user.Service,
	log *logger.Logger,
) *TelegramNotificationConsumer {
	return &TelegramNotificationConsumer{
		consumer:     consumer,
		notifService: notifService,
		userService:  userService,
		log:          log.With("component", "telegram_notification_consumer"),
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
		msg, err := tnc.consumer.ReadMessageWithShutdownCheck(ctx)
		if err != nil {
			if ctx.Err() != nil {
				tnc.log.Info("Telegram notification consumer stopping (context cancelled)")
				return nil
			}
			tnc.log.Debugw("Failed to read notification event", "error", err)
			continue
		}

		// Process message with timeout
		processCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := tnc.handleMessage(processCtx, msg); err != nil {
			tnc.log.Errorw("Failed to handle notification",
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

// handleMessage routes message to appropriate handler based on event type
func (tnc *TelegramNotificationConsumer) handleMessage(ctx context.Context, msg kafkago.Message) error {
	tnc.log.Debugw("Processing notification event", "topic", msg.Topic)

	// For telegram_notifications topic, we need to peek at BaseEvent.Type to route
	// First, try to unmarshal as BaseEvent to get the type
	var baseEvent eventspb.BaseEvent
	if err := proto.Unmarshal(msg.Value, &baseEvent); err != nil {
		tnc.log.Errorw("Failed to unmarshal base event", "error", err)
		return errors.Wrap(err, "unmarshal base event")
	}

	eventType := baseEvent.Type
	tnc.log.Debugw("Routing notification by event type", "event_type", eventType)

	// Route based on event type (not topic)
	switch eventType {
	case "trading.position_opened":
		return tnc.handlePositionOpened(ctx, msg.Value)

	case "trading.position_closed":
		return tnc.handlePositionClosed(ctx, msg.Value)

	case "trading.stop_loss_triggered":
		return tnc.handleStopLossTriggered(ctx, msg.Value)

	case "trading.take_profit_hit":
		return tnc.handleTakeProfitHit(ctx, msg.Value)

	case "risk.circuit_breaker_tripped":
		return tnc.handleCircuitBreakerTripped(ctx, msg.Value)

	case "report.daily":
		return tnc.handleDailyReport(ctx, msg.Value)

	case "market.opportunity_found":
		return tnc.handleOpportunityFound(ctx, msg.Value)

	case "exchange.deactivated":
		return tnc.handleExchangeDeactivated(ctx, msg.Value)

	case "investment.accepted":
		return tnc.handleInvestmentAccepted(ctx, msg.Value)

	case "portfolio.created":
		return tnc.handlePortfolioCreated(ctx, msg.Value)

	default:
		tnc.log.Debugw("Ignoring unknown event type", "event_type", eventType)
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
	tnc.log.Debugw("Opportunity found event received",
		"symbol", event.Symbol,
		"confidence", event.Confidence,
	)

	return nil
}

// getChatID gets Telegram chat ID for a user using UserService (Clean Architecture)
func (tnc *TelegramNotificationConsumer) getChatID(ctx context.Context, userIDStr string) (int64, error) {
	// Parse user ID
	userID, err := parseUserID(userIDStr)
	if err != nil {
		return 0, errors.Wrap(err, "invalid user_id")
	}

	// Get user from service
	usr, err := tnc.userService.GetByID(ctx, userID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get user")
	}

	// Check if notifications are enabled
	if !usr.Settings.NotificationsOn {
		tnc.log.Debugw("Notifications disabled for user", "user_id", userID)
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

// handleExchangeDeactivated handles exchange deactivated events
func (tnc *TelegramNotificationConsumer) handleExchangeDeactivated(ctx context.Context, data []byte) error {
	var event eventspb.ExchangeDeactivatedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal exchange_deactivated")
	}

	chatID, err := tnc.getChatID(ctx, event.Base.UserId)
	if err != nil {
		return err
	}

	notifData := telegram.ExchangeDeactivatedData{
		Exchange:     event.Exchange,
		Label:        event.Label,
		Reason:       event.Reason,
		ErrorMessage: event.ErrorMessage,
		IsTestnet:    event.IsTestnet,
	}

	return tnc.notifService.NotifyExchangeDeactivated(chatID, notifData)
}

// handleInvestmentAccepted handles investment accepted events
func (tnc *TelegramNotificationConsumer) handleInvestmentAccepted(ctx context.Context, data []byte) error {
	var event eventspb.InvestmentAcceptedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal investment_accepted")
	}

	// Use telegram_id directly from event (no need to query user)
	notifData := telegram.InvestmentAcceptedData{
		Capital:     event.Capital,
		RiskProfile: event.RiskProfile,
		Exchange:    event.Exchange,
	}

	return tnc.notifService.NotifyInvestmentAccepted(event.TelegramId, notifData)
}

// handlePortfolioCreated handles portfolio created events
func (tnc *TelegramNotificationConsumer) handlePortfolioCreated(ctx context.Context, data []byte) error {
	var event eventspb.PortfolioCreatedEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal portfolio_created")
	}

	// Use telegram_id directly from event (no need to query user)
	notifData := telegram.PortfolioCreatedData{
		StrategyName:   event.StrategyName,
		Invested:       event.Invested,
		PositionsCount: int(event.PositionsCount),
	}

	return tnc.notifService.NotifyPortfolioCreated(event.TelegramId, notifData)
}
