package telegram

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/time/rate"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Bot represents a Telegram bot instance
type Bot struct {
	api         *tgbotapi.BotAPI
	updates     tgbotapi.UpdatesChannel
	log         *logger.Logger
	mu          sync.RWMutex
	running     bool
	webhookMode bool                  // If true, use webhook instead of polling
	msgHandler  func(tgbotapi.Update) // Handler for incoming updates
	rateLimiter *rate.Limiter         // Rate limiter for Telegram API calls
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

// NewBot creates a new Telegram bot instance
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
		cfg.RateLimitBurst = 30 // Telegram allows bursts
	}
	if cfg.RateLimitRate == 0 {
		cfg.RateLimitRate = 20 // Conservative: 20 msg/sec (Telegram limit is 30)
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

	return &Bot{
		api:         api,
		webhookMode: cfg.WebhookMode,
		log:         log.With("component", "telegram_bot"),
		rateLimiter: rateLimiter,
	}, nil
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

	// If webhook mode, just block until context is cancelled
	// HTTP server handles webhook requests, no polling needed
	if b.webhookMode {
		b.log.Infow("Telegram bot in webhook mode (no polling)")
		<-ctx.Done()
		b.log.Infow("Telegram bot stopping (context cancelled)")
		b.Stop()
		return nil
	}

	// Polling mode: get updates from Telegram
	b.log.Infow("Starting Telegram bot in polling mode...")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)
	b.updates = updates

	b.log.Infow("✓ Telegram bot started, waiting for updates")

	// Process updates until context is cancelled
	for {
		select {
		case <-ctx.Done():
			b.log.Infow("Telegram bot stopping (context cancelled)")
			b.Stop()
			return nil

		case update := <-updates:
			// Handle update in goroutine to avoid blocking
			go b.handleUpdate(update)
		}
	}
}

// Stop gracefully stops the bot
func (b *Bot) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running {
		return
	}

	b.log.Infow("Stopping Telegram bot...")
	b.api.StopReceivingUpdates()
	b.running = false
	b.log.Infow("✓ Telegram bot stopped")
}

// handleUpdate processes a single update
func (b *Bot) handleUpdate(update tgbotapi.Update) {
	b.log.Debugw("Handling Telegram update",
		"update_id", update.UpdateID,
	)

	// Call registered handler if available
	if b.msgHandler != nil {
		b.msgHandler(update)
		return
	}

	// Default: just log
	if update.Message != nil {
		b.log.Debugw("Received message (no handler registered)",
			"update_id", update.UpdateID,
			"from", update.Message.From.UserName,
			"from_id", update.Message.From.ID,
			"text", update.Message.Text,
		)
	} else if update.CallbackQuery != nil {
		b.log.Debugw("Received callback query (no handler registered)",
			"update_id", update.UpdateID,
			"from", update.CallbackQuery.From.UserName,
			"from_id", update.CallbackQuery.From.ID,
			"data", update.CallbackQuery.Data,
		)
	} else {
		b.log.Debugw("Received update with no recognized type (no handler registered)",
			"update_id", update.UpdateID,
		)
	}
}

// SetMessageHandler registers a handler for incoming updates
func (b *Bot) SetMessageHandler(handler func(tgbotapi.Update)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.msgHandler = handler
}

// SendMessage sends a text message to a chat
func (b *Bot) SendMessage(chatID int64, text string) error {
	return b.SendMessageWithContext(context.Background(), chatID, text)
}

// SendMessageWithContext sends a text message to a chat with context support
func (b *Bot) SendMessageWithContext(ctx context.Context, chatID int64, text string) error {
	// Wait for rate limiter
	if err := b.rateLimiter.Wait(ctx); err != nil {
		return errors.Wrap(err, "rate limiter wait failed")
	}

	b.log.Debugw("Sending message",
		"chat_id", chatID,
		"text_length", len(text),
	)

	start := time.Now()

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	_, err := b.api.Send(msg)

	duration := time.Since(start)

	if err != nil {
		b.log.Errorw("Failed to send message",
			"chat_id", chatID,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return errors.Wrap(err, "failed to send message")
	}

	b.log.Debugw("Message sent successfully",
		"chat_id", chatID,
		"duration_ms", duration.Milliseconds(),
	)

	return nil
}

// SendMessageWithKeyboard sends a message with inline keyboard
func (b *Bot) SendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	return b.SendMessageWithKeyboardContext(context.Background(), chatID, text, keyboard)
}

