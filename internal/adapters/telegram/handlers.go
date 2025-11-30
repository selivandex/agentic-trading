package telegram

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"

	"prometheus/internal/domain/user"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"
)

// Handler manages command routing and execution
type Handler struct {
	bot              *Bot
	userRepo         user.Repository
	onboardingMgr    OnboardingManager
	statusHandler    StatusHandler
	portfolioHandler PortfolioHandler
	controlHandler   ControlHandler
	exchangeHandler  ExchangeHandler
	templates        *templates.Registry
	log              *logger.Logger
}

// Dependencies for Handler
type HandlerDeps struct {
	Bot              *Bot
	UserRepo         user.Repository
	OnboardingMgr    OnboardingManager
	StatusHandler    StatusHandler
	PortfolioHandler PortfolioHandler
	ControlHandler   ControlHandler
	ExchangeHandler  ExchangeHandler
	Templates        *templates.Registry
	Log              *logger.Logger
}

// NewHandler creates a new command handler
func NewHandler(deps HandlerDeps) *Handler {
	if deps.Templates == nil {
		deps.Templates = templates.Get()
	}

	return &Handler{
		bot:              deps.Bot,
		userRepo:         deps.UserRepo,
		onboardingMgr:    deps.OnboardingMgr,
		statusHandler:    deps.StatusHandler,
		portfolioHandler: deps.PortfolioHandler,
		controlHandler:   deps.ControlHandler,
		exchangeHandler:  deps.ExchangeHandler,
		templates:        deps.Templates,
		log:              deps.Log.With("component", "telegram_handler"),
	}
}

// OnboardingManager handles onboarding flow
type OnboardingManager interface {
	HandleMessage(ctx context.Context, userID uuid.UUID, telegramID int64, text string) error
	IsInOnboarding(ctx context.Context, telegramID int64) (bool, error)
}

// StatusHandler handles status-related commands
type StatusHandler interface {
	HandleStatus(ctx context.Context, chatID int64, userID uuid.UUID) error
}

// PortfolioHandler handles portfolio-related commands
type PortfolioHandler interface {
	HandlePortfolio(ctx context.Context, chatID int64, userID uuid.UUID) error
}

// ControlHandler handles control commands (stop, settings)
type ControlHandler interface {
	HandleStop(ctx context.Context, chatID int64, userID uuid.UUID) error
	HandleSettings(ctx context.Context, chatID int64, userID uuid.UUID) error
}

// ExchangeHandler handles all exchange operations (add, update, delete)
type ExchangeHandler interface {
	HandleAddExchange(ctx context.Context, chatID int64, userID uuid.UUID) error
	HandleUpdateExchange(ctx context.Context, chatID int64, userID uuid.UUID) error
	HandleMessage(ctx context.Context, userID uuid.UUID, telegramID int64, text string) error
	HandleCallback(ctx context.Context, userID uuid.UUID, telegramID int64, data string) error
	IsInSetup(ctx context.Context, telegramID int64) (bool, error)
}

// RegisterHandlers registers all command handlers with the bot
func (h *Handler) RegisterHandlers() {
	// Set our router as the message handler
	h.bot.SetMessageHandler(func(update tgbotapi.Update) {
		// Call our router
		if err := h.Route(context.Background(), update); err != nil {
			h.log.Errorw("Failed to handle update", "error", err)
		}
	})
}

// Route routes updates to appropriate handlers
func (h *Handler) Route(ctx context.Context, update tgbotapi.Update) error {
	h.log.Debugw("Routing Telegram update",
		"update_id", update.UpdateID,
		"has_message", update.Message != nil,
		"has_callback", update.CallbackQuery != nil,
	)

	// Handle messages
	if update.Message != nil {
		return h.handleMessage(ctx, update.Message)
	}

	// Handle callback queries (from inline keyboards)
	if update.CallbackQuery != nil {
		return h.handleCallbackQuery(ctx, update.CallbackQuery)
	}

	h.log.Debugw("Update type not handled",
		"update_id", update.UpdateID,
	)

	return nil
}

