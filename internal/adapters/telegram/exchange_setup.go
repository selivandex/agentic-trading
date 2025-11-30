package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/services/exchange"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"
)

// ExchangeSetupState represents the current stage of exchange setup/management
type ExchangeSetupState string

const (
	// Add exchange flow
	ExchangeStateSelectExchange ExchangeSetupState = "select_exchange"
	ExchangeStateAwaitingAPIKey ExchangeSetupState = "awaiting_api_key"
	ExchangeStateAwaitingSecret ExchangeSetupState = "awaiting_secret"
	ExchangeStateAwaitingLabel  ExchangeSetupState = "awaiting_label"
	ExchangeStateTesting        ExchangeSetupState = "testing"
	ExchangeStateComplete       ExchangeSetupState = "complete"
	ExchangeStateError          ExchangeSetupState = "error"

	// Update/manage exchange flow
	ExchangeStateSelectAccount ExchangeSetupState = "select_account"
	ExchangeStateSelectAction  ExchangeSetupState = "select_action"
	ExchangeStateUpdateLabel   ExchangeSetupState = "update_label"
	ExchangeStateUpdateAPIKey  ExchangeSetupState = "update_api_key"
	ExchangeStateUpdateSecret  ExchangeSetupState = "update_secret"
	ExchangeStateConfirmDelete ExchangeSetupState = "confirm_delete"
)

// ExchangeSetupSession stores the state of ongoing exchange setup/management
type ExchangeSetupSession struct {
	TelegramID        int64              `json:"telegram_id"`
	UserID            uuid.UUID          `json:"user_id"`
	State             ExchangeSetupState `json:"state"`
	SelectedExchange  string             `json:"selected_exchange"`
	SelectedAccountID uuid.UUID          `json:"selected_account_id,omitempty"` // For update flow
	APIKey            string             `json:"api_key"`
	Secret            string             `json:"secret"`
	Label             string             `json:"label"`
	IsTestnet         bool               `json:"is_testnet"`  // Use testnet mode
	IsUpdate          bool               `json:"is_update"`   // True if updating existing account
	LastMessageID     int                `json:"last_msg_id"` // Last bot message ID (for cleanup)
	StartedAt         time.Time          `json:"started_at"`
	LastInteractionAt time.Time          `json:"last_interaction_at"`
}

// ExchangeSetupService manages the exchange connection UI flow (Telegram-specific)
// All business logic is delegated to services/exchange.Service
type ExchangeSetupService struct {
	redis          *redis.Client
	bot            *Bot
	exchService    *exchange.Service // Business logic service
	exchFactory    exchanges.Factory // For testing credentials
	templates      *templates.Registry
	binanceTestnet bool // Use Binance testnet for user accounts
	bybitTestnet   bool // Use Bybit testnet for user accounts
	okxTestnet     bool // Use OKX testnet for user accounts
	log            *logger.Logger
}

// escapeMarkdown escapes special Markdown characters to prevent parsing errors
func escapeMarkdown(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}

// NewExchangeSetupService creates a new exchange setup service
func NewExchangeSetupService(
	redis *redis.Client,
	bot *Bot,
	exchService *exchange.Service, // Business logic service
	exchFactory exchanges.Factory,
	tmpl *templates.Registry,
	binanceTestnet bool, // Use Binance testnet
	bybitTestnet bool, // Use Bybit testnet
	okxTestnet bool, // Use OKX testnet
	log *logger.Logger,
) *ExchangeSetupService {
	if tmpl == nil {
		tmpl = templates.Get()
	}

	return &ExchangeSetupService{
		redis:          redis,
		bot:            bot,
		exchService:    exchService,
		exchFactory:    exchFactory,
		templates:      tmpl,
		binanceTestnet: binanceTestnet,
		bybitTestnet:   bybitTestnet,
		okxTestnet:     okxTestnet,
		log:            log.With("component", "exchange_setup"),
	}
}

// HandleAddExchange starts the exchange connection flow
func (es *ExchangeSetupService) HandleAddExchange(ctx context.Context, chatID int64, telegramID int64, userID uuid.UUID) error {
	es.log.Debugw("Starting exchange setup flow",
		"chat_id", chatID,
		"telegram_id", telegramID,
		"user_id", userID,
	)

	// Create new session
	session := &ExchangeSetupSession{
		TelegramID:        telegramID,
		UserID:            userID,
		State:             ExchangeStateSelectExchange,
		StartedAt:         time.Now(),
		LastInteractionAt: time.Now(),
	}

	if err := es.saveSession(ctx, session); err != nil {
		es.log.Errorw("Failed to save initial exchange setup session",
			"chat_id", chatID,
			"user_id", userID,
			"error", err,
		)
		return err
	}

	es.log.Debugw("Exchange setup session created, prompting exchange selection",
		"chat_id", chatID,
	)

	// Prompt exchange selection
	return es.promptExchangeSelection(chatID)
}

