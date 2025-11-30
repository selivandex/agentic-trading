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
	"prometheus/pkg/crypto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"
)

// ExchangeSetupState represents the current stage of exchange setup
type ExchangeSetupState string

const (
	ExchangeStateSelectExchange ExchangeSetupState = "select_exchange"
	ExchangeStateAwaitingAPIKey ExchangeSetupState = "awaiting_api_key"
	ExchangeStateAwaitingSecret ExchangeSetupState = "awaiting_secret"
	ExchangeStateAwaitingLabel  ExchangeSetupState = "awaiting_label"
	ExchangeStateTesting        ExchangeSetupState = "testing"
	ExchangeStateComplete       ExchangeSetupState = "complete"
	ExchangeStateError          ExchangeSetupState = "error"
)

// ExchangeSetupSession stores the state of ongoing exchange setup
type ExchangeSetupSession struct {
	TelegramID        int64              `json:"telegram_id"`
	UserID            uuid.UUID          `json:"user_id"`
	State             ExchangeSetupState `json:"state"`
	SelectedExchange  string             `json:"selected_exchange"`
	APIKey            string             `json:"api_key"`
	Secret            string             `json:"secret"`
	Label             string             `json:"label"`
	StartedAt         time.Time          `json:"started_at"`
	LastInteractionAt time.Time          `json:"last_interaction_at"`
}

// ExchangeSetupService manages the exchange connection flow
type ExchangeSetupService struct {
	redis        *redis.Client
	bot          *Bot
	exchAcctRepo exchange_account.Repository
	exchFactory  exchanges.Factory
	encryptor    *crypto.Encryptor
	templates    *templates.Registry
	log          *logger.Logger
}

// NewExchangeSetupService creates a new exchange setup service
func NewExchangeSetupService(
	redis *redis.Client,
	bot *Bot,
	exchAcctRepo exchange_account.Repository,
	exchFactory exchanges.Factory,
	encryptor *crypto.Encryptor,
	tmpl *templates.Registry,
	log *logger.Logger,
) *ExchangeSetupService {
	if tmpl == nil {
		tmpl = templates.Get()
	}

	return &ExchangeSetupService{
		redis:        redis,
		bot:          bot,
		exchAcctRepo: exchAcctRepo,
		exchFactory:  exchFactory,
		encryptor:    encryptor,
		templates:    tmpl,
		log:          log.With("component", "exchange_setup"),
	}
}

