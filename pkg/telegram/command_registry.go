package telegram

import (
	"context"
	"fmt"
	"strings"

	"prometheus/pkg/logger"
)

// CommandContext contains all data for command execution
type CommandContext struct {
	Ctx        context.Context
	User       interface{} // Generic user (application can cast to specific type)
	TelegramID int64
	ChatID     int64
	Command    string
	Args       string
	RawMessage string
	Bot        Bot // Bot interface for sending messages
}

// CommandHandler is a function that handles a command
type CommandHandler func(ctx *CommandContext) error

// CommandMiddleware wraps command handlers with additional logic
type CommandMiddleware func(next CommandHandler) CommandHandler

// CommandConfig defines a command registration
type CommandConfig struct {
	Name        string              // Primary command name (e.g., "invest")
	Aliases     []string            // Alternative names (e.g., ["i", "inv"])
	Description string              // Help text
	Usage       string              // Usage example (e.g., "/invest <amount>")
	Handler     CommandHandler      // Command handler function
	Middleware  []CommandMiddleware // Command-specific middleware
	Hidden      bool                // Don't show in /help
	Category    string              // Command category (e.g., "Trading", "Account")
}

// CommandRegistry manages command registration and routing
type CommandRegistry struct {
	commands   map[string]*CommandConfig // command name -> config
	middleware []CommandMiddleware       // Global middleware
	bot        Bot
	log        *logger.Logger
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry(bot Bot, log *logger.Logger) *CommandRegistry {
	return &CommandRegistry{
		commands:   make(map[string]*CommandConfig),
		middleware: make([]CommandMiddleware, 0),
		bot:        bot,
		log:        log.With("component", "command_registry"),
	}
}

// Register registers a command with the registry
func (cr *CommandRegistry) Register(config CommandConfig) {
	// Validate config
	if config.Name == "" {
		cr.log.Errorw("Cannot register command without name")
		return
	}
	if config.Handler == nil {
		cr.log.Errorw("Cannot register command without handler", "command", config.Name)
		return
	}

	// Register primary name
	cr.commands[config.Name] = &config
	cr.log.Debugw("Registered command",
		"name", config.Name,
		"aliases", config.Aliases,
		"description", config.Description,
		"category", config.Category,
	)

	// Register aliases
	for _, alias := range config.Aliases {
		cr.commands[alias] = &config
		cr.log.Debugw("Registered command alias",
			"alias", alias,
			"primary_name", config.Name,
		)
	}
}

// MustRegister registers a command and panics on error (for init-time registration)
func (cr *CommandRegistry) MustRegister(config CommandConfig) {
	if config.Name == "" || config.Handler == nil {
		panic(fmt.Sprintf("invalid command config: name=%s handler=%v", config.Name, config.Handler))
	}
	cr.Register(config)
}

// Use adds global middleware (applied to all commands)
func (cr *CommandRegistry) Use(middleware CommandMiddleware) {
	cr.middleware = append(cr.middleware, middleware)
	cr.log.Debugw("Added global middleware")
}

// Handle routes command to registered handler
func (cr *CommandRegistry) Handle(ctx context.Context, usr interface{}, telegramID, chatID int64, command, args, rawMessage string) error {
	// Normalize command name
	command = strings.ToLower(strings.TrimSpace(command))

	cr.log.Debugw("Routing command",
		"command", command,
		"telegram_id", telegramID,
		"has_args", args != "",
	)

	// Find command config
	config, exists := cr.commands[command]
	if !exists {
		cr.log.Warnw("Unknown command",
			"command", command,
			"telegram_id", telegramID,
		)
		return cr.bot.SendMessage(chatID, fmt.Sprintf("❌ Unknown command: /%s\n\nUse /help to see available commands.", command))
	}

	// Build command context
	cmdCtx := &CommandContext{
		Ctx:        ctx,
		User:       usr,
		TelegramID: telegramID,
		ChatID:     chatID,
		Command:    command,
		Args:       args,
		RawMessage: rawMessage,
		Bot:        cr.bot,
	}

	// Build middleware chain (command-specific + global)
	handler := config.Handler

	// Apply command-specific middleware (reverse order)
	for i := len(config.Middleware) - 1; i >= 0; i-- {
		handler = config.Middleware[i](handler)
	}

	// Apply global middleware (reverse order)
	for i := len(cr.middleware) - 1; i >= 0; i-- {
		handler = cr.middleware[i](handler)
	}

	// Execute with error handling
	if err := handler(cmdCtx); err != nil {
		cr.log.Errorw("Command execution failed",
			"command", command,
			"telegram_id", telegramID,
			"error", err,
		)
		return cr.handleCommandError(cmdCtx, err)
	}

	cr.log.Infow("Command executed successfully",
		"command", command,
		"telegram_id", telegramID,
	)

	return nil
}

// GetCommands returns all registered commands (for /help)
func (cr *CommandRegistry) GetCommands(includeHidden bool) []*CommandConfig {
	seen := make(map[string]bool)
	commands := make([]*CommandConfig, 0)

	for name, config := range cr.commands {
		// Skip if already added (aliases point to same config)
		if seen[config.Name] {
			continue
		}
		seen[config.Name] = true

		// Skip hidden commands unless requested
		if config.Hidden && !includeHidden {
			continue
		}

		// Only add if this is the primary name
		if name == config.Name {
			commands = append(commands, config)
		}
	}

	return commands
}

// GetCommandsByCategory returns commands grouped by category
func (cr *CommandRegistry) GetCommandsByCategory(includeHidden bool) map[string][]*CommandConfig {
	commands := cr.GetCommands(includeHidden)
	grouped := make(map[string][]*CommandConfig)

	for _, cmd := range commands {
		category := cmd.Category
		if category == "" {
			category = "General"
		}
		grouped[category] = append(grouped[category], cmd)
	}

	return grouped
}

// HasCommand checks if command is registered
func (cr *CommandRegistry) HasCommand(command string) bool {
	command = strings.ToLower(strings.TrimSpace(command))
	_, exists := cr.commands[command]
	return exists
}

// handleCommandError handles errors from command execution
func (cr *CommandRegistry) handleCommandError(cmdCtx *CommandContext, err error) error {
	// Check for validation errors
	if valErr, ok := err.(ValidationError); ok {
		return cmdCtx.Bot.SendMessage(cmdCtx.ChatID, fmt.Sprintf("❌ %s", valErr.Message))
	}

	// Generic error message
	errorMsg := "❌ Something went wrong. Please try again."
	return cmdCtx.Bot.SendMessage(cmdCtx.ChatID, errorMsg)
}
