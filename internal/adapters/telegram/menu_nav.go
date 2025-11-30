package telegram

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"prometheus/internal/domain/menu_session"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"
)

// MenuSessionService defines interface for menu session business logic
type MenuSessionService interface {
	GetSession(ctx context.Context, telegramID int64) (*menu_session.Session, error)
	SaveSession(ctx context.Context, session *menu_session.Session, ttl time.Duration) error
	DeleteSession(ctx context.Context, telegramID int64) error
	SessionExists(ctx context.Context, telegramID int64) (bool, error)
	CreateSession(ctx context.Context, telegramID int64, initialScreen string, initialData map[string]interface{}, ttl time.Duration) (*menu_session.Session, error)
}

// MenuNavigator provides enhanced navigation system for inline keyboard menus
// Features:
// - Clean Architecture (uses service layer, not repository directly)
// - Auto-parsing of callback parameters
// - OnEnter handlers for screens
// - Auto-save session state
// - Short callback format for Telegram limits
type MenuNavigator struct {
	sessionService MenuSessionService
	bot            *Bot
	templates      *templates.Registry
	log            *logger.Logger
	sessionTTL     time.Duration
}

// Screen defines a menu screen with auto-parameter handling
type Screen struct {
	ID       string                                                                                                              // Unique screen identifier
	Template string                                                                                                              // Template path
	OnEnter  func(ctx context.Context, nav *MenuNavigator, params map[string]string) error                                       // Called when entering screen (params auto-parsed and saved)
	Data     func(ctx context.Context, session *menu_session.Session) (map[string]interface{}, error)                            // Data provider for template
	Keyboard func(ctx context.Context, nav *MenuNavigator, session *menu_session.Session) (tgbotapi.InlineKeyboardMarkup, error) // Keyboard builder
}

// NewMenuNavigator creates enhanced menu navigator
func NewMenuNavigator(
	sessionService MenuSessionService,
	bot *Bot,
	tmpl *templates.Registry,
	log *logger.Logger,
	sessionTTL time.Duration,
) *MenuNavigator {
	if tmpl == nil {
		tmpl = templates.Get()
	}
	if sessionTTL == 0 {
		sessionTTL = 30 * time.Minute
	}

	return &MenuNavigator{
		sessionService: sessionService,
		bot:            bot,
		templates:      tmpl,
		log:            log.With("component", "menu_navigator"),
		sessionTTL:     sessionTTL,
	}
}

// StartMenu starts new menu session
func (mn *MenuNavigator) StartMenu(ctx context.Context, telegramID int64, initialScreen *Screen, initialData map[string]interface{}) error {
	session := menu_session.NewSession(telegramID, initialScreen.ID, initialData)

	// Call OnEnter if defined
	if initialScreen.OnEnter != nil {
		if err := initialScreen.OnEnter(ctx, mn, nil); err != nil {
			return errors.Wrap(err, "OnEnter handler failed")
		}
	}

	// Save session
	if err := mn.sessionService.SaveSession(ctx, session, mn.sessionTTL); err != nil {
		return err
	}

	return mn.showScreen(ctx, session, initialScreen)
}

// HandleCallback processes callback with auto-parameter parsing
// Format: screenID:key=val:key2=val2 or just "back"
func (mn *MenuNavigator) HandleCallback(ctx context.Context, telegramID int64, messageID int, callbackData string, screens map[string]*Screen) error {
	session, err := mn.sessionService.GetSession(ctx, telegramID)
	if err != nil {
		return errors.Wrap(err, "no active menu session")
	}

	session.MessageID = messageID
	session.UpdatedAt = time.Now()

	// Handle back button
	if callbackData == "back" {
		return mn.goBack(ctx, session, screens)
	}

	// Parse callback: screenID:key=val:key2=val2
	parts := strings.Split(callbackData, ":")
	screenID := parts[0]
	params := make(map[string]string)

	// Parse parameters
	for i := 1; i < len(parts); i++ {
		kv := strings.SplitN(parts[i], "=", 2)
		if len(kv) == 2 {
			// URL decode values
			if decoded, err := url.QueryUnescape(kv[1]); err == nil {
				params[kv[0]] = decoded
			} else {
				params[kv[0]] = kv[1]
			}
		}
	}

	// Find target screen
	screen, exists := screens[screenID]
	if !exists {
		return fmt.Errorf("screen not found: %s", screenID)
	}

	// Auto-save parameters to session.Data
	for k, v := range params {
		session.SetData(k, v)
	}

	// Save session with updated params
	if err := mn.sessionService.SaveSession(ctx, session, mn.sessionTTL); err != nil {
		return errors.Wrap(err, "failed to save session with params")
	}

	// Call OnEnter handler if defined
	if screen.OnEnter != nil {
		if err := screen.OnEnter(ctx, mn, params); err != nil {
			return errors.Wrap(err, "OnEnter handler failed")
		}
	}

	// Push to navigation stack
	session.PushScreen(screen.ID)

	// Save navigation state
	if err := mn.sessionService.SaveSession(ctx, session, mn.sessionTTL); err != nil {
		return errors.Wrap(err, "failed to save navigation state")
	}

	return mn.showScreen(ctx, session, screen)
}