// HandleMessage processes a message during exchange setup flow
func (es *ExchangeSetupService) HandleMessage(ctx context.Context, userID uuid.UUID, telegramID int64, messageID int, text string) error {
	es.log.Debugw("Processing exchange setup message",
		"telegram_id", telegramID,
		"user_id", userID,
		"message_id", messageID,
		"text_length", len(text),
	)

	// Get current session
	session, err := es.getSession(ctx, telegramID)
	if err != nil {
		es.log.Errorw("No active exchange setup session found",
			"telegram_id", telegramID,
			"error", err,
		)
		return errors.Wrap(err, "no active exchange setup session")
	}

	es.log.Debugw("Exchange setup session retrieved",
		"telegram_id", telegramID,
		"current_state", session.State,
		"selected_exchange", session.SelectedExchange,
	)

	// Process based on state
	switch session.State {
	case ExchangeStateSelectExchange:
		return es.handleExchangeSelection(ctx, session, text)

	case ExchangeStateAwaitingAPIKey, ExchangeStateUpdateAPIKey:
		return es.handleAPIKeyInput(ctx, session, text, messageID)

	case ExchangeStateAwaitingSecret, ExchangeStateUpdateSecret:
		return es.handleSecretInput(ctx, session, text, messageID)

	case ExchangeStateAwaitingLabel:
		return es.handleLabelInput(ctx, session, text)

	case ExchangeStateUpdateLabel:
		return es.handleLabelUpdate(ctx, session, text)

	case ExchangeStateTesting:
		es.bot.SendMessage(telegramID, "⏳ Testing connection... Please wait.")
		return nil

	case ExchangeStateComplete:
		es.bot.SendMessage(telegramID, "✅ Exchange already connected!")
		return es.deleteSession(ctx, telegramID)

	default:
		return es.deleteSession(ctx, telegramID)
	}
}

// IsInSetup checks if a user is currently in exchange setup
func (es *ExchangeSetupService) IsInSetup(ctx context.Context, telegramID int64) (bool, error) {
	key := es.getSessionKey(telegramID)
	exists, err := es.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, errors.Wrap(err, "failed to check exchange setup session")
	}
	return exists > 0, nil
}

// handleExchangeSelection processes exchange selection from inline keyboard callback
func (es *ExchangeSetupService) handleExchangeSelection(ctx context.Context, session *ExchangeSetupSession, text string) error {
	es.log.Debugw("Processing exchange selection",
		"telegram_id", session.TelegramID,
		"user_id", session.UserID,
		"raw_text", text,
	)

	text = strings.ToLower(strings.TrimSpace(text))

	es.log.Debugw("Normalized exchange selection input",
		"telegram_id", session.TelegramID,
		"normalized_text", text,
	)

	var exchangeType exchange_account.ExchangeType

	// Handle both callback data (e.g., "binance") and legacy text input (e.g., "1")
	switch text {
	case "binance", "1":
		exchangeType = exchange_account.ExchangeBinance
	case "bybit", "2":
		exchangeType = exchange_account.ExchangeBybit
	case "okx", "3":
		exchangeType = exchange_account.ExchangeOKX
	default:
		es.log.Debugw("Invalid exchange selection",
			"telegram_id", session.TelegramID,
			"invalid_input", text,
		)
		es.bot.SendMessage(session.TelegramID, "❌ Invalid exchange selection. Please use the buttons to select an exchange.")
		// Re-prompt with keyboard
		return es.promptExchangeSelection(session.TelegramID)
	}

	es.log.Debugw("Exchange selected successfully",
		"telegram_id", session.TelegramID,
		"user_id", session.UserID,
		"exchange", exchangeType,
	)

	session.SelectedExchange = string(exchangeType)
	session.State = ExchangeStateAwaitingAPIKey
	session.LastInteractionAt = time.Now()

	// Set testnet flag based on exchange and config
	switch exchangeType {
	case exchange_account.ExchangeBinance:
		session.IsTestnet = es.binanceTestnet
	case exchange_account.ExchangeBybit:
		session.IsTestnet = es.bybitTestnet
	case exchange_account.ExchangeOKX:
		session.IsTestnet = es.okxTestnet
	default:
		session.IsTestnet = false
	}

	if err := es.saveSession(ctx, session); err != nil {
		es.log.Errorw("Failed to save exchange setup session",
			"telegram_id", session.TelegramID,
			"error", err,
		)
		return err
	}

	es.log.Debugw("Session state updated",
		"telegram_id", session.TelegramID,
		"new_state", ExchangeStateAwaitingAPIKey,
	)

	// Ask for API key using template
	data := map[string]interface{}{
		"Exchange":      strings.ToUpper(string(exchangeType)),
		"ExchangeTitle": strings.Title(string(exchangeType)),
	}

	msg, err := es.templates.Render("telegram/exchange_api_key_prompt", data)
	if err != nil {
		return errors.Wrap(err, "failed to render exchange_api_key_prompt template")
	}

	return es.sendMessageAndCleanup(ctx, session, msg)
}

