package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	// Create new session
	session := &ExchangeSetupSession{
		TelegramID:        chatID,
		UserID:            userID,
		State:             ExchangeStateSelectExchange,
		StartedAt:         time.Now(),
		LastInteractionAt: time.Now(),
	}

	if err := es.saveSession(ctx, session); err != nil {
		return err
	}

	// Prompt exchange selection
	return es.promptExchangeSelection(chatID)
}

// HandleMessage processes a message during exchange setup flow
func (es *ExchangeSetupService) HandleMessage(ctx context.Context, userID uuid.UUID, telegramID int64, text string) error {
	// Get current session
	session, err := es.getSession(ctx, telegramID)
	if err != nil {
		return errors.Wrap(err, "no active exchange setup session")
	}

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

// handleExchangeSelection processes exchange selection
func (es *ExchangeSetupService) handleExchangeSelection(ctx context.Context, session *ExchangeSetupSession, text string) error {
	text = strings.ToLower(strings.TrimSpace(text))

	var exchangeType exchange_account.ExchangeType
	switch text {
	case "1", string(exchange_account.ExchangeBinance):
		exchangeType = exchange_account.ExchangeBinance
	case "2", string(exchange_account.ExchangeBybit):
		exchangeType = exchange_account.ExchangeBybit
	case "3", string(exchange_account.ExchangeOKX):
		exchangeType = exchange_account.ExchangeOKX
	default:
		msg, err := es.templates.Render("telegram/exchange_invalid_choice", nil)
		if err != nil {
			return errors.Wrap(err, "failed to render exchange_invalid_choice template")
		}
		return es.bot.SendMessage(session.TelegramID, msg)
	}

	session.SelectedExchange = string(exchangeType)
	session.State = ExchangeStateAwaitingAPIKey
	session.LastInteractionAt = time.Now()

	if err := es.saveSession(ctx, session); err != nil {
		return err
	}

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
	apiKey := strings.TrimSpace(text)

	if len(apiKey) < 16 {
		msg, err := es.templates.Render("telegram/exchange_api_key_invalid", nil)
		if err != nil {
			return errors.Wrap(err, "failed to render exchange_api_key_invalid template")
		}
		return es.bot.SendMessage(session.TelegramID, msg)
	}

	session.APIKey = apiKey
	session.State = ExchangeStateAwaitingSecret
	session.LastInteractionAt = time.Now()

	if err := es.saveSession(ctx, session); err != nil {
		return err
	}

	// Ask for secret using template
	msg, err := es.templates.Render("telegram/exchange_secret_prompt", nil)
	if err != nil {
		return errors.Wrap(err, "failed to render exchange_secret_prompt template")
	}

	return es.bot.SendMessage(session.TelegramID, msg)
}

// handleSecretInput processes API secret input
func (es *ExchangeSetupService) handleSecretInput(ctx context.Context, session *ExchangeSetupSession, text string, messageID int64) error {
	secret := strings.TrimSpace(text)

	if len(secret) < 16 {
		msg, err := es.templates.Render("telegram/exchange_secret_invalid", nil)
		if err != nil {
			return errors.Wrap(err, "failed to render exchange_secret_invalid template")
		}
		return es.bot.SendMessage(session.TelegramID, msg)
	}

	session.Secret = secret
	session.State = ExchangeStateAwaitingLabel
	session.LastInteractionAt = time.Now()

	if err := es.saveSession(ctx, session); err != nil {
		return err
	}

	// Ask for label using template
	msg, err := es.templates.Render("telegram/exchange_label_prompt", nil)
	if err != nil {
		return errors.Wrap(err, "failed to render exchange_label_prompt template")
	}

	return es.bot.SendMessage(session.TelegramID, msg)
}

// handleLabelInput processes label input and completes setup
func (es *ExchangeSetupService) handleLabelInput(ctx context.Context, session *ExchangeSetupSession, text string) error {
	label := strings.TrimSpace(text)
	if label == "" || strings.ToLower(label) == "skip" {
		label = fmt.Sprintf("%s Account", strings.Title(session.SelectedExchange))
	}

	session.Label = label
	session.State = ExchangeStateTesting
	session.LastInteractionAt = time.Now()

	if err := es.saveSession(ctx, session); err != nil {
		return err
	}

	// Notify testing using template
	msg, err := es.templates.Render("telegram/exchange_testing", nil)
	if err != nil {
		return errors.Wrap(err, "failed to render exchange_testing template")
	}
	es.bot.SendMessage(session.TelegramID, msg)

	// Test connection and save (business logic delegated to service)
	if err := es.testAndSaveExchange(ctx, session); err != nil {
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
	// TODO: Actually test connection by calling exchange API
	// For now, just validate format and save

	// Create exchange account entity
	account := &exchange_account.ExchangeAccount{
		ID:          uuid.New(),
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
	if err := account.SetAPIKey(session.APIKey, es.encryptor); err != nil {
		return errors.Wrap(err, "failed to encrypt API key")
	}

	if err := account.SetSecret(session.Secret, es.encryptor); err != nil {
		return errors.Wrap(err, "failed to encrypt secret")
	}

	// Save to repository
	if err := es.exchAcctRepo.Create(ctx, account); err != nil {
		return errors.Wrap(err, "failed to save exchange account")
	}

	es.log.Info("Exchange account created",
		"user_id", session.UserID,
		"exchange", session.SelectedExchange,
		"label", session.Label,
	)

	return nil
}

// promptExchangeSelection prompts user to select exchange
func (es *ExchangeSetupService) promptExchangeSelection(telegramID int64) error {
	// Build list of supported exchanges from constants
	supportedExchanges := []map[string]interface{}{
		{"Number": 1, "Name": strings.Title(string(exchange_account.ExchangeBinance)), "Value": string(exchange_account.ExchangeBinance)},
		{"Number": 2, "Name": strings.Title(string(exchange_account.ExchangeBybit)), "Value": string(exchange_account.ExchangeBybit)},
		{"Number": 3, "Name": strings.Title(string(exchange_account.ExchangeOKX)), "Value": string(exchange_account.ExchangeOKX)},
	}

	data := map[string]interface{}{
		"Exchanges": supportedExchanges,
	}

	msg, err := es.templates.Render("telegram/exchange_select", data)
	if err != nil {
		return errors.Wrap(err, "failed to render exchange_select template")
	}

	return es.bot.SendMessage(telegramID, msg)
}

// Redis session management

func (es *ExchangeSetupService) getSessionKey(telegramID int64) string {
	return fmt.Sprintf("exchange_setup:%d", telegramID)
}

func (es *ExchangeSetupService) getSession(ctx context.Context, telegramID int64) (*ExchangeSetupSession, error) {
	key := es.getSessionKey(telegramID)

	data, err := es.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, errors.Wrapf(errors.ErrNotFound, "no exchange setup session found")
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get exchange setup session")
	}

	var session ExchangeSetupSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal session")
	}

	return &session, nil
}

func (es *ExchangeSetupService) saveSession(ctx context.Context, session *ExchangeSetupSession) error {
	key := es.getSessionKey(session.TelegramID)

	data, err := json.Marshal(session)
	if err != nil {
		return errors.Wrap(err, "failed to marshal session")
	}

	// Set with 15 minute TTL (shorter than onboarding for security)
	if err := es.redis.Set(ctx, key, data, 15*time.Minute).Err(); err != nil {
		return errors.Wrap(err, "failed to save session")
	}

	return nil
}

func (es *ExchangeSetupService) deleteSession(ctx context.Context, telegramID int64) error {
	key := es.getSessionKey(telegramID)
	return es.redis.Del(ctx, key).Err()
}
