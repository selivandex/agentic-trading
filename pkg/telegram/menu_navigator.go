package telegram

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Screen defines a menu screen with auto-parameter handling
type Screen struct {
	ID       string                                                                                       // Unique screen identifier
	Template string                                                                                       // Template path
	OnEnter  func(ctx context.Context, nav *MenuNavigator, params map[string]string) error                // Called when entering screen (params auto-parsed and saved)
	Data     func(ctx context.Context, session Session) (map[string]interface{}, error)                   // Data provider for template
	Keyboard func(ctx context.Context, nav *MenuNavigator, session Session) (InlineKeyboardMarkup, error) // Keyboard builder
}

// OptionScreenConfig defines configuration for option-based selection screens
type OptionScreenConfig struct {
	ID             string                                                                                                             // Screen ID
	Template       string                                                                                                             // Template path
	NextScreenID   string                                                                                                             // Next screen to navigate after selection
	ParamKey       string                                                                                                             // Key to save selected value in session (e.g., "m" for market type)
	Options        func(ctx context.Context, session Session) ([]MenuOption, error)                                                   // Options provider (can be dynamic)
	TemplateData   func(ctx context.Context, session Session) (map[string]interface{}, error)                                         // Additional template data (optional)
	OnEnter        func(ctx context.Context, nav *MenuNavigator, params map[string]string) error                                      // OnEnter callback (optional)
	CustomKeyboard func(ctx context.Context, nav *MenuNavigator, session Session, options []MenuOption) (InlineKeyboardMarkup, error) // Custom keyboard builder (optional)
}

// ListScreenConfig defines configuration for dynamic list screens (e.g., exchange accounts)
type ListScreenConfig struct {
	ID           string                                                                        // Screen ID
	Template     string                                                                        // Template path
	NextScreenID string                                                                        // Next screen to navigate after selection
	ParamKey     string                                                                        // Key to save selected item ID in session
	Items        func(ctx context.Context, session Session) ([]ListItem, error)                // Items provider
	TemplateData func(ctx context.Context, session Session) (map[string]interface{}, error)    // Additional template data (optional)
	ItemsKey     string                                                                        // Key for items in template data (default: "Items")
	OnEnter      func(ctx context.Context, nav *MenuNavigator, params map[string]string) error // OnEnter callback (optional)
}

// MenuNavigator provides enhanced navigation system for inline keyboard menus
// Features:
// - Clean Architecture (uses service layer, not repository directly)
// - Auto-parsing of callback parameters
// - OnEnter handlers for screens
// - Auto-save session state
// - Short callback format for Telegram limits
type MenuNavigator struct {
	sessionService SessionService
	bot            Bot
	templates      TemplateRenderer
	log            *logger.Logger
	sessionTTL     time.Duration
}

// NewMenuNavigator creates enhanced menu navigator
func NewMenuNavigator(
	sessionService SessionService,
	bot Bot,
	templates TemplateRenderer,
	log *logger.Logger,
	sessionTTL time.Duration,
) *MenuNavigator {
	if sessionTTL == 0 {
		sessionTTL = 30 * time.Minute
	}

	return &MenuNavigator{
		sessionService: sessionService,
		bot:            bot,
		templates:      templates,
		log:            log.With("component", "menu_navigator"),
		sessionTTL:     sessionTTL,
	}
}