// handleAPIKeyInput processes API key input (for both create and update flows)
func (es *ExchangeSetupService) handleAPIKeyInput(ctx context.Context, session *ExchangeSetupSession, text string, messageID int) error {
	isUpdate := session.State == ExchangeStateUpdateAPIKey

	es.log.Debugw("Processing API key input",
		"telegram_id", session.TelegramID,
		"user_id", session.UserID,
		"exchange", session.SelectedExchange,
		"key_length", len(text),
		"message_id", messageID,
		"is_update_flow", isUpdate,
	)

	apiKey := strings.TrimSpace(text)

	// Delete user's message with API key immediately for security
	es.bot.DeleteMessageAsync(session.TelegramID, messageID, "sensitive: API key")

	if len(apiKey) < 16 {
		es.log.Debugw("API key validation failed: too short",
			"telegram_id", session.TelegramID,
			"provided_length", len(apiKey),
		)
		msg, err := es.templates.Render("telegram/exchange_api_key_invalid", nil)
		if err != nil {
			return errors.Wrap(err, "failed to render exchange_api_key_invalid template")
		}
		return es.sendMessageAndCleanup(ctx, session, msg)
	}

	es.log.Debugw("API key validated successfully",
		"telegram_id", session.TelegramID,
	)

	session.APIKey = apiKey

	// Different next states for create vs update
	if isUpdate {
		session.State = ExchangeStateUpdateSecret
	} else {
		session.State = ExchangeStateAwaitingSecret
	}

	session.LastInteractionAt = time.Now()

	if err := es.saveSession(ctx, session); err != nil {
		es.log.Errorw("Failed to save session after API key input",
			"telegram_id", session.TelegramID,
			"error", err,
		)
		return err
	}

	es.log.Debugw("Session updated, requesting secret",
		"telegram_id", session.TelegramID,
		"new_state", session.State,
	)

	// Ask for secret using template
	msg, err := es.templates.Render("telegram/exchange_secret_prompt", nil)
	if err != nil {
		return errors.Wrap(err, "failed to render exchange_secret_prompt template")
	}

	return es.sendMessageAndCleanup(ctx, session, msg)
}

// handleSecretInput processes API secret input (for both create and update flows)
func (es *ExchangeSetupService) handleSecretInput(ctx context.Context, session *ExchangeSetupSession, text string, messageID int) error {
	isUpdate := session.State == ExchangeStateUpdateSecret

	es.log.Debugw("Processing API secret input",
		"telegram_id", session.TelegramID,
		"user_id", session.UserID,
		"exchange", session.SelectedExchange,
		"secret_length", len(text),
		"message_id", messageID,
		"is_update_flow", isUpdate,
	)

	secret := strings.TrimSpace(text)

	// Delete user's message with API secret immediately for security
	es.bot.DeleteMessageAsync(session.TelegramID, messageID, "sensitive: API secret")

	if len(secret) < 16 {
		es.log.Debugw("API secret validation failed: too short",
			"telegram_id", session.TelegramID,
			"provided_length", len(secret),
		)
		msg, err := es.templates.Render("telegram/exchange_secret_invalid", nil)
		if err != nil {
			return errors.Wrap(err, "failed to render exchange_secret_invalid template")
		}
		return es.sendMessageAndCleanup(ctx, session, msg)
	}

	es.log.Debugw("API secret validated successfully",
		"telegram_id", session.TelegramID,
	)

	session.Secret = secret

	// For update flow - save credentials immediately
	if isUpdate {
		return es.updateAccountCredentials(ctx, session)
	}

	// For create flow - continue to label input
	session.State = ExchangeStateAwaitingLabel
	session.LastInteractionAt = time.Now()

	if err := es.saveSession(ctx, session); err != nil {
		es.log.Errorw("Failed to save session after secret input",
			"telegram_id", session.TelegramID,
			"error", err,
		)
		return err
	}

	es.log.Debugw("Session updated, requesting label",
		"telegram_id", session.TelegramID,
		"new_state", ExchangeStateAwaitingLabel,
	)

	// Ask for label using template
	msg, err := es.templates.Render("telegram/exchange_label_prompt", nil)
	if err != nil {
		return errors.Wrap(err, "failed to render exchange_label_prompt template")
	}

	return es.sendMessageAndCleanup(ctx, session, msg)
}

