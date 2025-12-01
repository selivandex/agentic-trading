package tgbotapi

import (
	"context"
	"net/http"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/time/rate"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/telegram"
)

// Bot represents a Telegram bot that implements telegram.Bot interface
type Bot struct {
	api         *tgbotapi.BotAPI
	updates     tgbotapi.UpdatesChannel
	log         *logger.Logger
	mu          sync.RWMutex
	running     bool
	webhookMode bool
	msgHandler  func(telegram.Update) // Handler works with abstracted Update
	rateLimiter *rate.Limiter
	asyncQueue  *telegram.AsyncMessageQueue // For async message sending
}

// Config contains Telegram bot configuration
type Config struct {
	Token          string
	Debug          bool
	Timeout        int  // Update timeout in seconds
	BufferSize     int  // Update channel buffer size
	WebhookMode    bool // If true, don't start polling (use webhook instead)
	HTTPTimeout    time.Duration
	RateLimitBurst int // Rate limiter burst (default: 30)
	RateLimitRate  int // Rate limiter per second (default: 20)
}

// NewBot creates a new Telegram bot instance that implements telegram.Bot interface
func NewBot(cfg Config, log *logger.Logger) (*Bot, error) {
	if cfg.Token == "" {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "telegram bot token is required")
	}

	// Set defaults
	if cfg.Timeout == 0 {
		cfg.Timeout = 60
	}
	if cfg.BufferSize == 0 {
		cfg.BufferSize = 100
	}
	if cfg.HTTPTimeout == 0 {
		cfg.HTTPTimeout = 30 * time.Second
	}
	if cfg.RateLimitBurst == 0 {
		cfg.RateLimitBurst = 30
	}
	if cfg.RateLimitRate == 0 {
		cfg.RateLimitRate = 20
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: cfg.HTTPTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// Create bot API with custom client
	api, err := tgbotapi.NewBotAPIWithClient(cfg.Token, tgbotapi.APIEndpoint, httpClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create telegram bot")
	}

	api.Debug = cfg.Debug

	log.Infof("Authorized on account %s", api.Self.UserName)

	// Create rate limiter
	rateLimiter := rate.NewLimiter(rate.Limit(cfg.RateLimitRate), cfg.RateLimitBurst)

	bot := &Bot{
		api:         api,
		webhookMode: cfg.WebhookMode,
		log:         log.With("component", "telegram_bot"),
		rateLimiter: rateLimiter,
	}

	// Create async message queue
	bot.asyncQueue = telegram.NewAsyncMessageQueue(bot, 5, 50*time.Millisecond, log)

	return bot, nil
}

// Start begins polling for updates (or just blocks if webhook mode)
func (b *Bot) Start(ctx context.Context) error {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return errors.New("bot is already running")
	}
	b.running = true
	b.mu.Unlock()

	// Start async message queue
	b.asyncQueue.Start()
	defer b.asyncQueue.Stop()

	if b.webhookMode {
		b.log.Infow("Bot running in webhook mode, not starting polling")
		<-ctx.Done()
		return ctx.Err()
	}

	// Setup polling
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)
	b.updates = updates

	b.log.Infow("Starting to poll for updates")

	for {
		select {
		case <-ctx.Done():
			b.log.Infow("Stopping bot due to context cancellation")
			b.Stop()
			return ctx.Err()

		case tgUpdate := <-updates:
			if b.msgHandler != nil {
				// Convert tgbotapi.Update to telegram.Update (abstraction layer)
				update := convertUpdate(tgUpdate)
				go b.msgHandler(update)
			}
		}
	}
}

// Stop stops the bot
func (b *Bot) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running {
		return
	}

	b.api.StopReceivingUpdates()
	b.running = false
	b.log.Infow("Bot stopped")
}

// SetHandler sets the message handler (uses abstracted Update type)
func (b *Bot) SetHandler(handler func(telegram.Update)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.msgHandler = handler
}

// =============================================================================
// telegram.Bot Interface Implementation
// =============================================================================