// handleMessage processes incoming messages
func (h *Handler) handleMessage(ctx context.Context, msg *tgbotapi.Message) error {
	if msg.From == nil {
		return nil
	}

	telegramID := msg.From.ID
	chatID := msg.Chat.ID
	text := msg.Text

	h.log.Debugw("Processing message",
		"telegram_id", telegramID,
		"username", msg.From.UserName,
		"text", text,
	)

	// Get or create user
	usr, err := h.getOrCreateUser(ctx, msg.From)
	if err != nil {
		h.bot.SendMessage(chatID, "❌ Failed to process your request. Please try again.")
		return errors.Wrap(err, "failed to get or create user")
	}

	// Check if user is in exchange setup/management flow
	if h.exchangeHandler != nil {
		inSetup, err := h.exchangeHandler.IsInSetup(ctx, telegramID)
		if err != nil {
			h.log.Errorw("Failed to check exchange setup status", "error", err)
		} else if inSetup {
			// Route to exchange handler (handles both setup and management)
			return h.exchangeHandler.HandleMessage(ctx, usr.ID, telegramID, text)
		}
	}

	// Check if user is in onboarding flow
	if h.onboardingMgr != nil {
		inOnboarding, err := h.onboardingMgr.IsInOnboarding(ctx, telegramID)
		if err != nil {
			h.log.Errorw("Failed to check onboarding status", "error", err)
		} else if inOnboarding {
			// Route to onboarding manager
			return h.onboardingMgr.HandleMessage(ctx, usr.ID, telegramID, text)
		}
	}

	// Handle commands
	if msg.IsCommand() {
		command := msg.Command()
		args := msg.CommandArguments()
		h.log.Debugw("Routing command",
			"telegram_id", telegramID,
			"user_id", usr.ID,
			"command", command,
			"has_args", args != "",
		)
		return h.handleCommand(ctx, chatID, usr, command, args)
	}

	// Non-command message
	h.log.Debugw("Received non-command message",
		"telegram_id", telegramID,
		"user_id", usr.ID,
		"text_length", len(text),
	)
	h.bot.SendMessage(chatID, "I don't understand that message. Use /help to see available commands.")
	return nil
}

// handleCommand routes commands to appropriate handlers
func (h *Handler) handleCommand(ctx context.Context, chatID int64, usr *user.User, command, args string) error {
	h.log.Infow("Handling command",
		"user_id", usr.ID,
		"command", command,
		"args", args,
	)

	switch command {
	case "start":
		h.log.Debugw("Handling /start command",
			"user_id", usr.ID,
			"telegram_id", chatID,
		)
		return h.handleStart(ctx, chatID, usr)

	case "help":
		h.log.Debugw("Handling /help command",
			"user_id", usr.ID,
			"telegram_id", chatID,
		)
		return h.handleHelp(ctx, chatID)

	case "invest":
		h.log.Debugw("Handling /invest command",
			"user_id", usr.ID,
			"telegram_id", chatID,
			"args", args,
		)
		return h.handleInvest(ctx, chatID, usr, args)

	case "status":
		h.log.Debugw("Handling /status command",
			"user_id", usr.ID,
			"telegram_id", chatID,
		)
		if h.statusHandler != nil {
			return h.statusHandler.HandleStatus(ctx, chatID, usr.ID)
		}
		return h.bot.SendMessage(chatID, "Status command not yet implemented")

	case "portfolio":
		h.log.Debugw("Handling /portfolio command",
			"user_id", usr.ID,
			"telegram_id", chatID,
		)
		if h.portfolioHandler != nil {
			return h.portfolioHandler.HandlePortfolio(ctx, chatID, usr.ID)
		}
		return h.bot.SendMessage(chatID, "Portfolio command not yet implemented")

	case "stop":
		h.log.Debugw("Handling /stop command",
			"user_id", usr.ID,
			"telegram_id", chatID,
		)
		if h.controlHandler != nil {
			return h.controlHandler.HandleStop(ctx, chatID, usr.ID)
		}
		return h.bot.SendMessage(chatID, "Stop command not yet implemented")

	case "settings":
		h.log.Debugw("Handling /settings command",
			"user_id", usr.ID,
			"telegram_id", chatID,
		)
		if h.controlHandler != nil {
			return h.controlHandler.HandleSettings(ctx, chatID, usr.ID)
		}
		return h.bot.SendMessage(chatID, "Settings command not yet implemented")

	case "add_exchange", "addexchange":
		h.log.Debugw("Handling /add_exchange command",
			"user_id", usr.ID,
			"telegram_id", chatID,
		)
		if h.exchangeHandler != nil {
			return h.exchangeHandler.HandleAddExchange(ctx, chatID, usr.ID)
		}
		return h.bot.SendMessage(chatID, "Add exchange command not yet implemented")

	case "update_exchange", "updateexchange", "manage_exchange":
		h.log.Debugw("Handling /update_exchange command",
			"user_id", usr.ID,
			"telegram_id", chatID,
		)
		if h.exchangeHandler != nil {
			return h.exchangeHandler.HandleUpdateExchange(ctx, chatID, usr.ID)
		}
		return h.bot.SendMessage(chatID, "Update exchange command not yet implemented")

	default:
		h.log.Warnw("Unknown command received",
			"user_id", usr.ID,
			"telegram_id", chatID,
			"command", command,
		)
		return h.bot.SendMessage(chatID, fmt.Sprintf("Unknown command: /%s\n\nUse /help to see available commands.", command))
	}
}