// updateAccountCredentials updates API credentials via service
func (es *ExchangeSetupService) updateAccountCredentials(ctx context.Context, session *ExchangeSetupSession) error {
	es.log.Debugw("Updating account credentials",
		"account_id", session.SelectedAccountID,
		"telegram_id", session.TelegramID,
	)

	// Update via service
	apiKey := session.APIKey
	secret := session.Secret
	input := exchange.UpdateAccountInput{
		AccountID: session.SelectedAccountID,
		APIKey:    &apiKey,
		Secret:    &secret,
	}

	if _, err := es.exchService.UpdateAccount(ctx, input); err != nil {
		es.log.Errorw("Failed to update credentials via service",
			"account_id", session.SelectedAccountID,
			"error", err,
		)
		return es.bot.SendMessage(session.TelegramID, "❌ Failed to update credentials")
	}

	es.deleteSession(ctx, session.TelegramID)
	return es.bot.SendMessage(session.TelegramID, "✅ Credentials updated successfully!\n\nThe system will reconnect automatically.")
}

// handleLabelInput processes label input and completes setup
func (es *ExchangeSetupService) handleLabelInput(ctx context.Context, session *ExchangeSetupSession, text string) error {
	es.log.Debugw("Processing label input",
		"telegram_id", session.TelegramID,
		"user_id", session.UserID,
		"exchange", session.SelectedExchange,
		"label_input", text,
	)

	label := strings.TrimSpace(text)
	if label == "" || strings.ToLower(label) == "skip" {
		label = fmt.Sprintf("%s Account", strings.Title(session.SelectedExchange))
		es.log.Debugw("Using default label",
			"telegram_id", session.TelegramID,
			"default_label", label,
		)
	}

	session.Label = label
	session.State = ExchangeStateTesting
	session.LastInteractionAt = time.Now()

	if err := es.saveSession(ctx, session); err != nil {
		es.log.Errorw("Failed to save session before testing",
			"telegram_id", session.TelegramID,
			"error", err,
		)
		return err
	}

	es.log.Debugw("Starting exchange connection test",
		"telegram_id", session.TelegramID,
		"exchange", session.SelectedExchange,
	)

	// Notify testing using template
	msg, err := es.templates.Render("telegram/exchange_testing", nil)
	if err != nil {
		return errors.Wrap(err, "failed to render exchange_testing template")
	}
	es.bot.SendMessage(session.TelegramID, msg)

	// Test connection and save (business logic delegated to service)
	if err := es.testAndSaveExchange(ctx, session); err != nil {
		es.log.Errorw("Exchange connection test failed",
			"telegram_id", session.TelegramID,
			"user_id", session.UserID,
			"exchange", session.SelectedExchange,
			"error", err,
		)

		session.State = ExchangeStateError
		es.saveSession(ctx, session)

		// Render error template
		errorData := map[string]interface{}{
			"Error": err.Error(),
		}
		errorMsg, renderErr := es.templates.Render("telegram/exchange_connection_failed", errorData)
		if renderErr != nil {
			return errors.Wrap(renderErr, "failed to render exchange_connection_failed template")
		}

		es.bot.SendMessage(session.TelegramID, errorMsg)
		es.deleteSession(ctx, session.TelegramID)
		return err
	}

	es.log.Infow("Exchange connected successfully",
		"telegram_id", session.TelegramID,
		"user_id", session.UserID,
		"exchange", session.SelectedExchange,
		"label", session.Label,
	)

	// Success - render success template
	successData := map[string]interface{}{
		"Exchange": strings.ToUpper(session.SelectedExchange),
		"Label":    session.Label,
	}

	successMsg, err := es.templates.Render("telegram/exchange_connected", successData)
	if err != nil {
		return errors.Wrap(err, "failed to render exchange_connected template")
	}

	es.bot.SendMessage(session.TelegramID, successMsg)
	return es.deleteSession(ctx, session.TelegramID)
}

