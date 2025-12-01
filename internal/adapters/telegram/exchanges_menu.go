package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/services/exchange"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/telegram"
)

// ExchangesServiceInterface defines interface for exchange operations
type ExchangesServiceInterface interface {
	GetActiveUserAccounts(ctx context.Context, userID uuid.UUID) ([]*exchange_account.ExchangeAccount, error)
	GetAccount(ctx context.Context, accountID uuid.UUID) (*exchange_account.ExchangeAccount, error)
	CreateAccount(ctx context.Context, input exchange.CreateAccountInput) (*exchange_account.ExchangeAccount, error)
	UpdateAccount(ctx context.Context, input exchange.UpdateAccountInput) (*exchange_account.ExchangeAccount, error)
	SetAccountActive(ctx context.Context, accountID uuid.UUID, active bool) error
}

// ExchangesFlowKeys defines parameter keys for callback data
type ExchangesFlowKeys struct {
	AccountID    string // "acc" - exchange account ID
	ExchangeType string // "ex" - exchange type (binance, bybit, okx, etc.)
	IsTestnet    string // "test" - testnet flag (true/false)
	Label        string // "lbl" - account label
	APIKey       string // "key" - API key
	Secret       string // "sec" - secret key
	Passphrase   string // "pass" - passphrase (for OKX)
	CredIndex    string // "idx" - current credential field index
}

var exchangeKeys = ExchangesFlowKeys{
	AccountID:    "acc",
	ExchangeType: "ex",
	IsTestnet:    "test",
	Label:        "lbl",
	APIKey:       "key",
	Secret:       "sec",
	Passphrase:   "pass",
	CredIndex:    "idx",
}

// ExchangeTypeOption represents an exchange type selection option
type ExchangeTypeOption struct {
	Emoji string
	Label string
	Value exchange_account.ExchangeType
}

// Implement telegram.MenuOption interface
func (e ExchangeTypeOption) GetValue() string { return string(e.Value) }
func (e ExchangeTypeOption) GetLabel() string { return e.Label }
func (e ExchangeTypeOption) GetEmoji() string { return e.Emoji }

// TestnetOption represents a testnet selection option
type TestnetOption struct {
	Emoji       string
	Label       string
	Value       string
	Description string
}

// Implement telegram.MenuOption interface
func (t TestnetOption) GetValue() string { return t.Value }
func (t TestnetOption) GetLabel() string { return t.Label }
func (t TestnetOption) GetEmoji() string { return t.Emoji }

var (
	// exchangeTypeOptions defines available exchange types
	exchangeTypeOptions = []ExchangeTypeOption{
		{
			Emoji: exchange_account.GetExchangeEmoji(exchange_account.ExchangeBinance),
			Label: exchange_account.GetExchangeDisplayName(exchange_account.ExchangeBinance),
			Value: exchange_account.ExchangeBinance,
		},
		{
			Emoji: exchange_account.GetExchangeEmoji(exchange_account.ExchangeBybit),
			Label: exchange_account.GetExchangeDisplayName(exchange_account.ExchangeBybit),
			Value: exchange_account.ExchangeBybit,
		},
		{
			Emoji: exchange_account.GetExchangeEmoji(exchange_account.ExchangeOKX),
			Label: exchange_account.GetExchangeDisplayName(exchange_account.ExchangeOKX),
			Value: exchange_account.ExchangeOKX,
		},
		{
			Emoji: exchange_account.GetExchangeEmoji(exchange_account.ExchangeKucoin),
			Label: exchange_account.GetExchangeDisplayName(exchange_account.ExchangeKucoin),
			Value: exchange_account.ExchangeKucoin,
		},
		{
			Emoji: exchange_account.GetExchangeEmoji(exchange_account.ExchangeGate),
			Label: exchange_account.GetExchangeDisplayName(exchange_account.ExchangeGate),
			Value: exchange_account.ExchangeGate,
		},
	}

	// testnetOptions defines testnet selection options
	testnetOptions = []TestnetOption{
		{
			Emoji:       "🚀",
			Label:       "Production",
			Value:       "false",
			Description: "Live trading with real funds",
		},
		{
			Emoji:       "🧪",
			Label:       "Testnet",
			Value:       "true",
			Description: "Practice trading with test funds",
		},
	}
)