// SendMessage sends a text message (blocking)
func (b *Bot) SendMessage(chatID int64, text string) error {
	if err := b.rateLimiter.Wait(context.Background()); err != nil {
		return errors.Wrap(err, "rate limiter error")
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	_, err := b.api.Send(msg)
	if err != nil {
		b.log.Errorw("Failed to send message", "chat_id", chatID, "error", err)
		return errors.Wrap(err, "failed to send telegram message")
	}

	return nil
}

// SendMessageAsync sends a text message asynchronously (non-blocking)
func (b *Bot) SendMessageAsync(chatID int64, text string, callback func(messageID int, err error)) {
	go func() {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"

		sentMsg, err := b.api.Send(msg)
		if callback != nil {
			if err != nil {
				callback(0, err)
			} else {
				callback(sentMsg.MessageID, nil)
			}
		}
	}()
}

// SendMessageWithKeyboard sends message with inline keyboard (blocking)
func (b *Bot) SendMessageWithKeyboard(chatID int64, text string, keyboard telegram.InlineKeyboardMarkup) error {
	if err := b.rateLimiter.Wait(context.Background()); err != nil {
		return errors.Wrap(err, "rate limiter error")
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = convertKeyboardToTgbotapi(keyboard)

	_, err := b.api.Send(msg)
	if err != nil {
		b.log.Errorw("Failed to send message with keyboard", "chat_id", chatID, "error", err)
		return errors.Wrap(err, "failed to send telegram message")
	}

	return nil
}

// SendMessageWithOptions sends message with custom options
func (b *Bot) SendMessageWithOptions(chatID int64, text string, opts telegram.MessageOptions) (int, error) {
	if err := b.rateLimiter.Wait(context.Background()); err != nil {
		return 0, errors.Wrap(err, "rate limiter error")
	}

	msg := tgbotapi.NewMessage(chatID, text)

	// Apply options
	if opts.ParseMode != "" {
		msg.ParseMode = opts.ParseMode
	} else {
		msg.ParseMode = "Markdown" // Default
	}

	msg.DisableWebPagePreview = opts.DisableWebPagePreview
	msg.DisableNotification = opts.DisableNotification

	if opts.ReplyToMessageID > 0 {
		msg.ReplyToMessageID = opts.ReplyToMessageID
	}

	if opts.Keyboard != nil {
		msg.ReplyMarkup = convertKeyboardToTgbotapi(*opts.Keyboard)
	}

	// Send message
	sentMsg, err := b.api.Send(msg)
	if err != nil {
		b.log.Errorw("Failed to send message with options", "chat_id", chatID, "error", err)
		return 0, errors.Wrap(err, "failed to send telegram message")
	}

	// Handle self-destruct
	if opts.SelfDestruct > 0 {
		go func() {
			time.Sleep(opts.SelfDestruct)
			b.DeleteMessage(chatID, sentMsg.MessageID)
		}()
	}

	// Call OnSent callback
	if opts.OnSent != nil {
		go opts.OnSent(sentMsg.MessageID, nil)
	}

	return sentMsg.MessageID, nil
}

// EditMessage edits existing message
func (b *Bot) EditMessage(chatID int64, messageID int, text string, keyboard *telegram.InlineKeyboardMarkup) error {
	if err := b.rateLimiter.Wait(context.Background()); err != nil {
		return errors.Wrap(err, "rate limiter error")
	}

	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = "Markdown"

	if keyboard != nil {
		tgKeyboard := convertKeyboardToTgbotapi(*keyboard)
		edit.ReplyMarkup = &tgKeyboard
	}

	_, err := b.api.Send(edit)
	if err != nil {
		b.log.Debugw("Failed to edit message", "chat_id", chatID, "message_id", messageID, "error", err)
		return errors.Wrap(err, "failed to edit telegram message")
	}

	return nil
}

// DeleteMessage deletes a message
func (b *Bot) DeleteMessage(chatID int64, messageID int) error {
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	_, err := b.api.Request(deleteMsg)
	if err != nil {
		b.log.Debugw("Failed to delete message", "chat_id", chatID, "message_id", messageID, "error", err)
		return errors.Wrap(err, "failed to delete telegram message")
	}

	return nil
}

// DeleteMessageAsync deletes a message asynchronously
func (b *Bot) DeleteMessageAsync(chatID int64, messageID int, reason string) {
	go func() {
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
		if _, err := b.api.Request(deleteMsg); err != nil {
			b.log.Debugw("Failed to delete message async",
				"chat_id", chatID,
				"message_id", messageID,
				"reason", reason,
				"error", err,
			)
		}
	}()
}

// SendTemporaryMessage sends a message that auto-deletes after duration
func (b *Bot) SendTemporaryMessage(chatID int64, text string, duration time.Duration) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	sentMsg, err := b.api.Send(msg)
	if err != nil {
		return errors.Wrap(err, "failed to send temporary message")
	}

	// Schedule deletion
	go func() {
		time.Sleep(duration)
		b.DeleteMessage(chatID, sentMsg.MessageID)
	}()

	return nil
}

// AnswerCallback answers callback query
func (b *Bot) AnswerCallback(callbackQueryID string, text string, showAlert bool) error {
	callback := tgbotapi.NewCallback(callbackQueryID, text)
	callback.ShowAlert = showAlert

	_, err := b.api.Request(callback)
	if err != nil {
		b.log.Errorw("Failed to answer callback", "callback_id", callbackQueryID, "error", err)
		return errors.Wrap(err, "failed to answer callback query")
	}

	return nil
}

// =============================================================================
// Additional Helper Methods
// =============================================================================

// SendTyping sends typing action
func (b *Bot) SendTyping(chatID int64) error {
	action := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	_, err := b.api.Send(action)
	return err
}

// IsRunning checks if bot is currently running
func (b *Bot) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.running
}