// StartMenu starts new menu session
func (mn *MenuNavigator) StartMenu(ctx context.Context, telegramID int64, initialScreen *Screen, initialData map[string]interface{}) error {
	session, err := mn.sessionService.CreateSession(ctx, telegramID, initialScreen.ID, initialData, mn.sessionTTL)
	if err != nil {
		return errors.Wrap(err, "failed to create session")
	}

	// Call OnEnter if defined
	if initialScreen.OnEnter != nil {
		if err := initialScreen.OnEnter(ctx, mn, nil); err != nil {
			return errors.Wrap(err, "OnEnter handler failed")
		}
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

	session.SetMessageID(messageID)

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
func (mn *MenuNavigator) GetSession(ctx context.Context, telegramID int64) (Session, error) {
	return mn.sessionService.GetSession(ctx, telegramID)
}

// GetBot returns the bot instance for sending messages
func (mn *MenuNavigator) GetBot() Bot {
	return mn.bot
}

// RenderTemplate renders a template with given data
func (mn *MenuNavigator) RenderTemplate(templatePath string, data map[string]interface{}) (string, error) {
	return mn.templates.Render(templatePath, data)
}

// IsInMenu checks if user has active menu session
func (mn *MenuNavigator) IsInMenu(ctx context.Context, telegramID int64) (bool, error) {
	return mn.sessionService.SessionExists(ctx, telegramID)
}

// EndMenu ends menu session and cleans up
func (mn *MenuNavigator) EndMenu(ctx context.Context, telegramID int64) error {
	session, err := mn.sessionService.GetSession(ctx, telegramID)
	if err == nil && session.GetMessageID() > 0 {
		// Delete menu message
		mn.bot.DeleteMessageAsync(telegramID, session.GetMessageID(), "menu cleanup")
	}

	return mn.sessionService.DeleteSession(ctx, telegramID)
}

// showScreen displays a screen
func (mn *MenuNavigator) showScreen(ctx context.Context, session Session, screen *Screen) error {
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
	var keyboard InlineKeyboardMarkup
	if screen.Keyboard != nil {
		keyboard, err = screen.Keyboard(ctx, mn, session)
		if err != nil {
			return errors.Wrap(err, "failed to build keyboard")
		}

		mn.log.Debugw("Built screen keyboard",
			"screen_id", screen.ID,
			"keyboard_rows", len(keyboard.InlineKeyboard),
			"has_navigation_history", session.HasNavigationHistory(),
		)

		// Auto-add back button if navigation history exists
		if session.HasNavigationHistory() {
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
				NewInlineKeyboardRow(
					NewInlineKeyboardButtonData("⬅️ Back", "back"),
				),
			)
		}
	}

	telegramID := session.GetTelegramID()
	messageID := session.GetMessageID()

	// Try to update existing message
	if messageID > 0 {
		err = mn.bot.EditMessage(telegramID, messageID, text, &keyboard)
		if err != nil {
			mn.log.Debugw("Failed to edit message, sending new one",
				"error", err,
				"message_id", messageID,
			)
			session.SetMessageID(0)
		}
	}

	// Send new message if needed
	if session.GetMessageID() == 0 {
		sentMessageID, err := mn.bot.SendMessageWithOptions(telegramID, text, MessageOptions{
			Keyboard:  &keyboard,
			ParseMode: "Markdown",
		})
		if err != nil {
			return errors.Wrap(err, "failed to send message")
		}
		session.SetMessageID(sentMessageID)
	}

	// Save updated session
	return mn.sessionService.SaveSession(ctx, session, mn.sessionTTL)
}

// goBack navigates to previous screen
func (mn *MenuNavigator) goBack(ctx context.Context, session Session, screens map[string]*Screen) error {
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

// BuildOptionScreen creates a screen from OptionScreenConfig (DRY factory for option-based screens)
func (mn *MenuNavigator) BuildOptionScreen(config OptionScreenConfig) *Screen {
	return &Screen{
		ID:       config.ID,
		Template: config.Template,
		OnEnter:  config.OnEnter,
		Data: func(ctx context.Context, session Session) (map[string]interface{}, error) {
			// Get options
			options, err := config.Options(ctx, session)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get options")
			}

			// Base template data with options
			data := map[string]interface{}{
				"Options": options,
			}

			// Merge additional template data if provided
			if config.TemplateData != nil {
				additionalData, err := config.TemplateData(ctx, session)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get additional template data")
				}
				for k, v := range additionalData {
					data[k] = v
				}
			}

			return data, nil
		},
		Keyboard: func(ctx context.Context, nav *MenuNavigator, session Session) (InlineKeyboardMarkup, error) {
			// Get options
			options, err := config.Options(ctx, session)
			if err != nil {
				return InlineKeyboardMarkup{}, errors.Wrap(err, "failed to get options for keyboard")
			}

			// Use custom keyboard builder if provided
			if config.CustomKeyboard != nil {
				return config.CustomKeyboard(ctx, nav, session, options)
			}

			// Default keyboard builder: one button per option
			return mn.BuildOptionKeyboard(config.NextScreenID, config.ParamKey, options), nil
		},
	}
}