// ExchangesMenuService handles /exchanges flow using MenuNavigator framework
type ExchangesMenuService struct {
	menuNav         *telegram.MenuNavigator
	exchangeService ExchangesServiceInterface
	userService     UserService
	log             *logger.Logger
}

// NewExchangesMenuService creates exchanges menu service
func NewExchangesMenuService(
	menuNav *telegram.MenuNavigator,
	exchangeService ExchangesServiceInterface,
	userService UserService,
	log *logger.Logger,
) *ExchangesMenuService {
	return &ExchangesMenuService{
		menuNav:         menuNav,
		exchangeService: exchangeService,
		userService:     userService,
		log:             log.With("component", "exchanges_menu"),
	}
}

// StartExchanges starts exchanges flow with list of accounts
func (ems *ExchangesMenuService) StartExchanges(ctx context.Context, userID uuid.UUID, telegramID int64) error {
	initialData := map[string]interface{}{
		"user_id":    userID.String(),
		"_menu_type": "exchanges", // Mark session as exchanges menu
	}

	screen := ems.buildListScreen()
	return ems.menuNav.StartMenu(ctx, telegramID, screen, initialData)
}

// HandleCallback processes exchanges menu callbacks
func (ems *ExchangesMenuService) HandleCallback(ctx context.Context, userID interface{}, telegramID int64, messageID int, callbackKey string) error {
	// Note: "cancel" and "back" are handled by MenuRegistry before reaching here

	// Check if this is a toggle callback by looking at stored params
	session, _ := ems.menuNav.GetSession(ctx, telegramID)
	if session != nil {
		callbackParams, ok := session.GetCallbackData(callbackKey)
		if ok {
			if screenID, exists := callbackParams["screen"]; exists && screenID == "toggle" {
				// Handle toggle separately (doesn't navigate, just updates message)
				return ems.handleToggleActive(ctx, telegramID, messageID, callbackKey)
			}
		}
	}

	screens := ems.getScreens()
	return ems.menuNav.HandleCallback(ctx, telegramID, messageID, callbackKey, screens)
}

// HandleMessage processes text messages (label/credential input)
func (ems *ExchangesMenuService) HandleMessage(ctx context.Context, userID interface{}, telegramID int64, messageID int, text string) error {
	session, err := ems.menuNav.GetSession(ctx, telegramID)
	if err != nil {
		return errors.Wrap(err, "no active exchanges session")
	}

	currentScreen := session.GetCurrentScreen()

	switch currentScreen {
	case "ent_lbl_add": // Enter label for new account
		return ems.handleEnterLabelAdd(ctx, session, text)
	case "ent_cred_add": // Enter credential for new account
		// Delete credential message after 3 seconds for security
		ems.menuNav.GetBot().DeleteMessageAfter(telegramID, messageID, 3*time.Second)
		return ems.handleEnterCredentialAdd(ctx, session, text)
	case "edit_lbl": // Edit existing account label
		return ems.handleEditLabel(ctx, session, text)
	case "ent_cred_edit": // Edit existing account credentials
		// Delete credential message after 3 seconds for security
		ems.menuNav.GetBot().DeleteMessageAfter(telegramID, messageID, 3*time.Second)
		return ems.handleEnterCredentialEdit(ctx, session, text)
	default:
		return fmt.Errorf("unexpected screen for text input: %s", currentScreen)
	}
}

// GetMenuType returns menu type identifier
func (ems *ExchangesMenuService) GetMenuType() string {
	return "exchanges"
}

// EndMenu ends the exchanges menu session
func (ems *ExchangesMenuService) EndMenu(ctx context.Context, telegramID int64) error {
	return ems.menuNav.EndMenu(ctx, telegramID)
}

// GetScreenIDs returns all screen IDs this handler owns (for MenuRegistry)
func (ems *ExchangesMenuService) GetScreenIDs() []string {
	return []string{
		"list", "detail", "sel_type", "sel_test", "ent_lbl_add", "ent_cred_add",
		"edit_lbl", "edit_cred", "ent_cred_edit",
	}
}