// testAndSaveExchange delegates account creation to business service
func (es *ExchangeSetupService) testAndSaveExchange(ctx context.Context, session *ExchangeSetupSession) error {
	es.log.Debugw("Creating exchange account via service",
		"user_id", session.UserID,
		"exchange", session.SelectedExchange,
	)

	// TODO: Actually test connection by calling exchange API before saving

	// Delegate to business service
	input := exchange.CreateAccountInput{
		UserID:    session.UserID,
		Exchange:  exchange_account.ExchangeType(session.SelectedExchange),
		APIKey:    session.APIKey,
		Secret:    session.Secret,
		Label:     session.Label,
		IsTestnet: session.IsTestnet, // Use testnet flag from session (set from config)
	}

	account, err := es.exchService.CreateAccount(ctx, input)
	if err != nil {
		es.log.Errorw("Failed to create exchange account via service",
			"user_id", session.UserID,
			"exchange", session.SelectedExchange,
			"error", err,
		)
		return err
	}

	es.log.Infow("✅ Exchange account created successfully",
		"account_id", account.ID,
		"user_id", session.UserID,
		"exchange", session.SelectedExchange,
		"label", session.Label,
	)

	return nil
}

// promptExchangeSelection prompts user to select exchange with inline keyboard
func (es *ExchangeSetupService) promptExchangeSelection(telegramID int64) error {
	es.log.Debugw("Prompting exchange selection with inline keyboard",
		"telegram_id", telegramID,
	)

	msg := "🔗 *Connect Exchange*\n\nSelect your exchange:"

	// Create inline keyboard with exchange options
	// For now, only Binance is available
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Binance", "exchange:binance"),
		),
		// TODO: Enable other exchanges when ready
		// tgbotapi.NewInlineKeyboardRow(
		// 	tgbotapi.NewInlineKeyboardButtonData("📈 Bybit", "exchange:bybit"),
		// ),
		// tgbotapi.NewInlineKeyboardRow(
		// 	tgbotapi.NewInlineKeyboardButtonData("📉 OKX", "exchange:okx"),
		// ),
	)

	if err := es.bot.SendMessageWithKeyboard(telegramID, msg, keyboard); err != nil {
		es.log.Errorw("Failed to send exchange selection keyboard",
			"telegram_id", telegramID,
			"error", err,
		)
		return err
	}

	es.log.Debugw("Exchange selection keyboard sent successfully",
		"telegram_id", telegramID,
	)
	return nil
}

// sendMessageAndCleanup sends a new message and deletes the previous one from session
func (es *ExchangeSetupService) sendMessageAndCleanup(ctx context.Context, session *ExchangeSetupSession, text string) error {
	// Delete previous message if exists
	if session.LastMessageID > 0 {
		es.bot.DeleteMessageAsync(session.TelegramID, session.LastMessageID, "UI cleanup")
	}

	// Send new message
	msg := tgbotapi.NewMessage(session.TelegramID, text)
	msg.ParseMode = "Markdown"

	sentMsg, err := es.bot.GetAPI().Send(msg)
	if err != nil {
		return errors.Wrap(err, "failed to send message")
	}

	// Save new message ID
	session.LastMessageID = sentMsg.MessageID
	return es.saveSession(ctx, session)
}

// sendMessageWithKeyboardAndCleanup sends a message with keyboard and deletes the previous one
func (es *ExchangeSetupService) sendMessageWithKeyboardAndCleanup(ctx context.Context, session *ExchangeSetupSession, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	// Delete previous message if exists
	if session.LastMessageID > 0 {
		es.bot.DeleteMessageAsync(session.TelegramID, session.LastMessageID, "UI cleanup")
	}

	// Send new message with keyboard
	msg := tgbotapi.NewMessage(session.TelegramID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := es.bot.GetAPI().Send(msg)
	if err != nil {
		return errors.Wrap(err, "failed to send message with keyboard")
	}

	// Save new message ID
	session.LastMessageID = sentMsg.MessageID
	return es.saveSession(ctx, session)
}

// Redis session management

func (es *ExchangeSetupService) getSessionKey(telegramID int64) string {
	return fmt.Sprintf("exchange_setup:%d", telegramID)
}

func (es *ExchangeSetupService) getSession(ctx context.Context, telegramID int64) (*ExchangeSetupSession, error) {
	key := es.getSessionKey(telegramID)

	es.log.Debugw("Retrieving exchange setup session from Redis",
		"telegram_id", telegramID,
		"redis_key", key,
	)

	data, err := es.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		es.log.Debugw("No exchange setup session found in Redis",
			"telegram_id", telegramID,
		)
		return nil, errors.Wrapf(errors.ErrNotFound, "no exchange setup session found")
	}
	if err != nil {
		es.log.Errorw("Failed to get exchange setup session from Redis",
			"telegram_id", telegramID,
			"error", err,
		)
		return nil, errors.Wrap(err, "failed to get exchange setup session")
	}

	var session ExchangeSetupSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		es.log.Errorw("Failed to unmarshal exchange setup session",
			"telegram_id", telegramID,
			"error", err,
		)
		return nil, errors.Wrap(err, "failed to unmarshal session")
	}

	es.log.Debugw("Exchange setup session retrieved successfully",
		"telegram_id", telegramID,
		"state", session.State,
		"exchange", session.SelectedExchange,
	)

	return &session, nil
}