// SendMessageWithKeyboardContext sends a message with inline keyboard and context support
func (b *Bot) SendMessageWithKeyboardContext(ctx context.Context, chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	// Wait for rate limiter
	if err := b.rateLimiter.Wait(ctx); err != nil {
		return errors.Wrap(err, "rate limiter wait failed")
	}

	b.log.Debugw("Sending message with inline keyboard",
		"chat_id", chatID,
		"text_length", len(text),
		"buttons_count", len(keyboard.InlineKeyboard),
	)

	start := time.Now()

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	_, err := b.api.Send(msg)

	duration := time.Since(start)

	if err != nil {
		b.log.Errorw("Failed to send message with keyboard",
			"chat_id", chatID,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return errors.Wrap(err, "failed to send message with keyboard")
	}

	b.log.Debugw("Message with keyboard sent successfully",
		"chat_id", chatID,
		"duration_ms", duration.Milliseconds(),
	)

	return nil
}

// SendNotification sends a formatted notification to a user
func (b *Bot) SendNotification(chatID int64, notification string) error {
	return b.SendMessage(chatID, notification)
}

// SendMessageAndGetID sends a message and returns the sent message (with ID)
// Use this when you need to save the message ID for later operations
func (b *Bot) SendMessageAndGetID(chatID int64, text string) (tgbotapi.Message, error) {
	return b.SendMessageAndGetIDWithContext(context.Background(), chatID, text)
}

// SendMessageAndGetIDWithContext sends a message and returns the sent message with context support
func (b *Bot) SendMessageAndGetIDWithContext(ctx context.Context, chatID int64, text string) (tgbotapi.Message, error) {
	// Wait for rate limiter
	if err := b.rateLimiter.Wait(ctx); err != nil {
		return tgbotapi.Message{}, errors.Wrap(err, "rate limiter wait failed")
	}

	b.log.Debugw("Sending message (with ID return)",
		"chat_id", chatID,
		"text_length", len(text),
	)

	start := time.Now()

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	sentMsg, err := b.api.Send(msg)

	duration := time.Since(start)

	if err != nil {
		b.log.Errorw("Failed to send message",
			"chat_id", chatID,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return tgbotapi.Message{}, errors.Wrap(err, "failed to send message")
	}

	b.log.Debugw("Message sent successfully",
		"chat_id", chatID,
		"message_id", sentMsg.MessageID,
		"duration_ms", duration.Milliseconds(),
	)

	return sentMsg, nil
}

// SendMessageWithKeyboardAndGetID sends a message with keyboard and returns the sent message
func (b *Bot) SendMessageWithKeyboardAndGetID(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) (tgbotapi.Message, error) {
	return b.SendMessageWithKeyboardAndGetIDContext(context.Background(), chatID, text, keyboard)
}

// SendMessageWithKeyboardAndGetIDContext sends a message with keyboard and returns the sent message with context
func (b *Bot) SendMessageWithKeyboardAndGetIDContext(ctx context.Context, chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) (tgbotapi.Message, error) {
	// Wait for rate limiter
	if err := b.rateLimiter.Wait(ctx); err != nil {
		return tgbotapi.Message{}, errors.Wrap(err, "rate limiter wait failed")
	}

	b.log.Debugw("Sending message with inline keyboard (with ID return)",
		"chat_id", chatID,
		"text_length", len(text),
		"buttons_count", len(keyboard.InlineKeyboard),
	)

	start := time.Now()

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := b.api.Send(msg)

	duration := time.Since(start)

	if err != nil {
		b.log.Errorw("Failed to send message with keyboard",
			"chat_id", chatID,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return tgbotapi.Message{}, errors.Wrap(err, "failed to send message with keyboard")
	}

	b.log.Debugw("Message with keyboard sent successfully",
		"chat_id", chatID,
		"message_id", sentMsg.MessageID,
		"duration_ms", duration.Milliseconds(),
	)

	return sentMsg, nil
}

// EditMessageWithKeyboard edits message with new text and keyboard
func (b *Bot) EditMessageWithKeyboard(chatID int64, messageID int, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	return b.EditMessageWithKeyboardContext(context.Background(), chatID, messageID, text, keyboard)
}

// EditMessageWithKeyboardContext edits message with new text and keyboard with context support
func (b *Bot) EditMessageWithKeyboardContext(ctx context.Context, chatID int64, messageID int, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	// Wait for rate limiter
	if err := b.rateLimiter.Wait(ctx); err != nil {
		return errors.Wrap(err, "rate limiter wait failed")
	}

	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = "Markdown"
	edit.ReplyMarkup = &keyboard

	_, err := b.api.Send(edit)
	if err != nil {
		return errors.Wrap(err, "failed to edit message")
	}

	return nil
}

// EditMessage updates an existing message
func (b *Bot) EditMessage(chatID int64, messageID int, text string) error {
	return b.EditMessageWithContext(context.Background(), chatID, messageID, text)
}

// EditMessageWithContext updates an existing message with context support
func (b *Bot) EditMessageWithContext(ctx context.Context, chatID int64, messageID int, text string) error {
	// Wait for rate limiter
	if err := b.rateLimiter.Wait(ctx); err != nil {
		return errors.Wrap(err, "rate limiter wait failed")
	}

	msg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	msg.ParseMode = "Markdown"

	_, err := b.api.Send(msg)
	if err != nil {
		return errors.Wrap(err, "failed to edit message")
	}

	return nil
}

// DeleteMessage deletes a message
func (b *Bot) DeleteMessage(chatID int64, messageID int) error {
	msg := tgbotapi.NewDeleteMessage(chatID, messageID)

	_, err := b.api.Request(msg)
	if err != nil {
		return errors.Wrap(err, "failed to delete message")
	}

	return nil
}

// DeleteMessageAsync deletes a message asynchronously with logging
// Use this for security-sensitive messages (API keys, secrets, etc.)
func (b *Bot) DeleteMessageAsync(chatID int64, messageID int, reason string) {
	if messageID <= 0 {
		return
	}

	go func(cid int64, mid int, r string) {
		if err := b.DeleteMessage(cid, mid); err != nil {
			b.log.Warnw("Failed to delete message",
				"chat_id", cid,
				"message_id", mid,
				"reason", r,
				"error", err,
			)
		} else {
			b.log.Debugw("✓ Message deleted",
				"chat_id", cid,
				"message_id", mid,
				"reason", r,
			)
		}
	}(chatID, messageID, reason)
}

// AnswerCallbackQuery answers a callback query from inline keyboard
func (b *Bot) AnswerCallbackQuery(callbackQueryID string, text string) error {
	return b.AnswerCallbackQueryWithContext(context.Background(), callbackQueryID, text)
}

// AnswerCallbackQueryWithContext answers a callback query with context support
func (b *Bot) AnswerCallbackQueryWithContext(ctx context.Context, callbackQueryID string, text string) error {
	// Wait for rate limiter
	if err := b.rateLimiter.Wait(ctx); err != nil {
		return errors.Wrap(err, "rate limiter wait failed")
	}

	b.log.Debugw("Answering callback query",
		"callback_id", callbackQueryID,
		"text", text,
	)

	callback := tgbotapi.NewCallback(callbackQueryID, text)

	_, err := b.api.Request(callback)
	if err != nil {
		b.log.Errorw("Failed to answer callback query",
			"callback_id", callbackQueryID,
			"error", err,
		)
		return errors.Wrap(err, "failed to answer callback query")
	}

	b.log.Debugw("Callback query answered successfully",
		"callback_id", callbackQueryID,
	)

	return nil
}

// GetAPI returns the underlying Telegram Bot API instance
func (b *Bot) GetAPI() *tgbotapi.BotAPI {
	return b.api
}

// IsRunning returns whether the bot is currently running
func (b *Bot) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.running
}

// SendTyping sends "typing..." action to chat
func (b *Bot) SendTyping(chatID int64) error {
	action := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	_, err := b.api.Send(action)
	return err
}

// SendMessageAsync sends a message asynchronously without blocking
// Use this for non-critical messages where you don't need to wait for result
func (b *Bot) SendMessageAsync(chatID int64, text string) {
	b.SendMessageAsyncWithCallback(chatID, text, nil)
}

// SendMessageAsyncWithCallback sends a message asynchronously with optional callback
func (b *Bot) SendMessageAsyncWithCallback(chatID int64, text string, callback func(error)) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := b.SendMessageWithContext(ctx, chatID, text)
		if err != nil {
			b.log.Errorw("Failed to send async message",
				"chat_id", chatID,
				"error", err,
			)
		}

		if callback != nil {
			callback(err)
		}
	}()
}