// getScreens returns all exchanges screens
func (ems *ExchangesMenuService) getScreens() map[string]*telegram.Screen {
	return map[string]*telegram.Screen{
		"list":          ems.buildListScreen(),
		"detail":        ems.buildDetailScreen(),
		"sel_type":      ems.buildSelectExchangeTypeScreen(),
		"sel_test":      ems.buildSelectTestnetScreen(),
		"ent_lbl_add":   ems.buildEnterLabelAddScreen(),
		"ent_cred_add":  ems.buildEnterCredentialAddScreen(),
		"edit_lbl":      ems.buildEditLabelScreen(),
		"edit_cred":     ems.buildEditCredentialsScreen(),
		"ent_cred_edit": ems.buildEnterCredentialEditScreen(),
	}
}

// buildListScreen builds exchange accounts list screen
func (ems *ExchangesMenuService) buildListScreen() *telegram.Screen {
	return &telegram.Screen{
		ID:       "list",
		Template: "exchange/manage_list",
		Data: func(ctx context.Context, session telegram.Session) (map[string]interface{}, error) {
			userIDStr, ok := session.GetString("user_id")
			if !ok {
				return nil, fmt.Errorf("user_id not found in session")
			}

			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				return nil, errors.Wrap(err, "invalid user_id")
			}

			accounts, err := ems.exchangeService.GetActiveUserAccounts(ctx, userID)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get active exchange accounts")
			}

			type AccountInfo struct {
				Label         string
				Exchange      string
				ExchangeEmoji string
				IsTestnet     bool
				IsActive      bool
			}

			var accountsData []AccountInfo
			for _, account := range accounts {
				accountsData = append(accountsData, AccountInfo{
					Label:         account.Label,
					Exchange:      exchange_account.GetExchangeDisplayName(account.Exchange),
					ExchangeEmoji: exchange_account.GetExchangeEmoji(account.Exchange),
					IsTestnet:     account.IsTestnet,
					IsActive:      account.IsActive,
				})
			}

			return map[string]interface{}{
				"Accounts":     accountsData,
				"HasAccounts":  len(accounts) > 0,
				"AccountCount": len(accounts),
			}, nil
		},
		Keyboard: func(ctx context.Context, nav *telegram.MenuNavigator, session telegram.Session) (telegram.InlineKeyboardMarkup, error) {
			userIDStr, _ := session.GetString("user_id")
			userID, _ := uuid.Parse(userIDStr)

			accounts, err := ems.exchangeService.GetActiveUserAccounts(ctx, userID)
			if err != nil {
				return telegram.InlineKeyboardMarkup{}, errors.Wrap(err, "failed to get accounts for keyboard")
			}

			var rows [][]telegram.InlineKeyboardButton
			for _, account := range accounts {
				emoji := exchange_account.GetExchangeEmoji(account.Exchange)
				displayName := exchange_account.GetExchangeDisplayName(account.Exchange)
				buttonText := fmt.Sprintf("%s %s - %s", emoji, account.Label, displayName)
				callbackData := nav.MakeCallback(session, "detail", exchangeKeys.AccountID, account.ID.String())
				rows = append(rows, telegram.NewInlineKeyboardRow(
					telegram.NewInlineKeyboardButtonData(buttonText, callbackData),
				))
			}

			// Add "Add New" button
			rows = append(rows, telegram.NewInlineKeyboardRow(
				telegram.NewInlineKeyboardButtonData("➕ Add New Exchange", nav.MakeCallback(session, "sel_type")),
			))

			return telegram.NewInlineKeyboardMarkup(rows...), nil
		},
	}
}