func (es *ExchangeSetupService) saveSession(ctx context.Context, session *ExchangeSetupSession) error {
	key := es.getSessionKey(session.TelegramID)

	es.log.Debugw("Saving exchange setup session to Redis",
		"telegram_id", session.TelegramID,
		"user_id", session.UserID,
		"state", session.State,
		"exchange", session.SelectedExchange,
		"redis_key", key,
	)

	data, err := json.Marshal(session)
	if err != nil {
		es.log.Errorw("Failed to marshal exchange setup session",
			"telegram_id", session.TelegramID,
			"error", err,
		)
		return errors.Wrap(err, "failed to marshal session")
	}

	// Set with 15 minute TTL (shorter than onboarding for security)
	if err := es.redis.Set(ctx, key, data, 15*time.Minute).Err(); err != nil {
		es.log.Errorw("Failed to save exchange setup session to Redis",
			"telegram_id", session.TelegramID,
			"error", err,
		)
		return errors.Wrap(err, "failed to save session")
	}

	es.log.Debugw("Exchange setup session saved successfully",
		"telegram_id", session.TelegramID,
		"ttl", "15m",
	)

	return nil
}

func (es *ExchangeSetupService) deleteSession(ctx context.Context, telegramID int64) error {
	key := es.getSessionKey(telegramID)

	es.log.Debugw("Deleting exchange setup session from Redis",
		"telegram_id", telegramID,
		"redis_key", key,
	)

	// Get session to clean up last message
	session, err := es.getSession(ctx, telegramID)
	if err == nil && session.LastMessageID > 0 {
		// Delete last UI message
		es.bot.DeleteMessageAsync(telegramID, session.LastMessageID, "session cleanup")
	}

	if err := es.redis.Del(ctx, key).Err(); err != nil {
		es.log.Errorw("Failed to delete exchange setup session from Redis",
			"telegram_id", telegramID,
			"error", err,
		)
		return err
	}

	es.log.Debugw("Exchange setup session deleted successfully",
		"telegram_id", telegramID,
	)

	return nil
}

// ========================================================================
// Exchange Account Management (Update/Delete existing accounts)
// ========================================================================

// HandleUpdateExchange starts the exchange update/management flow
func (es *ExchangeSetupService) HandleUpdateExchange(ctx context.Context, chatID int64, telegramID int64, userID uuid.UUID) error {
	es.log.Infow("Starting exchange management flow",
		"chat_id", chatID,
		"telegram_id", telegramID,
		"user_id", userID,
	)

	// Get user's exchange accounts via service
	accounts, err := es.exchService.GetUserAccounts(ctx, userID)
	if err != nil {
		es.log.Errorw("Failed to get user exchange accounts",
			"user_id", userID,
			"error", err,
		)
		return es.bot.SendMessage(chatID, "❌ Failed to load your exchange accounts")
	}

	if len(accounts) == 0 {
		return es.bot.SendMessage(chatID, "You don't have any connected exchanges.\n\nUse /add_exchange to connect one.")
	}

	// Create new session for management
	session := &ExchangeSetupSession{
		TelegramID:        telegramID,
		UserID:            userID,
		State:             ExchangeStateSelectAccount,
		IsUpdate:          true,
		StartedAt:         time.Now(),
		LastInteractionAt: time.Now(),
	}

	if err := es.saveSession(ctx, session); err != nil {
		return err
	}

	// Show exchange selection keyboard
	return es.showAccountList(ctx, session, accounts)
}

// HandleCallback processes callback queries from inline keyboards (for management flow)
func (es *ExchangeSetupService) HandleCallback(ctx context.Context, userID uuid.UUID, telegramID int64, messageID int, data string) error {
	es.log.Debugw("Processing exchange callback",
		"telegram_id", telegramID,
		"callback_data", data,
		"message_id", messageID,
	)

	session, err := es.getSession(ctx, telegramID)
	if err != nil {
		return errors.Wrap(err, "no active exchange session")
	}

	// Update last message ID to the callback message (will be deleted when showing next screen)
	session.LastMessageID = messageID

	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return fmt.Errorf("invalid callback data format")
	}

	action := parts[1] // parts[0] is "exch"

	switch session.State {
	case ExchangeStateSelectAccount:
		// User selected an exchange account to manage
		if action == "cancel" {
			es.deleteSession(ctx, telegramID)
			return es.bot.SendMessage(telegramID, "❌ Cancelled")
		}
		accountID, err := uuid.Parse(action)
		if err != nil {
			return errors.Wrap(err, "invalid account ID")
		}
		return es.handleAccountSelected(ctx, session, accountID)

	case ExchangeStateSelectAction:
		return es.handleActionSelected(ctx, session, action)

	case ExchangeStateConfirmDelete:
		return es.handleDeleteConfirmation(ctx, session, action)

	default:
		return fmt.Errorf("unexpected state for callback: %s", session.State)
	}
}

