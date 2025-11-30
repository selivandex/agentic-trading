package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/position"
	"prometheus/internal/domain/user"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"
)

// QueryCommandHandler handles query commands (status, portfolio)
type QueryCommandHandler struct {
	positionRepo position.Repository
	userService  queryUserService
	templates    *templates.Registry
	bot          *Bot
	log          *logger.Logger
}

// queryUserService defines interface for user operations in query handler
type queryUserService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*user.User, error)
}

// NewQueryCommandHandler creates a new query command handler
func NewQueryCommandHandler(
	positionRepo position.Repository,
	userService queryUserService,
	tmpl *templates.Registry,
	bot *Bot,
	log *logger.Logger,
) *QueryCommandHandler {
	if tmpl == nil {
		tmpl = templates.Get()
	}

	return &QueryCommandHandler{
		positionRepo: positionRepo,
		userService:  userService,
		templates:    tmpl,
		bot:          bot,
		log:          log.With("component", "telegram_query_handler"),
	}
}

// HandleStatus handles /status command
func (qh *QueryCommandHandler) HandleStatus(ctx context.Context, chatID int64, userID uuid.UUID) error {
	// Get user (via service)
	usr, err := qh.userService.GetByID(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "failed to get user")
	}

	// Get open positions
	openPositions, err := qh.positionRepo.GetOpenByUser(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "failed to get open positions")
	}

	// Calculate totals
	totalPnL := decimal.Zero
	totalInvested := decimal.Zero

	for _, pos := range openPositions {
		totalPnL = totalPnL.Add(pos.UnrealizedPnL)
		invested := pos.EntryPrice.Mul(pos.Size)
		totalInvested = totalInvested.Add(invested)
	}

	totalPnLPercent := 0.0
	if !totalInvested.IsZero() {
		totalPnLPercent = totalPnL.Div(totalInvested).Mul(decimal.NewFromInt(100)).InexactFloat64()
	}

	// Get circuit breaker status from user settings
	circuitBreakerStatus := "✅ Active"
	if !usr.Settings.CircuitBreakerOn {
		circuitBreakerStatus = "⏸️ Paused"
	}
	if !usr.IsActive {
		circuitBreakerStatus = "🛑 Stopped"
	}

	// Prepare template data
	data := map[string]interface{}{
		"FirstName":            usr.FirstName,
		"IsActive":             usr.IsActive,
		"OpenPositionsCount":   len(openPositions),
		"TotalPnL":             totalPnL.InexactFloat64(),
		"TotalPnLPercent":      totalPnLPercent,
		"TotalInvested":        totalInvested.InexactFloat64(),
		"CircuitBreakerStatus": circuitBreakerStatus,
		"RiskLevel":            usr.Settings.RiskLevel,
		"MaxPositions":         usr.Settings.MaxPositions,
		"Timestamp":            time.Now().Format("2006-01-02 15:04"),
	}

	// Render template
	msg, err := qh.templates.Render("telegram/status", data)
	if err != nil {
		return errors.Wrap(err, "failed to render status template")
	}

	return qh.bot.SendMessage(chatID, msg)
}

// HandlePortfolio handles /portfolio command
func (qh *QueryCommandHandler) HandlePortfolio(ctx context.Context, chatID int64, userID uuid.UUID) error {
	// Get open positions
	openPositions, err := qh.positionRepo.GetOpenByUser(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "failed to get open positions")
	}

	if len(openPositions) == 0 {
		msg, err := qh.templates.Render("telegram/portfolio_empty", nil)
		if err != nil {
			return errors.Wrap(err, "failed to render portfolio_empty template")
		}
		return qh.bot.SendMessage(chatID, msg)
	}

	// Prepare positions data for template
	positionsData := make([]map[string]interface{}, 0, len(openPositions))

	for _, pos := range openPositions {
		// Calculate PnL percent
		invested := pos.EntryPrice.Mul(pos.Size)
		pnlPercent := 0.0
		if !invested.IsZero() {
			pnlPercent = pos.UnrealizedPnL.Div(invested).Mul(decimal.NewFromInt(100)).InexactFloat64()
		}

		// Format duration
		duration := time.Since(pos.OpenedAt)
		durationStr := formatDuration(duration)

		positionsData = append(positionsData, map[string]interface{}{
			"Symbol":        pos.Symbol,
			"Side":          string(pos.Side),
			"EntryPrice":    pos.EntryPrice.InexactFloat64(),
			"CurrentPrice":  pos.CurrentPrice.InexactFloat64(),
			"Amount":        pos.Size.InexactFloat64(),
			"UnrealizedPnL": pos.UnrealizedPnL.InexactFloat64(),
			"PnLPercent":    pnlPercent,
			"StopLoss":      pos.StopLossPrice.InexactFloat64(),
			"TakeProfit":    pos.TakeProfitPrice.InexactFloat64(),
			"Duration":      durationStr,
			"OpenedAt":      pos.OpenedAt.Format("Jan 02, 15:04"),
		})
	}

	// Prepare template data
	data := map[string]interface{}{
		"Positions":  positionsData,
		"TotalCount": len(openPositions),
		"Timestamp":  time.Now().Format("2006-01-02 15:04"),
	}

	// Render template
	msg, err := qh.templates.Render("telegram/portfolio", data)
	if err != nil {
		return errors.Wrap(err, "failed to render portfolio template")
	}

	return qh.bot.SendMessage(chatID, msg)
}

// formatDuration formats duration in human-readable format
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "< 1m"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		return fmt.Sprintf("%dm", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		return fmt.Sprintf("%dh", hours)
	}
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	if hours > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	return fmt.Sprintf("%dd", days)
}
