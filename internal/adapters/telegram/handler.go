package telegram

import (
	"context"

	"prometheus/internal/domain/user"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/telegram"
)

// Handler processes Telegram updates using pkg/telegram framework
type Handler struct {
	bot             telegram.Bot
	commandRegistry *telegram.CommandRegistry
	menuRegistry    *telegram.MenuRegistry
	userService     UserService
	log             *logger.Logger
}

// UserService defines interface for user operations
type UserService interface {
	GetByTelegramID(ctx context.Context, telegramID int64) (*user.User, error)
	CreateFromTelegram(ctx context.Context, telegramUser *telegram.User) (*user.User, error)
}

// NewHandler creates a new telegram handler
func NewHandler(
	bot telegram.Bot,
	commandRegistry *telegram.CommandRegistry,
	menuRegistry *telegram.MenuRegistry,
	userService UserService,
	log *logger.Logger,
) *Handler {
	return &Handler{
		bot:             bot,
		commandRegistry: commandRegistry,
		menuRegistry:    menuRegistry,
		userService:     userService,
		log:             log.With("component", "telegram_handler"),
	}
}

// HandleUpdate processes incoming Telegram update (uses abstracted types)
// This is the main entry point for all updates
func (h *Handler) HandleUpdate(update telegram.Update) {
	ctx := context.Background()

	// Handle callback queries (inline keyboard buttons)
	if update.HasCallback() {
		if err := h.handleCallbackQuery(ctx, update.CallbackQuery); err != nil {
			h.log.Errorw("Failed to handle callback query",
				"callback_id", update.CallbackQuery.ID,
				"error", err,
			)
		}
		return
	}

	// Handle regular messages
	if update.HasMessage() {
		if err := h.handleMessage(ctx, update.Message); err != nil {
			h.log.Errorw("Failed to handle message",
				"message_id", update.Message.MessageID,
				"error", err,
			)
		}
		return
	}
}

// Start starts the handler (for compatibility)
func (h *Handler) Start(ctx context.Context) error {
	h.log.Info("Telegram handler started")
	return nil
}

// Stop stops the handler gracefully
func (h *Handler) Stop() {
	h.log.Info("Telegram handler stopped")
}

// handleMessage processes incoming messages
func (h *Handler) handleMessage(ctx context.Context, msg *telegram.Message) error {
	if msg.From == nil {
		return nil
	}

	telegramID := msg.From.ID
	chatID := msg.Chat.ID
	text := msg.Text

	h.log.Debugw("Processing message",
		"telegram_id", telegramID,
		"username", msg.From.Username,
		"text", text,
	)

	// Get or create user
	usr, err := h.getOrCreateUser(ctx, msg.From)
	if err != nil {
		_ = h.bot.SendMessage(chatID, "❌ Failed to process your request. Please try again.")
		return errors.Wrap(err, "failed to get or create user")
	}

	// PRIORITY 1: Handle commands FIRST (they should cancel any active flow)
	if msg.IsCommand {
		h.log.Debugw("Routing command",
			"telegram_id", telegramID,
			"user_id", usr.ID,
			"command", msg.Command,
			"has_args", msg.Arguments != "",
		)

		return h.commandRegistry.Handle(ctx, usr, telegramID, chatID, msg.Command, msg.Arguments, text)
	}

	// PRIORITY 2: Check if user is in any menu flow (auto-routing via registry)
	if h.menuRegistry != nil {
		routed, err := h.menuRegistry.RouteTextMessage(ctx, usr, telegramID, msg.MessageID, text)
		if err != nil {
			h.log.Errorw("Menu text routing failed", "error", err, "telegram_id", telegramID)
			
			// Send error message to user
			_ = h.bot.SendMessage(chatID, err.Error())
			return err
		} else if routed {
			// Successfully routed to menu handler
			return nil
		}
	}

	// No handler found
	h.log.Debugw("Received non-command message",
		"telegram_id", telegramID,
		"user_id", usr.ID,
		"text_length", len(text),
	)

	_ = h.bot.SendMessage(chatID, "I don't understand that message. Use /help to see available commands.")
	return nil
}

// handleCallbackQuery processes callback queries from inline keyboards
func (h *Handler) handleCallbackQuery(ctx context.Context, callback *telegram.CallbackQuery) error {
	telegramID := callback.From.ID
	messageID := callback.Message.MessageID
	callbackData := callback.Data

	h.log.Debugw("Processing callback query",
		"telegram_id", telegramID,
		"message_id", messageID,
		"callback_data", callbackData,
	)

	// Get user
	usr, err := h.getOrCreateUser(ctx, callback.From)
	if err != nil {
		// Answer callback to stop loading spinner
		_ = h.bot.AnswerCallback(callback.ID, "❌ Error processing request", true)
		return errors.Wrap(err, "failed to get user")
	}

	// Answer callback query immediately (stops loading spinner)
	if err := h.bot.AnswerCallback(callback.ID, "", false); err != nil {
		h.log.Errorw("Failed to answer callback", "callback_id", callback.ID, "error", err)
	}

	// Route to menu registry
	if h.menuRegistry != nil {
		if err := h.menuRegistry.RouteCallback(ctx, usr, telegramID, messageID, callbackData); err != nil {
			h.log.Errorw("Failed to route callback",
				"telegram_id", telegramID,
				"callback_data", callbackData,
				"error", err,
			)

			_ = h.bot.SendMessage(telegramID, "❌ Failed to process your selection. Please try again.")
			return err
		}
	}

	return nil
}

// getOrCreateUser gets existing user or creates new one from Telegram user
func (h *Handler) getOrCreateUser(ctx context.Context, tgUser *telegram.User) (*user.User, error) {
	usr, err := h.userService.GetByTelegramID(ctx, tgUser.ID)
	if err == nil {
		return usr, nil
	}

	// User not found, create new
	if errors.Is(err, errors.ErrNotFound) {
		h.log.Infow("Creating new user from Telegram",
			"telegram_id", tgUser.ID,
			"username", tgUser.Username,
		)

		return h.userService.CreateFromTelegram(ctx, tgUser)
	}

	return nil, errors.Wrap(err, "failed to get user by telegram ID")
}
