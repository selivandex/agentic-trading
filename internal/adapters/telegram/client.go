package telegram

import (
	"context"
	"fmt"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Bot represents a Telegram bot instance
type Bot struct {
	api        *tgbotapi.BotAPI
	updates    tgbotapi.UpdatesChannel
	log        *logger.Logger
	mu         sync.RWMutex
	running    bool
	msgHandler func(tgbotapi.Update) // Handler for incoming updates
}

// Config contains Telegram bot configuration
type Config struct {
	Token      string
	Debug      bool
	Timeout    int // Update timeout in seconds
	BufferSize int // Update channel buffer size
}

// NewBot creates a new Telegram bot instance
func NewBot(cfg Config, log *logger.Logger) (*Bot, error) {
	if cfg.Token == "" {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "telegram bot token is required")
	}

	api, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create telegram bot")
	}

	api.Debug = cfg.Debug

	if cfg.Timeout == 0 {
		cfg.Timeout = 60
	}
	if cfg.BufferSize == 0 {
		cfg.BufferSize = 100
	}

	log.Infof("Authorized on account %s", api.Self.UserName)

	return &Bot{
		api: api,
		log: log.With("component", "telegram_bot"),
	}, nil
}

// Start begins polling for updates
func (b *Bot) Start(ctx context.Context) error {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return errors.New("bot is already running")
	}
	b.running = true
	b.mu.Unlock()

	b.log.Info("Starting Telegram bot...")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)
	b.updates = updates

	b.log.Info("✓ Telegram bot started, waiting for updates")

	// Process updates until context is cancelled
	for {
		select {
		case <-ctx.Done():
			b.log.Info("Telegram bot stopping (context cancelled)")
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

	b.log.Info("Stopping Telegram bot...")
	b.api.StopReceivingUpdates()
	b.running = false
	b.log.Info("✓ Telegram bot stopped")
}

// handleUpdate processes a single update
func (b *Bot) handleUpdate(update tgbotapi.Update) {
	// Call registered handler if available
	if b.msgHandler != nil {
		b.msgHandler(update)
		return
	}

	// Default: just log
	if update.Message != nil {
		b.log.Debug("Received message (no handler registered)",
			"from", update.Message.From.UserName,
			"text", update.Message.Text,
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
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	_, err := b.api.Send(msg)
	if err != nil {
		return errors.Wrap(err, "failed to send message")
	}

	return nil
}

// SendMessageWithKeyboard sends a message with inline keyboard
func (b *Bot) SendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	_, err := b.api.Send(msg)
	if err != nil {
		return errors.Wrap(err, "failed to send message with keyboard")
	}

	return nil
}

// SendNotification sends a formatted notification to a user
func (b *Bot) SendNotification(chatID int64, notification string) error {
	return b.SendMessage(chatID, notification)
}

// EditMessage updates an existing message
func (b *Bot) EditMessage(chatID int64, messageID int, text string) error {
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

// AnswerCallbackQuery answers a callback query from inline keyboard
func (b *Bot) AnswerCallbackQuery(callbackQueryID string, text string) error {
	callback := tgbotapi.NewCallback(callbackQueryID, text)

	_, err := b.api.Request(callback)
	if err != nil {
		return errors.Wrap(err, "failed to answer callback query")
	}

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
func (b *Bot) SendMessageAsync(chatID int64, text string) {
	go func() {
		if err := b.SendMessage(chatID, text); err != nil {
			b.log.Error("Failed to send async message",
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
		b.log.Warn("Failed to send notification, retrying...",
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