// MakeCallback creates callback string with parameters
// Example: MakeCallback("select_exchange", "account_id", "123") → "select_exchange:account_id=123"
func (mn *MenuNavigator) MakeCallback(screenID string, params ...string) string {
	if len(params) == 0 {
		return screenID
	}

	parts := []string{screenID}
	for i := 0; i < len(params); i += 2 {
		if i+1 < len(params) {
			key := params[i]
			value := url.QueryEscape(params[i+1]) // URL encode for safety
			parts = append(parts, fmt.Sprintf("%s=%s", key, value))
		}
	}

	return strings.Join(parts, ":")
}

// GetSession retrieves current session
func (mn *MenuNavigator) GetSession(ctx context.Context, telegramID int64) (*menu_session.Session, error) {
	return mn.sessionService.GetSession(ctx, telegramID)
}

// IsInMenu checks if user has active menu session
func (mn *MenuNavigator) IsInMenu(ctx context.Context, telegramID int64) (bool, error) {
	return mn.sessionService.SessionExists(ctx, telegramID)
}

// EndMenu ends menu session and cleans up
func (mn *MenuNavigator) EndMenu(ctx context.Context, telegramID int64) error {
	session, err := mn.sessionService.GetSession(ctx, telegramID)
	if err == nil && session.MessageID > 0 {
		// Delete menu message
		mn.bot.DeleteMessageAsync(telegramID, session.MessageID, "menu cleanup")
	}

	return mn.sessionService.DeleteSession(ctx, telegramID)
}

// showScreen displays a screen
func (mn *MenuNavigator) showScreen(ctx context.Context, session *menu_session.Session, screen *Screen) error {
	// Get template data
	var data map[string]interface{}
	var err error
	if screen.Data != nil {
		data, err = screen.Data(ctx, session)
		if err != nil {
			return errors.Wrap(err, "failed to get screen data")
		}
	}

	// Render template
	text, err := mn.templates.Render(screen.Template, data)
	if err != nil {
		return errors.Wrap(err, "failed to render screen template")
	}

	// Build keyboard
	var keyboard tgbotapi.InlineKeyboardMarkup
	if screen.Keyboard != nil {
		keyboard, err = screen.Keyboard(ctx, mn, session)
		if err != nil {
			return errors.Wrap(err, "failed to build keyboard")
		}

		// Auto-add back button if navigation history exists
		if session.HasNavigationHistory() {
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("⬅️ Back", "back"),
				),
			)
		}
	}

	// Try to update existing message
	if session.MessageID > 0 {
		edit := tgbotapi.NewEditMessageText(session.TelegramID, session.MessageID, text)
		edit.ParseMode = "Markdown"
		if screen.Keyboard != nil {
			edit.ReplyMarkup = &keyboard
		}

		if _, err := mn.bot.GetAPI().Send(edit); err != nil {
			mn.log.Debugw("Failed to edit message, sending new one",
				"error", err,
				"message_id", session.MessageID,
			)
			session.MessageID = 0
		}
	}

	// Send new message if needed
	if session.MessageID == 0 {
		msg := tgbotapi.NewMessage(session.TelegramID, text)
		msg.ParseMode = "Markdown"
		if screen.Keyboard != nil {
			msg.ReplyMarkup = keyboard
		}

		sentMsg, err := mn.bot.GetAPI().Send(msg)
		if err != nil {
			return errors.Wrap(err, "failed to send message")
		}
		session.MessageID = sentMsg.MessageID
	}

	// Save updated session
	return mn.sessionService.SaveSession(ctx, session, mn.sessionTTL)
}

// goBack navigates to previous screen
func (mn *MenuNavigator) goBack(ctx context.Context, session *menu_session.Session, screens map[string]*Screen) error {
	prevScreenID, ok := session.PopScreen()
	if !ok {
		return fmt.Errorf("navigation stack is empty")
	}

	// Find previous screen
	screen, exists := screens[prevScreenID]
	if !exists {
		return fmt.Errorf("previous screen not found: %s", prevScreenID)
	}

	// Save navigation state
	if err := mn.sessionService.SaveSession(ctx, session, mn.sessionTTL); err != nil {
		return errors.Wrap(err, "failed to save navigation state")
	}

	return mn.showScreen(ctx, session, screen)
}