// SendMessageWithKeyboardAsync sends a message with keyboard asynchronously
func (b *Bot) SendMessageWithKeyboardAsync(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := b.SendMessageWithKeyboardContext(ctx, chatID, text, keyboard); err != nil {
			b.log.Errorw("Failed to send async message with keyboard",
				"chat_id", chatID,
				"error", err,
			)
		}
	}()
}

// SendNotificationWithRetry sends notification with retry logic
func (b *Bot) SendNotificationWithRetry(chatID int64, notification string, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := b.SendNotification(chatID, notification)
		if err == nil {
			return nil
		}

		lastErr = err
		b.log.Warnw("Failed to send notification, retrying...",
			"attempt", attempt+1,
			"max_retries", maxRetries,
			"error", err,
		)

		// Exponential backoff
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}

	return errors.Wrapf(lastErr, "failed to send notification after %d retries", maxRetries)
}

// BroadcastMessage sends a message to multiple chats
func (b *Bot) BroadcastMessage(chatIDs []int64, text string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(chatIDs))

	for _, chatID := range chatIDs {
		wg.Add(1)
		go func(cid int64) {
			defer wg.Done()
			if err := b.SendMessage(cid, text); err != nil {
				errChan <- fmt.Errorf("chat %d: %w", cid, err)
			}
		}(chatID)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Wrapf(errors.ErrInternal, "failed to send to %d chats: %v", len(errs), errs)
	}

	return nil
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
func (b *Bot) GetWebhookInfo() (*tgbotapi.WebhookInfo, error) {
	info, err := b.api.GetWebhookInfo()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get webhook info")
	}

	b.log.Debugw("Retrieved webhook info",
		"url", info.URL,
		"has_custom_certificate", info.HasCustomCertificate,
		"pending_update_count", info.PendingUpdateCount,
		"max_connections", info.MaxConnections,
	)

	return &info, nil
}