// SetWebhook configures the bot to use webhook mode
func (b *Bot) SetWebhook(webhookURL string, certificate ...string) error {
	webhookConfig, err := tgbotapi.NewWebhook(webhookURL)
	if err != nil {
		return errors.Wrap(err, "failed to create webhook config")
	}

	// Optional: set certificate for self-signed cert
	if len(certificate) > 0 {
		webhookConfig.Certificate = tgbotapi.FilePath(certificate[0])
	}

	// Configure webhook settings
	webhookConfig.MaxConnections = 40
	webhookConfig.AllowedUpdates = []string{"message", "callback_query"}

	_, err = b.api.Request(webhookConfig)
	if err != nil {
		return errors.Wrap(err, "failed to set webhook")
	}

	b.log.Infow("Webhook configured successfully", "url", webhookURL)
	return nil
}

// DeleteWebhook removes webhook and returns to polling mode
func (b *Bot) DeleteWebhook(dropPendingUpdates bool) error {
	deleteConfig := tgbotapi.DeleteWebhookConfig{
		DropPendingUpdates: dropPendingUpdates,
	}

	_, err := b.api.Request(deleteConfig)
	if err != nil {
		return errors.Wrap(err, "failed to delete webhook")
	}

	b.log.Infow("Webhook deleted successfully")
	return nil
}

// GetWebhookInfo returns current webhook information
func (b *Bot) GetWebhookInfo() (telegram.WebhookInfo, error) {
	info, err := b.api.GetWebhookInfo()
	if err != nil {
		return telegram.WebhookInfo{}, errors.Wrap(err, "failed to get webhook info")
	}

	b.log.Debugw("Retrieved webhook info",
		"url", info.URL,
		"has_custom_certificate", info.HasCustomCertificate,
		"pending_update_count", info.PendingUpdateCount,
	)

	return telegram.WebhookInfo{
		URL:                  info.URL,
		HasCustomCertificate: info.HasCustomCertificate,
		PendingUpdateCount:   info.PendingUpdateCount,
		LastErrorDate:        info.LastErrorDate,
		LastErrorMessage:     info.LastErrorMessage,
		MaxConnections:       info.MaxConnections,
	}, nil
}