// buildDetailScreen builds exchange account detail screen
func (ems *ExchangesMenuService) buildDetailScreen() *telegram.Screen {
	return &telegram.Screen{
		ID:       "detail",
		Template: "exchange/manage_actions",
		Data: func(ctx context.Context, session telegram.Session) (map[string]interface{}, error) {
			accountIDStr, ok := session.GetString(exchangeKeys.AccountID)
			if !ok {
				return nil, fmt.Errorf("account_id not found in session")
			}

			accountID, err := uuid.Parse(accountIDStr)
			if err != nil {
				return nil, errors.Wrap(err, "invalid account_id")
			}

			account, err := ems.exchangeService.GetAccount(ctx, accountID)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get exchange account")
			}

			return map[string]interface{}{
				"Label":         account.Label,
				"Exchange":      exchange_account.GetExchangeDisplayName(account.Exchange),
				"ExchangeEmoji": exchange_account.GetExchangeEmoji(account.Exchange),
				"IsTestnet":     account.IsTestnet,
				"IsActive":      account.IsActive,
				"AccountID":     account.ID.String(),
			}, nil
		},
		Keyboard: func(ctx context.Context, nav *telegram.MenuNavigator, session telegram.Session) (telegram.InlineKeyboardMarkup, error) {
			accountIDStr, _ := session.GetString(exchangeKeys.AccountID)
			accountID, _ := uuid.Parse(accountIDStr)

			account, err := ems.exchangeService.GetAccount(ctx, accountID)
			if err != nil {
				return telegram.InlineKeyboardMarkup{}, errors.Wrap(err, "failed to get account for keyboard")
			}

			// Determine toggle button text based on current state
			toggleText := "🔴 Deactivate"
			if !account.IsActive {
				toggleText = "✅ Activate"
			}

			rows := [][]telegram.InlineKeyboardButton{
				telegram.NewInlineKeyboardRow(
					telegram.NewInlineKeyboardButtonData("✏️ Edit Label", nav.MakeCallback(session, "edit_lbl", exchangeKeys.AccountID, accountIDStr)),
				),
				telegram.NewInlineKeyboardRow(
					telegram.NewInlineKeyboardButtonData("🔑 Edit Credentials", nav.MakeCallback(session, "edit_cred", exchangeKeys.AccountID, accountIDStr)),
				),
				telegram.NewInlineKeyboardRow(
					telegram.NewInlineKeyboardButtonData(toggleText, nav.MakeCallback(session, "toggle", exchangeKeys.AccountID, accountIDStr)),
				),
				telegram.NewInlineKeyboardRow(
					telegram.NewInlineKeyboardButtonData("⬅️ Back", nav.MakeCallback(session, "list")),
				),
			}

			return telegram.NewInlineKeyboardMarkup(rows...), nil
		},
	}
}

// buildSelectExchangeTypeScreen builds exchange type selection screen
func (ems *ExchangesMenuService) buildSelectExchangeTypeScreen() *telegram.Screen {
	return ems.menuNav.BuildOptionScreen(telegram.OptionScreenConfig{
		ID:           "sel_type",
		Template:     "exchange/select",
		NextScreenID: "sel_test",
		ParamKey:     exchangeKeys.ExchangeType,
		Options: func(ctx context.Context, session telegram.Session) ([]telegram.MenuOption, error) {
			options := make([]telegram.MenuOption, len(exchangeTypeOptions))
			for i, opt := range exchangeTypeOptions {
				options[i] = opt
			}
			return options, nil
		},
		TemplateData: func(ctx context.Context, session telegram.Session) (map[string]interface{}, error) {
			return map[string]interface{}{
				"ExchangeTypes": exchangeTypeOptions,
			}, nil
		},
	})
}

// buildSelectTestnetScreen builds testnet selection screen
func (ems *ExchangesMenuService) buildSelectTestnetScreen() *telegram.Screen {
	return ems.menuNav.BuildOptionScreen(telegram.OptionScreenConfig{
		ID:           "sel_test",
		Template:     "exchange/select_testnet",
		NextScreenID: "ent_lbl_add",
		ParamKey:     exchangeKeys.IsTestnet,
		Options: func(ctx context.Context, session telegram.Session) ([]telegram.MenuOption, error) {
			options := make([]telegram.MenuOption, len(testnetOptions))
			for i, opt := range testnetOptions {
				options[i] = opt
			}
			return options, nil
		},
		TemplateData: func(ctx context.Context, session telegram.Session) (map[string]interface{}, error) {
			return map[string]interface{}{
				"TestnetOptions": testnetOptions,
			}, nil
		},
	})
}

// buildEnterLabelAddScreen builds label input screen for new account
func (ems *ExchangesMenuService) buildEnterLabelAddScreen() *telegram.Screen {
	return ems.menuNav.BuildTextInputScreen(
		"ent_lbl_add",
		"exchange/label_prompt",
		func(ctx context.Context, nav *telegram.MenuNavigator, params map[string]string) error {
			ems.log.Debugw("Entered label add screen", "params", params)
			return nil
		},
	)
}