// handleStart handles /start command (user registration/welcome)
func (h *Handler) handleStart(ctx context.Context, chatID int64, usr *user.User) error {
	data := map[string]interface{}{
		"FirstName": usr.FirstName,
		"UserID":    usr.ID.String(),
	}

	welcomeMsg, err := h.templates.Render("telegram/welcome", data)
	if err != nil {
		return errors.Wrap(err, "failed to render welcome template")
	}

	return h.bot.SendMessage(chatID, welcomeMsg)
}

// handleHelp handles /help command
func (h *Handler) handleHelp(ctx context.Context, chatID int64) error {
	helpMsg, err := h.templates.Render("telegram/help", nil)
	if err != nil {
		return errors.Wrap(err, "failed to render help template")
	}

	return h.bot.SendMessage(chatID, helpMsg)
}

// handleInvest handles /invest command (starts onboarding)
func (h *Handler) handleInvest(ctx context.Context, chatID int64, usr *user.User, args string) error {
	if h.onboardingMgr == nil {
		return h.bot.SendMessage(chatID, "❌ Onboarding service not available")
	}

	// Parse amount if provided
	amount := strings.TrimSpace(args)

	if amount == "" {
		// Start interactive onboarding
		msg, err := h.templates.Render("telegram/invest_start", nil)
		if err != nil {
			return errors.Wrap(err, "failed to render invest_start template")
		}
		return h.bot.SendMessage(chatID, msg)
	}

	// Amount provided directly - start onboarding with this amount
	return h.onboardingMgr.HandleMessage(ctx, usr.ID, chatID, amount)
}

