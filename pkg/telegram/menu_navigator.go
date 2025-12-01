package telegram

import (
	"context"
	"fmt"
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

	// Set current screen (important for text message routing)
	session.SetCurrentScreen(initialScreen.ID)

	// Call OnEnter if defined
	if initialScreen.OnEnter != nil {
		if err := initialScreen.OnEnter(ctx, mn, nil); err != nil {
			return errors.Wrap(err, "OnEnter handler failed")
		}
	}

	return mn.showScreen(ctx, session, initialScreen)
}

// HandleCallback processes callback with stored parameters
// Format: "cb:a1f2e9" (short key) or "back" (special case)
func (mn *MenuNavigator) HandleCallback(ctx context.Context, telegramID int64, messageID int, callbackData string, screens map[string]*Screen) error {
	session, err := mn.sessionService.GetSession(ctx, telegramID)
	if err != nil {
		return errors.Wrap(err, "no active menu session")
	}

	session.SetMessageID(messageID)

	// Handle back button (special case - no storage needed)
	if callbackData == "back" {
		return mn.goBack(ctx, session, screens)
	}

	// Retrieve callback parameters from session storage
	callbackParams, ok := session.GetCallbackData(callbackData)
	if !ok {
		mn.log.Warnw("Callback data not found in session (expired or invalid)",
			"callback_key", callbackData,
			"telegram_id", telegramID,
		)
		return fmt.Errorf("callback data expired or invalid")
	}

	// Extract screen ID from stored params
	screenIDInterface, ok := callbackParams["screen"]
	if !ok {
		return fmt.Errorf("screen ID not found in callback params")
	}
	screenID := screenIDInterface.(string)

	// Convert params to map[string]string for OnEnter handler
	params := make(map[string]string)
	for k, v := range callbackParams {
		if k != "screen" {
			if strVal, ok := v.(string); ok {
				params[k] = strVal
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

	// Set current screen (important for text message routing)
	session.SetCurrentScreen(screen.ID)

	// Save navigation state
	if err := mn.sessionService.SaveSession(ctx, session, mn.sessionTTL); err != nil {
		return errors.Wrap(err, "failed to save navigation state")
	}

	return mn.showScreen(ctx, session, screen)
}

// MakeCallback creates callback string with parameters stored in session
// Uses short keys to avoid Telegram's 64-byte callback_data limit
// Example: MakeCallback(session, "detail", "account_id", "uuid") → "cb:a1f2e9"
func (mn *MenuNavigator) MakeCallback(session Session, screenID string, params ...string) string {
	// Build params map
	paramsMap := map[string]interface{}{"screen": screenID}
	for i := 0; i < len(params); i += 2 {
		if i+1 < len(params) {
			paramsMap[params[i]] = params[i+1]
		}
	}

	// Generate short callback key (6 hex chars = 16M combinations)
	key := generateCallbackKey()

	// Store params in session
	session.SetCallbackData(key, paramsMap)

	return key
}

// generateCallbackKey generates a short random key for callback data
func generateCallbackKey() string {
	// Use timestamp + random for uniqueness (6 chars hex)
	now := time.Now().UnixNano()
	return fmt.Sprintf("cb:%x", now%0xFFFFFF) // 6 hex digits
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

	// Clear old callback data before building new keyboard (prevent memory leak from stale callbacks)
	session.ClearCallbackData()

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
			"note", "old_callback_data_cleared",
		)

		// Auto-add back button if navigation history exists
		if session.HasNavigationHistory() {
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
				NewInlineKeyboardRow(
					NewInlineKeyboardButtonData("⬅️ Back", "back"),
				),
			)
		}

		// Save session after building keyboard (callback data was added)
		if err := mn.sessionService.SaveSession(ctx, session, mn.sessionTTL); err != nil {
			return errors.Wrap(err, "failed to save session with callback data")
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

	// Set current screen (important for text message routing)
	session.SetCurrentScreen(prevScreenID)

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
			return mn.BuildOptionKeyboard(session, config.NextScreenID, config.ParamKey, options), nil
		},
	}
}

// BuildOptionKeyboard creates keyboard with one button per option
func (mn *MenuNavigator) BuildOptionKeyboard(session Session, nextScreenID, paramKey string, options []MenuOption) InlineKeyboardMarkup {
	var rows [][]InlineKeyboardButton
	for _, option := range options {
		buttonText := option.GetLabel()
		if emoji := option.GetEmoji(); emoji != "" {
			buttonText = fmt.Sprintf("%s %s", emoji, buttonText)
		}
		callbackData := mn.MakeCallback(session, nextScreenID, paramKey, option.GetValue())
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
				callbackData := nav.MakeCallback(session, config.NextScreenID, config.ParamKey, item.ID)
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