// buildEnterCredentialAddScreen builds credential input screen for new account
func (ems *ExchangesMenuService) buildEnterCredentialAddScreen() *telegram.Screen {
	return &telegram.Screen{
		ID:       "ent_cred_add",
		Template: "exchange/enter_credential",
		Data: func(ctx context.Context, session telegram.Session) (map[string]interface{}, error) {
			exchangeTypeStr, _ := session.GetString(exchangeKeys.ExchangeType)
			exchangeType := exchange_account.ExchangeType(exchangeTypeStr)

			credIndexStr, _ := session.GetString(exchangeKeys.CredIndex)
			credIndex := 0
			if credIndexStr != "" {
				credIndex, _ = strconv.Atoi(credIndexStr)
			}

			fields := exchange_account.GetRequiredCredentials(exchangeType)
			if credIndex >= len(fields) {
				return nil, fmt.Errorf("invalid credential index")
			}

			field := fields[credIndex]
			return map[string]interface{}{
				"FieldName":    field.Name,
				"Placeholder":  field.Placeholder,
				"CurrentIndex": credIndex + 1,
				"TotalFields":  len(fields),
			}, nil
		},
		Keyboard: func(ctx context.Context, nav *telegram.MenuNavigator, session telegram.Session) (telegram.InlineKeyboardMarkup, error) {
			// No keyboard for text input
			return telegram.InlineKeyboardMarkup{}, nil
		},
	}
}

// buildEditLabelScreen builds label edit screen for existing account
func (ems *ExchangesMenuService) buildEditLabelScreen() *telegram.Screen {
	return ems.menuNav.BuildTextInputScreen(
		"edit_lbl",
		"exchange/label_prompt",
		func(ctx context.Context, nav *telegram.MenuNavigator, params map[string]string) error {
			ems.log.Debugw("Entered edit label screen", "params", params)
			return nil
		},
	)
}

// buildEditCredentialsScreen builds credentials edit start screen
func (ems *ExchangesMenuService) buildEditCredentialsScreen() *telegram.Screen {
	return &telegram.Screen{
		ID:       "edit_cred",
		Template: "exchange/label_prompt", // Reuse same template, will transition to input
		OnEnter: func(ctx context.Context, nav *telegram.MenuNavigator, params map[string]string) error {
			// Initialize credential index to 0
			session, _ := nav.GetSession(ctx, 0) // Will get from context in real implementation
			session.SetData(exchangeKeys.CredIndex, "0")
			return nil
		},
		Data: func(ctx context.Context, session telegram.Session) (map[string]interface{}, error) {
			return map[string]interface{}{
				"IsCredentialEdit": true,
			}, nil
		},
		Keyboard: func(ctx context.Context, nav *telegram.MenuNavigator, session telegram.Session) (telegram.InlineKeyboardMarkup, error) {
			// Immediately transition to credential input
			return telegram.InlineKeyboardMarkup{}, nil
		},
	}
}

// buildEnterCredentialEditScreen builds credential input screen for existing account
func (ems *ExchangesMenuService) buildEnterCredentialEditScreen() *telegram.Screen {
	return &telegram.Screen{
		ID:       "ent_cred_edit",
		Template: "exchange/enter_credential",
		Data: func(ctx context.Context, session telegram.Session) (map[string]interface{}, error) {
			accountIDStr, _ := session.GetString(exchangeKeys.AccountID)
			accountID, _ := uuid.Parse(accountIDStr)

			account, err := ems.exchangeService.GetAccount(ctx, accountID)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get account")
			}

			credIndexStr, _ := session.GetString(exchangeKeys.CredIndex)
			credIndex := 0
			if credIndexStr != "" {
				credIndex, _ = strconv.Atoi(credIndexStr)
			}

			fields := exchange_account.GetRequiredCredentials(account.Exchange)
			if credIndex >= len(fields) {
				return nil, fmt.Errorf("invalid credential index")
			}

			field := fields[credIndex]
			return map[string]interface{}{
				"FieldName":    field.Name,
				"Placeholder":  field.Placeholder,
				"CurrentIndex": credIndex + 1,
				"TotalFields":  len(fields),
			}, nil
		},
		Keyboard: func(ctx context.Context, nav *telegram.MenuNavigator, session telegram.Session) (telegram.InlineKeyboardMarkup, error) {
			// No keyboard for text input
			return telegram.InlineKeyboardMarkup{}, nil
		},
	}
}

