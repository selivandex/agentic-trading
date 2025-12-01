package telegram

import (
	"context"
	"fmt"
	"time"
)

// Bot interface abstracts telegram bot operations (for dependency injection)
type Bot interface {
	// Start starts the bot (polling or webhook mode)
	Start(ctx context.Context) error

	// Stop stops the bot
	Stop()

	// SetHandler sets update handler
	SetHandler(handler func(Update))

	// SendMessage sends a text message (blocking)
	SendMessage(chatID int64, text string) error

	// SendMessageAsync sends a text message asynchronously (non-blocking)
	SendMessageAsync(chatID int64, text string, callback func(messageID int, err error))

	// SendMessageWithKeyboard sends message with inline keyboard (blocking)
	SendMessageWithKeyboard(chatID int64, text string, keyboard InlineKeyboardMarkup) error

	// SendMessageWithOptions sends message with custom options
	SendMessageWithOptions(chatID int64, text string, opts MessageOptions) (int, error)

	// EditMessage edits existing message
	EditMessage(chatID int64, messageID int, text string, keyboard *InlineKeyboardMarkup) error

	// DeleteMessage deletes a message
	DeleteMessage(chatID int64, messageID int) error

	// DeleteMessageAsync deletes a message asynchronously
	DeleteMessageAsync(chatID int64, messageID int, reason string)

	// DeleteMessageAfter deletes a message after specified delay (for security - credential messages)
	DeleteMessageAfter(chatID int64, messageID int, delay time.Duration)

	// SendTemporaryMessage sends a message that auto-deletes after duration
	SendTemporaryMessage(chatID int64, text string, duration time.Duration) error

	// AnswerCallback answers callback query
	AnswerCallback(callbackQueryID string, text string, showAlert bool) error
}

// MessageOptions defines options for sending messages
type MessageOptions struct {
	// Keyboard for inline buttons
	Keyboard *InlineKeyboardMarkup

	// ParseMode (Markdown, HTML, MarkdownV2)
	ParseMode string

	// DisableWebPagePreview disables link previews
	DisableWebPagePreview bool

	// DisableNotification sends message silently
	DisableNotification bool

	// ReplyToMessageID replies to specific message
	ReplyToMessageID int

	// SelfDestruct auto-deletes message after duration (0 = no auto-delete)
	SelfDestruct time.Duration

	// OnSent callback called after message is sent (async)
	OnSent func(messageID int, err error)
}

// Session represents a menu session state
type Session interface {
	// GetTelegramID returns telegram user ID
	GetTelegramID() int64

	// GetMessageID returns current message ID
	GetMessageID() int

	// SetMessageID sets current message ID
	SetMessageID(messageID int)

	// GetCurrentScreen returns current screen ID
	GetCurrentScreen() string

	// SetCurrentScreen sets current screen ID
	SetCurrentScreen(screenID string)

	// GetData retrieves session data by key
	GetData(key string) (interface{}, bool)

	// SetData stores session data
	SetData(key string, value interface{})

	// GetString retrieves string value from session
	GetString(key string) (string, bool)

	// SetCallbackData stores callback parameters with short key (for Telegram 64-byte limit)
	SetCallbackData(key string, data map[string]interface{})

	// GetCallbackData retrieves callback parameters by short key
	GetCallbackData(key string) (map[string]interface{}, bool)

	// ClearCallbackData removes all callback data (called when navigating to new screen)
	ClearCallbackData()

	// PushScreen adds screen to navigation stack
	PushScreen(screenID string)

	// PopScreen removes and returns last screen from stack
	PopScreen() (string, bool)

	// HasNavigationHistory checks if there are screens in navigation stack
	HasNavigationHistory() bool

	// GetCreatedAt returns session creation time
	GetCreatedAt() time.Time

	// GetUpdatedAt returns last update time
	GetUpdatedAt() time.Time
}

// SessionService defines interface for session storage operations
type SessionService interface {
	// GetSession retrieves session for telegram user
	GetSession(ctx context.Context, telegramID int64) (Session, error)

	// SaveSession saves session with TTL
	SaveSession(ctx context.Context, session Session, ttl time.Duration) error

	// DeleteSession removes session
	DeleteSession(ctx context.Context, telegramID int64) error

	// SessionExists checks if session exists
	SessionExists(ctx context.Context, telegramID int64) (bool, error)

	// CreateSession creates new session
	CreateSession(ctx context.Context, telegramID int64, initialScreen string, initialData map[string]interface{}, ttl time.Duration) (Session, error)
}

// TemplateRenderer defines interface for rendering message templates
type TemplateRenderer interface {
	// Render renders a template with data (accepts any type - struct or map)
	Render(templatePath string, data interface{}) (string, error)
}

// MenuOption represents a selectable option in menu (universal interface)
type MenuOption interface {
	// GetValue returns unique value for this option (used in callbacks)
	GetValue() string

	// GetLabel returns display label for button
	GetLabel() string

	// GetEmoji returns emoji prefix for button (optional)
	GetEmoji() string
}

// ListItem represents an item in dynamic list
type ListItem struct {
	ID           string      // Unique identifier (saved to session)
	ButtonText   string      // Text for button
	TemplateData interface{} // Data for template rendering (optional)
}

// ValidationError represents validation failure
type ValidationError struct {
	Field   string
	Message string
}

// Error implements error interface
func (v ValidationError) Error() string {
	return v.Message
}

// ValidationResult contains validation results
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// IsValid checks if validation passed
func (vr ValidationResult) IsValid() bool {
	return vr.Valid && len(vr.Errors) == 0
}

// FirstError returns first validation error or empty string
func (vr ValidationResult) FirstError() string {
	if len(vr.Errors) > 0 {
		return vr.Errors[0].Message
	}
	return ""
}

// InputValidator validates user input
type InputValidator func(input string) ValidationResult

// ProgressInfo represents multi-step progress
type ProgressInfo struct {
	CurrentStep int
	TotalSteps  int
	StepName    string
}

// FormatProgress formats progress as "Step 2/4: Select Market"
func (p ProgressInfo) FormatProgress() string {
	if p.TotalSteps == 0 {
		return ""
	}
	if p.StepName != "" {
		return fmt.Sprintf("Step %d/%d: %s", p.CurrentStep, p.TotalSteps, p.StepName)
	}
	return fmt.Sprintf("Step %d/%d", p.CurrentStep, p.TotalSteps)
}

// PaginationInfo represents pagination state
type PaginationInfo struct {
	CurrentPage int
	TotalPages  int
	PageSize    int
	TotalItems  int
}

// HasNextPage checks if there's a next page
func (p PaginationInfo) HasNextPage() bool {
	return p.CurrentPage < p.TotalPages
}

// HasPrevPage checks if there's a previous page
func (p PaginationInfo) HasPrevPage() bool {
	return p.CurrentPage > 1
}

// WebhookInfo contains information about current webhook setup
type WebhookInfo struct {
	URL                  string
	HasCustomCertificate bool
	PendingUpdateCount   int
	LastErrorDate        int
	LastErrorMessage     string
	MaxConnections       int
}
