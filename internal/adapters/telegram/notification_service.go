package telegram

import (
	"prometheus/pkg/logger"
	"prometheus/pkg/telegram"
)

// NotificationService handles sending structured Telegram notifications
// Uses pkg/telegram framework (Bot interface + TemplateRenderer)
type NotificationService struct {
	bot       telegram.Bot
	templates telegram.TemplateRenderer
	log       *logger.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(
	bot telegram.Bot,
	templates telegram.TemplateRenderer,
	log *logger.Logger,
) *NotificationService {
	return &NotificationService{
		bot:       bot,
		templates: templates,
		log:       log.With("component", "telegram_notifications"),
	}
}

// NotifyPositionOpened sends position opened notification
func (ns *NotificationService) NotifyPositionOpened(chatID int64, data PositionOpenedData) error {
	text, err := ns.templates.Render("notifications/position_opened", data)
	if err != nil {
		ns.log.Errorw("Failed to render position_opened template", "error", err)
		return err
	}

	return ns.bot.SendMessage(chatID, text)
}

// NotifyPositionClosed sends position closed notification
func (ns *NotificationService) NotifyPositionClosed(chatID int64, data PositionClosedData) error {
	text, err := ns.templates.Render("notifications/position_closed", data)
	if err != nil {
		ns.log.Errorw("Failed to render position_closed template", "error", err)
		return err
	}

	return ns.bot.SendMessage(chatID, text)
}

// NotifyStopLossHit sends stop loss hit notification
func (ns *NotificationService) NotifyStopLossHit(chatID int64, data StopLossHitData) error {
	text, err := ns.templates.Render("notifications/stop_loss_hit", data)
	if err != nil {
		ns.log.Errorw("Failed to render stop_loss_hit template", "error", err)
		return err
	}

	return ns.bot.SendMessage(chatID, text)
}

// NotifyTakeProfitHit sends take profit hit notification
func (ns *NotificationService) NotifyTakeProfitHit(chatID int64, data TakeProfitHitData) error {
	text, err := ns.templates.Render("notifications/take_profit_hit", data)
	if err != nil {
		ns.log.Errorw("Failed to render take_profit_hit template", "error", err)
		return err
	}

	return ns.bot.SendMessage(chatID, text)
}

// NotifyCircuitBreaker sends circuit breaker notification
func (ns *NotificationService) NotifyCircuitBreaker(chatID int64, data CircuitBreakerData) error {
	text, err := ns.templates.Render("notifications/circuit_breaker", data)
	if err != nil {
		ns.log.Errorw("Failed to render circuit_breaker template", "error", err)
		return err
	}

	return ns.bot.SendMessage(chatID, text)
}

// NotifyExchangeDeactivated sends exchange deactivated notification
func (ns *NotificationService) NotifyExchangeDeactivated(chatID int64, data ExchangeDeactivatedData) error {
	text, err := ns.templates.Render("notifications/exchange_deactivated", data)
	if err != nil {
		ns.log.Errorw("Failed to render exchange_deactivated template", "error", err)
		return err
	}

	return ns.bot.SendMessage(chatID, text)
}