// handleToggleActive handles activate/deactivate toggle
func (ems *ExchangesMenuService) handleToggleActive(ctx context.Context, telegramID int64, messageID int, callbackKey string) error {
	session, err := ems.menuNav.GetSession(ctx, telegramID)
	if err != nil {
		return errors.Wrap(err, "no active session")
	}

	// Retrieve callback parameters from session storage
	callbackParams, ok := session.GetCallbackData(callbackKey)
	if !ok {
		ems.log.Warnw("Toggle callback data not found in session",
			"callback_key", callbackKey,
			"telegram_id", telegramID,
		)
		return fmt.Errorf("callback data expired or invalid")
	}

	// Extract account_id from stored params
	accountIDInterface, ok := callbackParams[exchangeKeys.AccountID]
	if !ok {
		return fmt.Errorf("account_id not found in callback params")
	}
	accountIDStr, ok := accountIDInterface.(string)
	if !ok {
		return fmt.Errorf("invalid account_id type in callback params")
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return errors.Wrap(err, "invalid account_id in toggle")
	}

	// Get current account state
	account, err := ems.exchangeService.GetAccount(ctx, accountID)
	if err != nil {
		return errors.Wrap(err, "failed to get account for toggle")
	}

	// Toggle active state
	newState := !account.IsActive
	if err := ems.exchangeService.SetAccountActive(ctx, accountID, newState); err != nil {
		return errors.Wrap(err, "failed to toggle account state")
	}

	statusText := "activated"
	if !newState {
		statusText = "deactivated"
	}

	ems.log.Infow("Exchange account toggled",
		"account_id", accountID,
		"new_state", newState,
		"status", statusText,
	)

	// Re-render detail screen with updated state (DON'T close session)
	detailScreen := ems.buildDetailScreen()
	session.SetMessageID(messageID)

	// Get updated data
	screenData, err := detailScreen.Data(ctx, session)
	if err != nil {
		return errors.Wrap(err, "failed to get detail screen data")
	}

	// Render template
	text, err := ems.menuNav.RenderTemplate(detailScreen.Template, screenData)
	if err != nil {
		return errors.Wrap(err, "failed to render detail screen")
	}

	// Build keyboard with updated toggle button
	keyboard, err := detailScreen.Keyboard(ctx, ems.menuNav, session)
	if err != nil {
		return errors.Wrap(err, "failed to build detail keyboard")
	}

	// Update message
	return ems.menuNav.GetBot().EditMessage(telegramID, messageID, text, &keyboard)
}

// handleEnterLabelAdd handles label input for new account
func (ems *ExchangesMenuService) handleEnterLabelAdd(ctx context.Context, session telegram.Session, label string) error {
	label = strings.TrimSpace(label)
	if label == "" {
		return fmt.Errorf("❌ Label cannot be empty. Please enter a valid label.")
	}

	if len(label) > 50 {
		return fmt.Errorf("❌ Label is too long (max 50 characters). Please try again.")
	}

	// Save label to session
	session.SetData(exchangeKeys.Label, label)

	// Initialize credential index
	session.SetData(exchangeKeys.CredIndex, "0")

	// Navigate to credential input screen
	screens := ems.getScreens()
	credScreen := screens["ent_cred_add"]

	// Show credential input screen
	data, _ := credScreen.Data(ctx, session)
	text, _ := ems.menuNav.RenderTemplate(credScreen.Template, data)
	keyboard, _ := credScreen.Keyboard(ctx, ems.menuNav, session)

	telegramID := session.GetTelegramID()
	messageID := session.GetMessageID()

	// Update current screen in session
	session.PushScreen("ent_cred_add")
	session.SetCurrentScreen("ent_cred_add")

	if messageID > 0 {
		return ems.menuNav.GetBot().EditMessage(telegramID, messageID, text, &keyboard)
	}

	sentMessageID, err := ems.menuNav.GetBot().SendMessageWithOptions(telegramID, text, telegram.MessageOptions{
		Keyboard:  &keyboard,
		ParseMode: "Markdown",
	})
	if err == nil {
		session.SetMessageID(sentMessageID)
	}
	return err
}