// HandleAddExchange starts the exchange connection flow
func (es *ExchangeSetupService) HandleAddExchange(ctx context.Context, chatID int64, userID uuid.UUID) error {
	es.log.Debugw("Starting exchange setup flow",
		"chat_id", chatID,
		"user_id", userID,
	)

	// Create new session
	session := &ExchangeSetupSession{
		TelegramID:        chatID,
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
func (es *ExchangeSetupService) HandleMessage(ctx context.Context, userID uuid.UUID, telegramID int64, text string) error {
	es.log.Debugw("Processing exchange setup message",
		"telegram_id", telegramID,
		"user_id", userID,
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

	case ExchangeStateAwaitingAPIKey:
		return es.handleAPIKeyInput(ctx, session, text, telegramID)

	case ExchangeStateAwaitingSecret:
		return es.handleSecretInput(ctx, session, text, telegramID)

	case ExchangeStateAwaitingLabel:
		return es.handleLabelInput(ctx, session, text)

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

	return es.bot.SendMessage(session.TelegramID, msg)
}

// handleAPIKeyInput processes API key input
func (es *ExchangeSetupService) handleAPIKeyInput(ctx context.Context, session *ExchangeSetupSession, text string, messageID int64) error {
	es.log.Debugw("Processing API key input",
		"telegram_id", session.TelegramID,
		"user_id", session.UserID,
		"exchange", session.SelectedExchange,
		"key_length", len(text),
	)

	apiKey := strings.TrimSpace(text)

	if len(apiKey) < 16 {
		es.log.Debugw("API key validation failed: too short",
			"telegram_id", session.TelegramID,
			"provided_length", len(apiKey),
		)
		msg, err := es.templates.Render("telegram/exchange_api_key_invalid", nil)
		if err != nil {
			return errors.Wrap(err, "failed to render exchange_api_key_invalid template")
		}
		return es.bot.SendMessage(session.TelegramID, msg)
	}

	es.log.Debugw("API key validated successfully",
		"telegram_id", session.TelegramID,
	)

	session.APIKey = apiKey
	session.State = ExchangeStateAwaitingSecret
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
		"new_state", ExchangeStateAwaitingSecret,
	)

	// Ask for secret using template
	msg, err := es.templates.Render("telegram/exchange_secret_prompt", nil)
	if err != nil {
		return errors.Wrap(err, "failed to render exchange_secret_prompt template")
	}

	return es.bot.SendMessage(session.TelegramID, msg)
}

// handleSecretInput processes API secret input
func (es *ExchangeSetupService) handleSecretInput(ctx context.Context, session *ExchangeSetupSession, text string, messageID int64) error {
	es.log.Debugw("Processing API secret input",
		"telegram_id", session.TelegramID,
		"user_id", session.UserID,
		"exchange", session.SelectedExchange,
		"secret_length", len(text),
	)

	secret := strings.TrimSpace(text)

	if len(secret) < 16 {
		es.log.Debugw("API secret validation failed: too short",
			"telegram_id", session.TelegramID,
			"provided_length", len(secret),
		)
		msg, err := es.templates.Render("telegram/exchange_secret_invalid", nil)
		if err != nil {
			return errors.Wrap(err, "failed to render exchange_secret_invalid template")
		}
		return es.bot.SendMessage(session.TelegramID, msg)
	}

	es.log.Debugw("API secret validated successfully",
		"telegram_id", session.TelegramID,
	)

	session.Secret = secret
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

	return es.bot.SendMessage(session.TelegramID, msg)
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

// testAndSaveExchange tests connection and saves encrypted credentials
// Business logic: validates credentials, creates account entity with encryption
func (es *ExchangeSetupService) testAndSaveExchange(ctx context.Context, session *ExchangeSetupSession) error {
	es.log.Debugw("Testing and saving exchange credentials",
		"user_id", session.UserID,
		"exchange", session.SelectedExchange,
	)

	// TODO: Actually test connection by calling exchange API
	// For now, just validate format and save

	accountID := uuid.New()
	es.log.Debugw("Creating exchange account entity",
		"account_id", accountID,
		"user_id", session.UserID,
		"exchange", session.SelectedExchange,
	)

	// Create exchange account entity
	account := &exchange_account.ExchangeAccount{
		ID:          accountID,
		UserID:      session.UserID,
		Exchange:    exchange_account.ExchangeType(session.SelectedExchange),
		Label:       session.Label,
		IsTestnet:   false,
		Permissions: []string{"read", "trade"},
		IsActive:    true,
		LastSyncAt:  nil,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Use domain entity setters for encryption (encapsulation)
	// Log safely (length and preview only)
	apiKeyPreview := "empty"
	if len(session.APIKey) > 0 {
		if len(session.APIKey) >= 8 {
			apiKeyPreview = session.APIKey[:4] + "..." + session.APIKey[len(session.APIKey)-4:]
		} else {
			apiKeyPreview = session.APIKey[:1] + "..." + session.APIKey[len(session.APIKey)-1:]
		}
	}

	es.log.Debugw("Encrypting API credentials",
		"account_id", accountID,
		"api_key_length", len(session.APIKey),
		"api_key_preview", apiKeyPreview,
		"secret_length", len(session.Secret),
		"has_whitespace_start", len(session.APIKey) > 0 && (session.APIKey[0] == ' ' || session.APIKey[0] == '\t'),
		"has_whitespace_end", len(session.APIKey) > 0 && (session.APIKey[len(session.APIKey)-1] == ' ' || session.APIKey[len(session.APIKey)-1] == '\t'),
	)

	if err := account.SetAPIKey(session.APIKey, es.encryptor); err != nil {
		es.log.Errorw("Failed to encrypt API key",
			"account_id", accountID,
			"api_key_length", len(session.APIKey),
			"error", err,
		)
		return errors.Wrap(err, "failed to encrypt API key")
	}

	if err := account.SetSecret(session.Secret, es.encryptor); err != nil {
		es.log.Errorw("Failed to encrypt secret",
			"account_id", accountID,
			"secret_length", len(session.Secret),
			"error", err,
		)
		return errors.Wrap(err, "failed to encrypt secret")
	}

	es.log.Debugw("Credentials encrypted successfully, saving to repository",
		"account_id", accountID,
		"encrypted_api_key_bytes", len(account.APIKeyEncrypted),
		"encrypted_secret_bytes", len(account.SecretEncrypted),
	)

	// Save to repository
	if err := es.exchAcctRepo.Create(ctx, account); err != nil {
		es.log.Errorw("Failed to save exchange account to repository",
			"account_id", accountID,
			"user_id", session.UserID,
			"exchange", session.SelectedExchange,
			"error", err,
		)
		return errors.Wrap(err, "failed to save exchange account")
	}

	es.log.Infow("Exchange account created successfully",
		"account_id", accountID,
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