// HandleWebhookUpdate processes update received via webhook (call from HTTP handler)
func (b *Bot) HandleWebhookUpdate(update tgbotapi.Update) {
	if b.msgHandler != nil {
		// Convert to abstracted Update
		abstractUpdate := convertUpdate(update)
		go b.msgHandler(abstractUpdate)
	}
}

// Verify Bot implements telegram.Bot interface at compile time
var _ telegram.Bot = (*Bot)(nil)

// =============================================================================
// Conversion Functions (tgbotapi → telegram abstractions)
// =============================================================================

// convertUpdate converts tgbotapi.Update to telegram.Update (abstraction layer)
func convertUpdate(tgUpdate tgbotapi.Update) telegram.Update {
	update := telegram.Update{
		UpdateID: tgUpdate.UpdateID,
	}

	// Convert message if present
	if tgUpdate.Message != nil {
		update.Message = convertMessage(tgUpdate.Message)
	}

	// Convert callback query if present
	if tgUpdate.CallbackQuery != nil {
		update.CallbackQuery = convertCallbackQuery(tgUpdate.CallbackQuery)
	}

	return update
}

// convertMessage converts tgbotapi.Message to telegram.Message
func convertMessage(tgMsg *tgbotapi.Message) *telegram.Message {
	msg := &telegram.Message{
		MessageID: tgMsg.MessageID,
		Text:      tgMsg.Text,
		IsCommand: tgMsg.IsCommand(),
	}

	if tgMsg.From != nil {
		msg.From = convertUser(tgMsg.From)
	}

	if tgMsg.Chat != nil {
		msg.Chat = convertChat(tgMsg.Chat)
	}

	if msg.IsCommand {
		msg.Command = tgMsg.Command()
		msg.Arguments = tgMsg.CommandArguments()
	}

	if tgMsg.ReplyToMessage != nil {
		msg.ReplyTo = convertMessage(tgMsg.ReplyToMessage)
	}

	return msg
}

// convertCallbackQuery converts tgbotapi.CallbackQuery to telegram.CallbackQuery
func convertCallbackQuery(tgCallback *tgbotapi.CallbackQuery) *telegram.CallbackQuery {
	callback := &telegram.CallbackQuery{
		ID:   tgCallback.ID,
		Data: tgCallback.Data,
	}

	if tgCallback.From != nil {
		callback.From = convertUser(tgCallback.From)
	}

	if tgCallback.Message != nil {
		callback.Message = convertMessage(tgCallback.Message)
	}

	return callback
}

// convertUser converts tgbotapi.User to telegram.User
func convertUser(tgUser *tgbotapi.User) *telegram.User {
	return &telegram.User{
		ID:        tgUser.ID,
		FirstName: tgUser.FirstName,
		LastName:  tgUser.LastName,
		Username:  tgUser.UserName,
		IsBot:     tgUser.IsBot,
	}
}

// convertChat converts tgbotapi.Chat to telegram.Chat
func convertChat(tgChat *tgbotapi.Chat) *telegram.Chat {
	return &telegram.Chat{
		ID:       tgChat.ID,
		Type:     tgChat.Type,
		Title:    tgChat.Title,
		Username: tgChat.UserName,
	}
}

// convertKeyboardToTgbotapi converts telegram.InlineKeyboardMarkup to tgbotapi.InlineKeyboardMarkup
func convertKeyboardToTgbotapi(keyboard telegram.InlineKeyboardMarkup) tgbotapi.InlineKeyboardMarkup {
	var tgRows [][]tgbotapi.InlineKeyboardButton

	for _, row := range keyboard.InlineKeyboard {
		var tgRow []tgbotapi.InlineKeyboardButton
		for _, button := range row {
			tgButton := tgbotapi.InlineKeyboardButton{
				Text: button.Text,
			}

			if button.CallbackData != "" {
				tgButton.CallbackData = &button.CallbackData
			}

			if button.URL != "" {
				tgButton.URL = &button.URL
			}

			tgRow = append(tgRow, tgButton)
		}
		tgRows = append(tgRows, tgRow)
	}

	return tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: tgRows,
	}
}