// handleEnterCredentialAdd handles credential input for new account
func (ems *ExchangesMenuService) handleEnterCredentialAdd(ctx context.Context, session telegram.Session, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("❌ Credential cannot be empty. Please enter a valid value.")
	}

	exchangeTypeStr, _ := session.GetString(exchangeKeys.ExchangeType)
	exchangeType := exchange_account.ExchangeType(exchangeTypeStr)

	credIndexStr, _ := session.GetString(exchangeKeys.CredIndex)
	credIndex, _ := strconv.Atoi(credIndexStr)

	fields := exchange_account.GetRequiredCredentials(exchangeType)
	if credIndex >= len(fields) {
		return fmt.Errorf("invalid credential index")
	}

	field := fields[credIndex]

	// Save credential to session
	session.SetData(field.Key, value)

	// Check if we have more credentials to collect
	if credIndex+1 < len(fields) {
		// Move to next credential
		session.SetData(exchangeKeys.CredIndex, strconv.Itoa(credIndex+1))

		// Show next credential input screen
		screens := ems.getScreens()
		credScreen := screens["ent_cred_add"]

		data, _ := credScreen.Data(ctx, session)
		text, _ := ems.menuNav.RenderTemplate(credScreen.Template, data)
		keyboard, _ := credScreen.Keyboard(ctx, ems.menuNav, session)

		telegramID := session.GetTelegramID()
		messageID := session.GetMessageID()

		if messageID > 0 {
			return ems.menuNav.GetBot().EditMessage(telegramID, messageID, text, &keyboard)
		}
		return nil
	}

	// All credentials collected, create account
	return ems.createAccount(ctx, session)
}

// handleEditLabel handles label edit for existing account
func (ems *ExchangesMenuService) handleEditLabel(ctx context.Context, session telegram.Session, newLabel string) error {
	newLabel = strings.TrimSpace(newLabel)
	if newLabel == "" {
		return fmt.Errorf("❌ Label cannot be empty. Please enter a valid label.")
	}

	if len(newLabel) > 50 {
		return fmt.Errorf("❌ Label is too long (max 50 characters). Please try again.")
	}

	accountIDStr, _ := session.GetString(exchangeKeys.AccountID)
	accountID, _ := uuid.Parse(accountIDStr)

	// Update label
	if _, err := ems.exchangeService.UpdateAccount(ctx, exchange.UpdateAccountInput{
		AccountID: accountID,
		Label:     &newLabel,
	}); err != nil {
		ems.log.Errorw("Failed to update account label", "error", err, "account_id", accountID)
		return fmt.Errorf("❌ Failed to update label. Please try again.")
	}

	ems.log.Infow("Account label updated", "account_id", accountID, "new_label", newLabel)

	// Show success message and close session
	telegramID := session.GetTelegramID()
	_ = ems.menuNav.EndMenu(ctx, telegramID)

	successMsg, _ := ems.menuNav.RenderTemplate("exchange/success_updated_label", map[string]interface{}{
		"Label": newLabel,
	})
	return ems.menuNav.GetBot().SendMessage(telegramID, successMsg)
}

// handleEnterCredentialEdit handles credential input for existing account
func (ems *ExchangesMenuService) handleEnterCredentialEdit(ctx context.Context, session telegram.Session, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("❌ Credential cannot be empty. Please enter a valid value.")
	}

	accountIDStr, _ := session.GetString(exchangeKeys.AccountID)
	accountID, _ := uuid.Parse(accountIDStr)

	account, err := ems.exchangeService.GetAccount(ctx, accountID)
	if err != nil {
		return errors.Wrap(err, "failed to get account")
	}

	credIndexStr, _ := session.GetString(exchangeKeys.CredIndex)
	credIndex, _ := strconv.Atoi(credIndexStr)

	fields := exchange_account.GetRequiredCredentials(account.Exchange)
	if credIndex >= len(fields) {
		return fmt.Errorf("invalid credential index")
	}

	field := fields[credIndex]

	// Save credential to session
	session.SetData(field.Key, value)

	// Check if we have more credentials to collect
	if credIndex+1 < len(fields) {
		// Move to next credential
		session.SetData(exchangeKeys.CredIndex, strconv.Itoa(credIndex+1))

		// Show next credential input screen
		screens := ems.getScreens()
		credScreen := screens["ent_cred_edit"]

		data, _ := credScreen.Data(ctx, session)
		text, _ := ems.menuNav.RenderTemplate(credScreen.Template, data)
		keyboard, _ := credScreen.Keyboard(ctx, ems.menuNav, session)

		telegramID := session.GetTelegramID()
		messageID := session.GetMessageID()

		if messageID > 0 {
			return ems.menuNav.GetBot().EditMessage(telegramID, messageID, text, &keyboard)
		}
		return nil
	}

	// All credentials collected, update account
	return ems.updateAccountCredentials(ctx, session, accountID)
}