// SendSelfDestructingMessage sends a message and immediately deletes the user's message containing sensitive data
// Optionally deletes the response message after a specified duration
func (b *Bot) SendSelfDestructingMessage(chatID int64, userMessageID int, text string, deleteResponseAfter time.Duration) error {
	b.log.Debugw("Sending self-destructing message",
		"chat_id", chatID,
		"user_message_id", userMessageID,
		"delete_response_after", deleteResponseAfter,
	)

	// Delete user's message immediately (contains sensitive data like API keys)
	b.DeleteMessageAsync(chatID, userMessageID, "sensitive: user input")

	// Send response message
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	sentMsg, err := b.api.Send(msg)
	if err != nil {
		b.log.Errorw("Failed to send self-destructing message",
			"chat_id", chatID,
			"error", err,
		)
		return errors.Wrap(err, "failed to send message")
	}

	b.log.Debugw("Self-destructing message sent",
		"chat_id", chatID,
		"sent_message_id", sentMsg.MessageID,
	)

	// If delete duration specified, schedule deletion of response message
	if deleteResponseAfter > 0 {
		go func(cid int64, mid int, duration time.Duration) {
			time.Sleep(duration)
			if err := b.DeleteMessage(cid, mid); err != nil {
				b.log.Warnw("Failed to delete response message",
					"chat_id", cid,
					"message_id", mid,
					"error", err,
				)
			} else {
				b.log.Debugw("Response message deleted after timeout",
					"chat_id", cid,
					"message_id", mid,
					"after", duration,
				)
			}
		}(chatID, sentMsg.MessageID, deleteResponseAfter)
	}

	return nil
}
