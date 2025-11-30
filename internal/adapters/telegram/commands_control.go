package telegram

import (
	"context"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"

	"prometheus/internal/domain/position"
	"prometheus/internal/domain/user"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"
)

// ControlCommandHandler handles control commands (stop, settings)
type ControlCommandHandler struct {
	positionRepo position.Repository
	userService  userServiceInterface
	templates    *templates.Registry
	bot          *Bot
	log          *logger.Logger
}

// userServiceInterface defines interface for user operations in control handler
type userServiceInterface interface {
	GetByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	SetActive(ctx context.Context, userID uuid.UUID, active bool) error
	UpdateSettings(ctx context.Context, userID uuid.UUID, settings user.Settings) error
	ToggleCircuitBreaker(ctx context.Context, userID uuid.UUID) error
	ToggleNotifications(ctx context.Context, userID uuid.UUID) error
	UpdateRiskLevel(ctx context.Context, userID uuid.UUID, riskLevel string) error
	UpdateMaxPositions(ctx context.Context, userID uuid.UUID, maxPositions int) error
}

// NewControlCommandHandler creates a new control command handler
func NewControlCommandHandler(
	positionRepo position.Repository,
	userService userServiceInterface,
	tmpl *templates.Registry,
	bot *Bot,
	log *logger.Logger,
) *ControlCommandHandler {
	if tmpl == nil {
		tmpl = templates.Get()
	}

	return &ControlCommandHandler{
		positionRepo: positionRepo,
		userService:  userService,
		templates:    tmpl,
		bot:          bot,
		log:          log.With("component", "telegram_control_handler"),
	}
}

// HandleStop handles /stop command - pauses all trading
func (ch *ControlCommandHandler) HandleStop(ctx context.Context, chatID int64, userID uuid.UUID) error {
	// Get user
	usr, err := ch.userService.GetByID(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "failed to get user")
	}

	// Check if already stopped
	if !usr.IsActive {
		msg, err := ch.templates.Render("telegram/already_stopped", nil)
		if err != nil {
			return errors.Wrap(err, "failed to render already_stopped template")
		}
		return ch.bot.SendMessage(chatID, msg)
	}

	// Pause user trading (via service - business logic)
	if err := ch.userService.SetActive(ctx, userID, false); err != nil {
		return errors.Wrap(err, "failed to deactivate user")
	}

	ch.log.Infow("User trading paused", "user_id", userID)

	// Render confirmation with current positions info
	openPositions, _ := ch.positionRepo.GetOpenByUser(ctx, userID)

	data := map[string]interface{}{
		"OpenPositionsCount": len(openPositions),
		"WillClosePositions": false, // For MVP, keep positions open
	}

	msg, err := ch.templates.Render("telegram/trading_stopped", data)
	if err != nil {
		return errors.Wrap(err, "failed to render trading_stopped template")
	}

	return ch.bot.SendMessage(chatID, msg)
}

// HandleSettings handles /settings command - manages user preferences
func (ch *ControlCommandHandler) HandleSettings(ctx context.Context, chatID int64, userID uuid.UUID) error {
	// Get user (via service)
	usr, err := ch.userService.GetByID(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "failed to get user")
	}

	// Build settings keyboard
	keyboard := ch.buildSettingsKeyboard(usr)

	// Prepare template data
	data := map[string]interface{}{
		"RiskLevel":          usr.Settings.RiskLevel,
		"MaxPositions":       usr.Settings.MaxPositions,
		"CircuitBreakerOn":   usr.Settings.CircuitBreakerOn,
		"NotificationsOn":    usr.Settings.NotificationsOn,
		"MaxDailyDrawdown":   usr.Settings.MaxDailyDrawdown,
		"MaxPositionSizeUSD": usr.Settings.MaxPositionSizeUSD,
	}

	msg, err := ch.templates.Render("telegram/settings", data)
	if err != nil {
		return errors.Wrap(err, "failed to render settings template")
	}

	return ch.bot.SendMessageWithKeyboard(chatID, msg, keyboard)
}

// buildSettingsKeyboard creates inline keyboard for settings
func (ch *ControlCommandHandler) buildSettingsKeyboard(usr *user.User) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎯 Risk Level", "settings:risk"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Max Positions", "settings:max_positions"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				ch.formatToggle("Circuit Breaker", usr.Settings.CircuitBreakerOn),
				"settings:toggle_circuit_breaker",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				ch.formatToggle("Notifications", usr.Settings.NotificationsOn),
				"settings:toggle_notifications",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Done", "settings:done"),
		),
	)
}

