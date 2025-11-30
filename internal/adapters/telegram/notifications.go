package telegram

import (
	"time"

	"github.com/shopspring/decimal"

	"prometheus/internal/domain/position"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"
)

// NotificationService handles sending templated notifications to users via Telegram
type NotificationService struct {
	bot       *Bot
	templates *templates.Registry
	log       *logger.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(bot *Bot, tmpl *templates.Registry, log *logger.Logger) *NotificationService {
	return &NotificationService{
		bot:       bot,
		templates: tmpl,
		log:       log.With("component", "telegram_notifications"),
	}
}

// PositionOpenedData represents data for position opened notification
type PositionOpenedData struct {
	Symbol     string
	Side       string
	EntryPrice float64
	Amount     float64
	StopLoss   float64
	TakeProfit float64
	Exchange   string
	Reasoning  string
	Timestamp  time.Time
}

// PositionClosedData represents data for position closed notification
type PositionClosedData struct {
	Symbol      string
	Side        string
	EntryPrice  float64
	ExitPrice   float64
	PnL         float64
	PnLPercent  float64
	Duration    time.Duration
	CloseReason string
	Timestamp   time.Time
}

// StopLossHitData represents data for stop loss hit notification
type StopLossHitData struct {
	Symbol      string
	Side        string
	EntryPrice  float64
	StopPrice   float64
	Loss        float64
	LossPercent float64
	Timestamp   time.Time
}

// TakeProfitHitData represents data for take profit hit notification
type TakeProfitHitData struct {
	Symbol        string
	Side          string
	EntryPrice    float64
	TargetPrice   float64
	Profit        float64
	ProfitPercent float64
	Timestamp     time.Time
}

// CircuitBreakerData represents data for circuit breaker notification
type CircuitBreakerData struct {
	Reason          string
	DailyLoss       float64
	LossPercent     float64
	ConsecutiveLoss int
	Timestamp       time.Time
	Action          string
}

// OpportunityFoundData represents data for opportunity found notification
type OpportunityFoundData struct {
	Symbol     string
	Direction  string
	Entry      float64
	StopLoss   float64
	TakeProfit float64
	Confidence float64
	Strategy   string
	Reasoning  string
	Timestamp  time.Time
}

// DailyReportData represents data for daily report notification
type DailyReportData struct {
	Date            string
	TotalPnL        float64
	TotalPnLPercent float64
	TradesCount     int
	WinRate         float64
	BestTrade       string
	WorstTrade      string
	OpenPositions   int
	PortfolioValue  float64
}

// ExchangeDeactivatedData represents data for exchange deactivation notification
type ExchangeDeactivatedData struct {
	Exchange     string
	Label        string
	Reason       string
	ErrorMessage string
	IsTestnet    bool
}

// NotifyPositionOpened sends position opened notification
func (ns *NotificationService) NotifyPositionOpened(chatID int64, data PositionOpenedData) error {
	text, err := ns.templates.Render("notifications/position_opened", data)
	if err != nil {
		return errors.Wrap(err, "failed to render position_opened template")
	}

	return ns.bot.SendNotificationWithRetry(chatID, text, 3)
}

// NotifyPositionClosed sends position closed notification
func (ns *NotificationService) NotifyPositionClosed(chatID int64, data PositionClosedData) error {
	text, err := ns.templates.Render("notifications/position_closed", data)
	if err != nil {
		return errors.Wrap(err, "failed to render position_closed template")
	}

	return ns.bot.SendNotificationWithRetry(chatID, text, 3)
}

// NotifyStopLossHit sends stop loss hit notification
func (ns *NotificationService) NotifyStopLossHit(chatID int64, data StopLossHitData) error {
	text, err := ns.templates.Render("notifications/stop_loss_hit", data)
	if err != nil {
		return errors.Wrap(err, "failed to render stop_loss_hit template")
	}

	return ns.bot.SendNotificationWithRetry(chatID, text, 3)
}

// NotifyTakeProfitHit sends take profit hit notification
func (ns *NotificationService) NotifyTakeProfitHit(chatID int64, data TakeProfitHitData) error {
	text, err := ns.templates.Render("notifications/take_profit_hit", data)
	if err != nil {
		return errors.Wrap(err, "failed to render take_profit_hit template")
	}

	return ns.bot.SendNotificationWithRetry(chatID, text, 3)
}

// NotifyCircuitBreaker sends circuit breaker notification
func (ns *NotificationService) NotifyCircuitBreaker(chatID int64, data CircuitBreakerData) error {
	text, err := ns.templates.Render("notifications/circuit_breaker", data)
	if err != nil {
		return errors.Wrap(err, "failed to render circuit_breaker template")
	}

	return ns.bot.SendNotificationWithRetry(chatID, text, 3)
}

// NotifyOpportunityFound sends opportunity found notification
func (ns *NotificationService) NotifyOpportunityFound(chatID int64, data OpportunityFoundData) error {
	text, err := ns.templates.Render("notifications/opportunity_found", data)
	if err != nil {
		return errors.Wrap(err, "failed to render opportunity_found template")
	}

	return ns.bot.SendNotificationWithRetry(chatID, text, 3)
}

// NotifyDailyReport sends daily report notification
func (ns *NotificationService) NotifyDailyReport(chatID int64, data DailyReportData) error {
	text, err := ns.templates.Render("notifications/daily_report", data)
	if err != nil {
		return errors.Wrap(err, "failed to render daily_report template")
	}

	return ns.bot.SendNotificationWithRetry(chatID, text, 3)
}

// NotifyExchangeDeactivated sends exchange deactivated notification
func (ns *NotificationService) NotifyExchangeDeactivated(chatID int64, data ExchangeDeactivatedData) error {
	text, err := ns.templates.Render("telegram/exchange_deactivated", data)
	if err != nil {
		return errors.Wrap(err, "failed to render exchange_deactivated template")
	}

	return ns.bot.SendNotificationWithRetry(chatID, text, 3)
}

// NotifyPosition sends a position notification (convenience method)
func (ns *NotificationService) NotifyPosition(chatID int64, pos *position.Position, eventType string) error {
	switch eventType {
	case "opened":
		data := PositionOpenedData{
			Symbol:     pos.Symbol,
			Side:       string(pos.Side),
			EntryPrice: pos.EntryPrice.InexactFloat64(),
			Amount:     pos.Size.InexactFloat64(),
			StopLoss:   pos.StopLossPrice.InexactFloat64(),
			TakeProfit: pos.TakeProfitPrice.InexactFloat64(),
			Exchange:   pos.ExchangeAccountID.String(), // Convert UUID to string
			Timestamp:  pos.OpenedAt,
		}
		return ns.NotifyPositionOpened(chatID, data)

	case "closed":
		if pos.ClosedAt == nil {
			return errors.New("position is not closed")
		}

		duration := pos.ClosedAt.Sub(pos.OpenedAt)
		pnl := pos.RealizedPnL.InexactFloat64()

		// Calculate PnL percentage
		invested := pos.EntryPrice.Mul(pos.Size)
		pnlPct := 0.0
		if !invested.IsZero() {
			pnlPct = pos.RealizedPnL.Div(invested).Mul(decimal.NewFromInt(100)).InexactFloat64()
		}

		data := PositionClosedData{
			Symbol:      pos.Symbol,
			Side:        string(pos.Side),
			EntryPrice:  pos.EntryPrice.InexactFloat64(),
			ExitPrice:   pos.CurrentPrice.InexactFloat64(), // Use current price as exit price
			PnL:         pnl,
			PnLPercent:  pnlPct,
			Duration:    duration,
			CloseReason: string(pos.Status), // Use status as close reason
			Timestamp:   *pos.ClosedAt,
		}
		return ns.NotifyPositionClosed(chatID, data)

	default:
		return errors.Wrapf(errors.ErrInvalidInput, "unknown position event type: %s", eventType)
	}
}