// showAccountList displays list of user's exchange accounts with inline keyboard
func (es *ExchangeSetupService) showAccountList(ctx context.Context, session *ExchangeSetupSession, accounts []*exchange_account.ExchangeAccount) error {
	// Prepare data for template
	type AccountInfo struct {
		StatusEmoji string
		Exchange    string
		Label       string
	}

	accountsData := make([]AccountInfo, len(accounts))
	for i, account := range accounts {
		statusEmoji := "✅"
		if !account.IsActive {
			statusEmoji = "❌"
		}
		accountsData[i] = AccountInfo{
			StatusEmoji: statusEmoji,
			Exchange:    strings.Title(string(account.Exchange)),
			Label:       account.Label,
		}
	}

	data := map[string]interface{}{
		"Accounts": accountsData,
	}

	msg, err := es.templates.Render("telegram/exchange_manage_list", data)
	if err != nil {
		return errors.Wrap(err, "failed to render exchange_manage_list template")
	}

	// Create keyboard with exchange accounts
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, account := range accounts {
		statusEmoji := "✅"
		if !account.IsActive {
			statusEmoji = "❌"
		}

		// Don't need to escape in button text - Telegram handles it
		buttonText := fmt.Sprintf("%s %s - %s", statusEmoji, strings.Title(string(account.Exchange)), account.Label)
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("exch:%s", account.ID.String())),
		)
		rows = append(rows, row)
	}

	// Add cancel button
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", "exch:cancel"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return es.sendMessageWithKeyboardAndCleanup(ctx, session, msg, keyboard)
}

// handleAccountSelected processes exchange account selection
func (es *ExchangeSetupService) handleAccountSelected(ctx context.Context, session *ExchangeSetupSession, accountID uuid.UUID) error {
	es.log.Infow("Exchange account selected for management",
		"telegram_id", session.TelegramID,
		"account_id", accountID,
	)

	// Load account details via service
	account, err := es.exchService.GetAccount(ctx, accountID)
	if err != nil {
		es.log.Errorw("Failed to load exchange account",
			"account_id", accountID,
			"error", err,
		)
		return es.bot.SendMessage(session.TelegramID, "❌ Failed to load exchange account")
	}

	// Update session
	session.SelectedAccountID = accountID
	session.SelectedExchange = string(account.Exchange)
	session.Label = account.Label
	session.IsTestnet = account.IsTestnet // Preserve testnet flag from existing account
	session.State = ExchangeStateSelectAction
	session.LastInteractionAt = time.Now()

	if err := es.saveSession(ctx, session); err != nil {
		return err
	}

	// Show action menu
	return es.showActionMenu(ctx, session, account)
}

// showActionMenu displays available actions for selected exchange
func (es *ExchangeSetupService) showActionMenu(ctx context.Context, session *ExchangeSetupSession, account *exchange_account.ExchangeAccount) error {
	statusText := "✅ Active"
	toggleAction := "deactivate"
	if !account.IsActive {
		statusText = "❌ Inactive"
		toggleAction = "activate"
	}

	data := map[string]interface{}{
		"Exchange":   strings.Title(string(account.Exchange)),
		"Label":      account.Label,
		"StatusText": statusText,
		"IsTestnet":  account.IsTestnet,
	}

	msg, err := es.templates.Render("telegram/exchange_manage_actions", data)
	if err != nil {
		return errors.Wrap(err, "failed to render exchange_manage_actions template")
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Update Label", "exch:update_label"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔑 Update Credentials", "exch:update_creds"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("⚙️ %s", strings.Title(toggleAction)), fmt.Sprintf("exch:%s", toggleAction)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Delete", "exch:delete"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Back", "exch:back"),
		),
	)

	return es.sendMessageWithKeyboardAndCleanup(ctx, session, msg, keyboard)
}