// createAccount creates a new exchange account
func (ems *ExchangesMenuService) createAccount(ctx context.Context, session telegram.Session) error {
	userIDStr, _ := session.GetString("user_id")
	userID, _ := uuid.Parse(userIDStr)

	exchangeTypeStr, _ := session.GetString(exchangeKeys.ExchangeType)
	exchangeType := exchange_account.ExchangeType(exchangeTypeStr)

	isTestnetStr, _ := session.GetString(exchangeKeys.IsTestnet)
	isTestnet := isTestnetStr == "true"

	label, _ := session.GetString(exchangeKeys.Label)
	apiKey, _ := session.GetString(exchangeKeys.APIKey)
	secret, _ := session.GetString(exchangeKeys.Secret)
	passphrase, _ := session.GetString(exchangeKeys.Passphrase)

	input := exchange.CreateAccountInput{
		UserID:     userID,
		Exchange:   exchangeType,
		APIKey:     apiKey,
		Secret:     secret,
		Label:      label,
		IsTestnet:  isTestnet,
		Passphrase: passphrase,
	}

	account, err := ems.exchangeService.CreateAccount(ctx, input)
	if err != nil {
		ems.log.Errorw("Failed to create exchange account", "error", err, "user_id", userID)

		telegramID := session.GetTelegramID()
		_ = ems.menuNav.EndMenu(ctx, telegramID)

		return ems.menuNav.GetBot().SendMessage(telegramID, fmt.Sprintf("❌ Failed to create exchange account: %v", err))
	}

	ems.log.Infow("Exchange account created", "account_id", account.ID, "exchange", account.Exchange, "user_id", userID)

	// Show success message and close session
	telegramID := session.GetTelegramID()
	_ = ems.menuNav.EndMenu(ctx, telegramID)

	successMsg, _ := ems.menuNav.RenderTemplate("exchange/connected", map[string]interface{}{
		"Label":    account.Label,
		"Exchange": exchange_account.GetExchangeDisplayName(account.Exchange),
	})
	return ems.menuNav.GetBot().SendMessage(telegramID, successMsg)
}

// updateAccountCredentials updates credentials for existing account
func (ems *ExchangesMenuService) updateAccountCredentials(ctx context.Context, session telegram.Session, accountID uuid.UUID) error {
	apiKey, _ := session.GetString(exchangeKeys.APIKey)
	secret, _ := session.GetString(exchangeKeys.Secret)
	passphrase, _ := session.GetString(exchangeKeys.Passphrase)

	input := exchange.UpdateAccountInput{
		AccountID:  accountID,
		APIKey:     &apiKey,
		Secret:     &secret,
		Passphrase: &passphrase,
	}

	account, err := ems.exchangeService.UpdateAccount(ctx, input)
	if err != nil {
		ems.log.Errorw("Failed to update account credentials", "error", err, "account_id", accountID)

		telegramID := session.GetTelegramID()
		_ = ems.menuNav.EndMenu(ctx, telegramID)

		return ems.menuNav.GetBot().SendMessage(telegramID, "❌ Failed to update credentials. Please try again.")
	}

	ems.log.Infow("Account credentials updated", "account_id", accountID)

	// Show success message and close session
	telegramID := session.GetTelegramID()
	_ = ems.menuNav.EndMenu(ctx, telegramID)

	successMsg, _ := ems.menuNav.RenderTemplate("exchange/success_updated_creds", map[string]interface{}{
		"Label": account.Label,
	})
	return ems.menuNav.GetBot().SendMessage(telegramID, successMsg)
}

// Verify ExchangesMenuService implements telegram.MenuHandler interface at compile time
var _ telegram.MenuHandler = (*ExchangesMenuService)(nil)