// BuildOptionKeyboard creates keyboard with one button per option
func (mn *MenuNavigator) BuildOptionKeyboard(nextScreenID, paramKey string, options []MenuOption) InlineKeyboardMarkup {
	var rows [][]InlineKeyboardButton
	for _, option := range options {
		buttonText := option.GetLabel()
		if emoji := option.GetEmoji(); emoji != "" {
			buttonText = fmt.Sprintf("%s %s", emoji, buttonText)
		}
		callbackData := mn.MakeCallback(nextScreenID, paramKey, option.GetValue())
		row := NewInlineKeyboardRow(
			NewInlineKeyboardButtonData(buttonText, callbackData),
		)
		rows = append(rows, row)
	}
	return NewInlineKeyboardMarkup(rows...)
}

// BuildListScreen creates a screen from ListScreenConfig (DRY factory for dynamic list screens)
func (mn *MenuNavigator) BuildListScreen(config ListScreenConfig) *Screen {
	return &Screen{
		ID:       config.ID,
		Template: config.Template,
		OnEnter:  config.OnEnter,
		Data: func(ctx context.Context, session Session) (map[string]interface{}, error) {
			// Get items
			items, err := config.Items(ctx, session)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get list items")
			}

			// Extract template data from items
			var templateItems []interface{}
			for _, item := range items {
				if item.TemplateData != nil {
					templateItems = append(templateItems, item.TemplateData)
				}
			}

			// Default key is "Items", but can be overridden
			itemsKey := config.ItemsKey
			if itemsKey == "" {
				itemsKey = "Items"
			}

			data := map[string]interface{}{
				itemsKey: templateItems,
			}

			// Merge additional template data if provided
			if config.TemplateData != nil {
				additionalData, err := config.TemplateData(ctx, session)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get additional template data")
				}
				for k, v := range additionalData {
					// Don't override items key
					if k != itemsKey {
						data[k] = v
					}
				}
			}

			return data, nil
		},
		Keyboard: func(ctx context.Context, nav *MenuNavigator, session Session) (InlineKeyboardMarkup, error) {
			items, err := config.Items(ctx, session)
			if err != nil {
				return InlineKeyboardMarkup{}, errors.Wrap(err, "failed to get items for keyboard")
			}

			var rows [][]InlineKeyboardButton
			for _, item := range items {
				callbackData := nav.MakeCallback(config.NextScreenID, config.ParamKey, item.ID)
				row := NewInlineKeyboardRow(
					NewInlineKeyboardButtonData(item.ButtonText, callbackData),
				)
				rows = append(rows, row)
			}
			return NewInlineKeyboardMarkup(rows...), nil
		},
	}
}

// BuildTextInputScreen creates a screen that expects text input from user
func (mn *MenuNavigator) BuildTextInputScreen(id, template string, onEnter func(ctx context.Context, nav *MenuNavigator, params map[string]string) error) *Screen {
	return &Screen{
		ID:       id,
		Template: template,
		OnEnter:  onEnter,
		Data:     nil, // Text input screens usually don't need dynamic data
		Keyboard: func(ctx context.Context, nav *MenuNavigator, session Session) (InlineKeyboardMarkup, error) {
			// No keyboard for text input screens (user types message)
			return InlineKeyboardMarkup{}, nil
		},
	}
}