// handleCallbackQuery processes callback queries from inline keyboards
func (h *Handler) handleCallbackQuery(ctx context.Context, callback *tgbotapi.CallbackQuery) error {
	data := callback.Data
	chatID := callback.Message.Chat.ID
	telegramID := callback.From.ID

	h.log.Debugw("Processing callback query",
		"callback_id", callback.ID,
		"telegram_id", telegramID,
		"username", callback.From.UserName,
		"data", data,
		"chat_id", chatID,
	)

	// Answer the callback immediately
	if err := h.bot.AnswerCallbackQuery(callback.ID, ""); err != nil {
		h.log.Errorw("Failed to answer callback query",
			"callback_id", callback.ID,
			"error", err,
		)
	}

	// Get user
	usr, err := h.getUserByTelegramID(ctx, telegramID)
	if err != nil {
		h.log.Errorw("Failed to get user for callback query",
			"telegram_id", telegramID,
			"error", err,
		)
		h.bot.SendMessage(chatID, "❌ User not found. Please /start first.")
		return errors.Wrap(err, "failed to get user")
	}

	// Route based on callback data prefix
	parts := strings.Split(data, ":")
	if len(parts) < 1 {
		h.log.Warnw("Empty callback data",
			"telegram_id", telegramID,
		)
		return nil
	}

	action := parts[0]

	h.log.Debugw("Routing callback action",
		"telegram_id", telegramID,
		"user_id", usr.ID,
		"action", action,
		"parts_count", len(parts),
	)

	switch action {
	case "exch":
		// Handle exchange management callbacks (update, delete, etc.)
		if h.exchangeHandler != nil {
			h.log.Debugw("Routing to exchange handler (management callback)",
				"telegram_id", telegramID,
				"callback_data", data,
			)
			return h.exchangeHandler.HandleCallback(ctx, usr.ID, chatID, data)
		}
		h.log.Warnw("Exchange handler not available",
			"telegram_id", telegramID,
		)

	case "exchange":
		// Handle exchange selection callbacks
		if h.exchangeHandler != nil && len(parts) > 1 {
			exchangeType := parts[1]
			h.log.Debugw("Routing to exchange handler",
				"telegram_id", telegramID,
				"exchange_type", exchangeType,
			)
			return h.exchangeHandler.HandleMessage(ctx, usr.ID, chatID, exchangeType)
		}
		h.log.Warnw("Exchange handler not available or invalid callback data",
			"telegram_id", telegramID,
			"parts", parts,
		)

	case "settings":
		// Handle settings callbacks
		if h.controlHandler != nil {
			h.log.Debugw("Routing to settings handler",
				"telegram_id", telegramID,
			)
			return h.controlHandler.HandleSettings(ctx, chatID, usr.ID)
		}
		h.log.Warnw("Settings handler not available",
			"telegram_id", telegramID,
		)

	case "onboarding":
		// Handle onboarding callbacks
		if h.onboardingMgr != nil && len(parts) > 1 {
			h.log.Debugw("Routing to onboarding handler",
				"telegram_id", telegramID,
				"callback_value", parts[1],
			)
			return h.onboardingMgr.HandleMessage(ctx, usr.ID, chatID, parts[1])
		}
		h.log.Warnw("Onboarding handler not available or invalid callback data",
			"telegram_id", telegramID,
			"parts", parts,
		)

	default:
		h.log.Warnw("Unknown callback action",
			"telegram_id", telegramID,
			"action", action,
			"full_data", data,
		)
	}

	return nil
}

// getOrCreateUser gets existing user or creates a new one
func (h *Handler) getOrCreateUser(ctx context.Context, from *tgbotapi.User) (*user.User, error) {
	h.log.Debugw("Getting or creating user",
		"telegram_id", from.ID,
		"username", from.UserName,
	)

	// Try to get existing user
	usr, err := h.userRepo.GetByTelegramID(ctx, from.ID)
	if err == nil {
		h.log.Debugw("User found in database",
			"user_id", usr.ID,
			"telegram_id", usr.TelegramID,
		)
		return usr, nil
	}

	h.log.Debugw("User not found, creating new user",
		"telegram_id", from.ID,
		"username", from.UserName,
	)

	// User doesn't exist, create new one
	newUser := &user.User{
		ID:               uuid.New(),
		TelegramID:       from.ID,
		TelegramUsername: from.UserName,
		FirstName:        from.FirstName,
		LastName:         from.LastName,
		LanguageCode:     from.LanguageCode,
		IsActive:         true,
		IsPremium:        false, // Premium status can be checked separately
		Settings:         user.DefaultSettings(),
	}

	if err := h.userRepo.Create(ctx, newUser); err != nil {
		h.log.Errorw("Failed to create new user",
			"telegram_id", from.ID,
			"error", err,
		)
		return nil, errors.Wrap(err, "failed to create user")
	}

	h.log.Infow("Created new user",
		"user_id", newUser.ID,
		"telegram_id", newUser.TelegramID,
		"username", newUser.TelegramUsername,
	)

	return newUser, nil
}

// getUserByTelegramID gets user by Telegram ID
func (h *Handler) getUserByTelegramID(ctx context.Context, telegramID int64) (*user.User, error) {
	usr, err := h.userRepo.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user")
	}
	return usr, nil
}