// formatToggle formats toggle button text with status
func (ch *ControlCommandHandler) formatToggle(label string, enabled bool) string {
	if enabled {
		return fmt.Sprintf("%s: ✅", label)
	}
	return fmt.Sprintf("%s: ❌", label)
}

// HandleSettingsCallback handles callback queries from settings keyboard
func (ch *ControlCommandHandler) HandleSettingsCallback(ctx context.Context, chatID int64, userID uuid.UUID, action string) error {
	usr, err := ch.userService.GetByID(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "failed to get user")
	}

	switch action {
	case "risk":
		return ch.handleRiskLevelChange(ctx, chatID, usr)

	case "max_positions":
		return ch.handleMaxPositionsChange(ctx, chatID, usr)

	case "toggle_circuit_breaker":
		if err := ch.userService.ToggleCircuitBreaker(ctx, userID); err != nil {
			return err
		}
		return ch.HandleSettings(ctx, chatID, userID)

	case "toggle_notifications":
		if err := ch.userService.ToggleNotifications(ctx, userID); err != nil {
			return err
		}
		return ch.HandleSettings(ctx, chatID, userID)

	case "done":
		msg, err := ch.templates.Render("telegram/settings_saved", nil)
		if err != nil {
			return errors.Wrap(err, "failed to render settings_saved template")
		}
		return ch.bot.SendMessage(chatID, msg)

	default:
		ch.log.Warn("Unknown settings action", "action", action)
		return nil
	}
}

// handleRiskLevelChange handles risk level change
func (ch *ControlCommandHandler) handleRiskLevelChange(ctx context.Context, chatID int64, usr *user.User) error {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Conservative", "settings:set_risk:conservative"),
			tgbotapi.NewInlineKeyboardButtonData("Moderate", "settings:set_risk:moderate"),
			tgbotapi.NewInlineKeyboardButtonData("Aggressive", "settings:set_risk:aggressive"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← Back", "settings:back"),
		),
	)

	msg, err := ch.templates.Render("telegram/settings_risk_select", map[string]interface{}{
		"CurrentRisk": usr.Settings.RiskLevel,
	})
	if err != nil {
		return errors.Wrap(err, "failed to render settings_risk_select template")
	}

	return ch.bot.SendMessageWithKeyboard(chatID, msg, keyboard)
}

// handleMaxPositionsChange handles max positions change
func (ch *ControlCommandHandler) handleMaxPositionsChange(ctx context.Context, chatID int64, usr *user.User) error {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1", "settings:set_max_pos:1"),
			tgbotapi.NewInlineKeyboardButtonData("2", "settings:set_max_pos:2"),
			tgbotapi.NewInlineKeyboardButtonData("3", "settings:set_max_pos:3"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("4", "settings:set_max_pos:4"),
			tgbotapi.NewInlineKeyboardButtonData("5", "settings:set_max_pos:5"),
			tgbotapi.NewInlineKeyboardButtonData("6", "settings:set_max_pos:6"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← Back", "settings:back"),
		),
	)

	msg, err := ch.templates.Render("telegram/settings_max_positions_select", map[string]interface{}{
		"CurrentMaxPositions": usr.Settings.MaxPositions,
	})
	if err != nil {
		return errors.Wrap(err, "failed to render settings_max_positions_select template")
	}

	return ch.bot.SendMessageWithKeyboard(chatID, msg, keyboard)
}

// HandleSettingsValueUpdate handles updating specific setting values
func (ch *ControlCommandHandler) HandleSettingsValueUpdate(ctx context.Context, chatID int64, userID uuid.UUID, setting, value string) error {
	switch setting {
	case "risk":
		if value == "conservative" || value == "moderate" || value == "aggressive" {
			if err := ch.userService.UpdateRiskLevel(ctx, userID, value); err != nil {
				return errors.Wrap(err, "failed to update risk level")
			}
		}

	case "max_pos":
		if maxPos, err := strconv.Atoi(value); err == nil && maxPos >= 1 && maxPos <= 10 {
			if err := ch.userService.UpdateMaxPositions(ctx, userID, maxPos); err != nil {
				return errors.Wrap(err, "failed to update max positions")
			}
		}
	}

	return ch.HandleSettings(ctx, chatID, userID)
}