// handleActionSelected processes selected action
func (es *ExchangeSetupService) handleActionSelected(ctx context.Context, session *ExchangeSetupSession, action string) error {
	es.log.Infow("Management action selected",
		"telegram_id", session.TelegramID,
		"account_id", session.SelectedAccountID,
		"action", action,
	)

	switch action {
	case "update_label":
		session.State = ExchangeStateUpdateLabel
		session.LastInteractionAt = time.Now()
		return es.sendMessageAndCleanup(ctx, session, "✏️ Please enter a new label:")

	case "update_creds":
		session.State = ExchangeStateUpdateAPIKey
		session.LastInteractionAt = time.Now()
		return es.sendMessageAndCleanup(ctx, session, "🔑 Please enter your new API Key:")

	case "activate", "deactivate":
		return es.toggleAccountActive(ctx, session, action == "activate")

	case "delete":
		session.State = ExchangeStateConfirmDelete
		session.LastInteractionAt = time.Now()
		es.saveSession(ctx, session)
		return es.showDeleteConfirmation(ctx, session)

	case "back":
		es.deleteSession(ctx, session.TelegramID)
		return es.HandleUpdateExchange(ctx, session.TelegramID, session.TelegramID, session.UserID)

	case "cancel":
		es.deleteSession(ctx, session.TelegramID)
		return es.bot.SendMessage(session.TelegramID, "❌ Cancelled")

	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

// toggleAccountActive activates or deactivates exchange account via service
func (es *ExchangeSetupService) toggleAccountActive(ctx context.Context, session *ExchangeSetupSession, activate bool) error {
	// Delegate to service
	if err := es.exchService.SetAccountActive(ctx, session.SelectedAccountID, activate); err != nil {
		es.log.Errorw("Failed to toggle account status via service",
			"account_id", session.SelectedAccountID,
			"activate", activate,
			"error", err,
		)
		return es.bot.SendMessage(session.TelegramID, "❌ Failed to update status")
	}

	status := "deactivated"
	if activate {
		status = "activated"
	}

	es.deleteSession(ctx, session.TelegramID)
	return es.bot.SendMessage(session.TelegramID, fmt.Sprintf("✅ Account %s!", status))
}

// showDeleteConfirmation shows delete confirmation dialog
func (es *ExchangeSetupService) showDeleteConfirmation(ctx context.Context, session *ExchangeSetupSession) error {
	msg, err := es.templates.Render("telegram/exchange_manage_delete_confirm", nil)
	if err != nil {
		return errors.Wrap(err, "failed to render exchange_manage_delete_confirm template")
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Yes", "exch:confirm_delete"),
			tgbotapi.NewInlineKeyboardButtonData("❌ No", "exch:cancel_delete"),
		),
	)

	return es.sendMessageWithKeyboardAndCleanup(ctx, session, msg, keyboard)
}

// handleDeleteConfirmation processes delete confirmation
func (es *ExchangeSetupService) handleDeleteConfirmation(ctx context.Context, session *ExchangeSetupSession, action string) error {
	if action == "cancel_delete" {
		es.deleteSession(ctx, session.TelegramID)
		return es.bot.SendMessage(session.TelegramID, "❌ Deletion cancelled")
	}

	if action != "confirm_delete" {
		return fmt.Errorf("unexpected delete action: %s", action)
	}

	// Delete account via service
	if err := es.exchService.DeleteAccount(ctx, session.SelectedAccountID); err != nil {
		es.log.Errorw("Failed to delete exchange account via service",
			"account_id", session.SelectedAccountID,
			"error", err,
		)
		return es.bot.SendMessage(session.TelegramID, "❌ Failed to delete account")
	}

	es.deleteSession(ctx, session.TelegramID)
	return es.bot.SendMessage(session.TelegramID, "✅ Account deleted!")
}

// handleLabelUpdate updates exchange account label
func (es *ExchangeSetupService) handleLabelUpdate(ctx context.Context, session *ExchangeSetupSession, text string) error {
	newLabel := strings.TrimSpace(text)
	if newLabel == "" {
		return es.sendMessageAndCleanup(ctx, session, "❌ Label cannot be empty. Please try again:")
	}

	es.log.Debugw("Updating account label",
		"account_id", session.SelectedAccountID,
		"new_label", newLabel,
	)

	// Load current account to show old label
	account, err := es.exchService.GetAccount(ctx, session.SelectedAccountID)
	if err != nil {
		return err
	}
	oldLabel := account.Label

	// Update via service
	label := newLabel
	input := exchange.UpdateAccountInput{
		AccountID: session.SelectedAccountID,
		Label:     &label,
	}

	if _, err := es.exchService.UpdateAccount(ctx, input); err != nil {
		es.log.Errorw("Failed to update account label via service",
			"account_id", session.SelectedAccountID,
			"error", err,
		)
		return es.bot.SendMessage(session.TelegramID, "❌ Failed to update label")
	}

	es.deleteSession(ctx, session.TelegramID)
	return es.bot.SendMessage(session.TelegramID, fmt.Sprintf("✅ Label updated: *%s* → *%s*", oldLabel, newLabel))
}
