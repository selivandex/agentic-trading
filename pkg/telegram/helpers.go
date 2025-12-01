package telegram

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MessageBuilder provides fluent API for building messages
type MessageBuilder struct {
	chatID                int64
	text                  string
	parseMode             string
	keyboard              *InlineKeyboardMarkup
	disableWebPagePreview bool
	disableNotification   bool
	replyToMessageID      int
	selfDestruct          time.Duration
	onSent                func(messageID int, err error)
}

// NewMessage creates a new message builder
func NewMessage(chatID int64, text string) *MessageBuilder {
	return &MessageBuilder{
		chatID:    chatID,
		text:      text,
		parseMode: "Markdown", // Default to Markdown
	}
}

// WithMarkdown sets parse mode to Markdown
func (mb *MessageBuilder) WithMarkdown() *MessageBuilder {
	mb.parseMode = "Markdown"
	return mb
}

// WithHTML sets parse mode to HTML
func (mb *MessageBuilder) WithHTML() *MessageBuilder {
	mb.parseMode = "HTML"
	return mb
}

// WithKeyboard adds inline keyboard
func (mb *MessageBuilder) WithKeyboard(keyboard InlineKeyboardMarkup) *MessageBuilder {
	mb.keyboard = &keyboard
	return mb
}

// WithButtons adds buttons to inline keyboard
func (mb *MessageBuilder) WithButtons(buttons ...[]InlineKeyboardButton) *MessageBuilder {
	keyboard := NewInlineKeyboardMarkup(buttons...)
	mb.keyboard = &keyboard
	return mb
}

// Silent sends message silently (no notification)
func (mb *MessageBuilder) Silent() *MessageBuilder {
	mb.disableNotification = true
	return mb
}

// NoPreview disables link previews
func (mb *MessageBuilder) NoPreview() *MessageBuilder {
	mb.disableWebPagePreview = true
	return mb
}

// ReplyTo sets reply to message ID
func (mb *MessageBuilder) ReplyTo(messageID int) *MessageBuilder {
	mb.replyToMessageID = messageID
	return mb
}

// SelfDestruct sets auto-delete duration
func (mb *MessageBuilder) SelfDestruct(duration time.Duration) *MessageBuilder {
	mb.selfDestruct = duration
	return mb
}

// OnSent sets callback for when message is sent
func (mb *MessageBuilder) OnSent(callback func(messageID int, err error)) *MessageBuilder {
	mb.onSent = callback
	return mb
}

// Build creates MessageOptions from builder
func (mb *MessageBuilder) Build() MessageOptions {
	return MessageOptions{
		Keyboard:              mb.keyboard,
		ParseMode:             mb.parseMode,
		DisableWebPagePreview: mb.disableWebPagePreview,
		DisableNotification:   mb.disableNotification,
		ReplyToMessageID:      mb.replyToMessageID,
		SelfDestruct:          mb.selfDestruct,
		OnSent:                mb.onSent,
	}
}

// Send sends the message using bot
func (mb *MessageBuilder) Send(bot Bot) (int, error) {
	return bot.SendMessageWithOptions(mb.chatID, mb.text, mb.Build())
}

// Common Input Validators

// ValidateAmount validates monetary amount
func ValidateAmount(minAmount, maxAmount float64) InputValidator {
	return func(input string) ValidationResult {
		// Clean input
		input = strings.TrimSpace(input)
		input = strings.ReplaceAll(input, "$", "")
		input = strings.ReplaceAll(input, ",", "")
		input = strings.ReplaceAll(input, " ", "")

		amount, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return ValidationResult{
				Valid: false,
				Errors: []ValidationError{
					{Field: "amount", Message: "Invalid number format. Example: 1000 or 1000.50"},
				},
			}
		}

		if amount < minAmount {
			return ValidationResult{
				Valid: false,
				Errors: []ValidationError{
					{Field: "amount", Message: fmt.Sprintf("Amount must be at least $%.2f", minAmount)},
				},
			}
		}

		if maxAmount > 0 && amount > maxAmount {
			return ValidationResult{
				Valid: false,
				Errors: []ValidationError{
					{Field: "amount", Message: fmt.Sprintf("Amount cannot exceed $%.2f", maxAmount)},
				},
			}
		}

		return ValidationResult{Valid: true}
	}
}

// ValidateEmail validates email format
func ValidateEmail() InputValidator {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	return func(input string) ValidationResult {
		input = strings.TrimSpace(input)

		if !emailRegex.MatchString(input) {
			return ValidationResult{
				Valid: false,
				Errors: []ValidationError{
					{Field: "email", Message: "Invalid email format"},
				},
			}
		}

		return ValidationResult{Valid: true}
	}
}

// ValidateLength validates input length
func ValidateLength(minLen, maxLen int) InputValidator {
	return func(input string) ValidationResult {
		input = strings.TrimSpace(input)
		length := len(input)

		if length < minLen {
			return ValidationResult{
				Valid: false,
				Errors: []ValidationError{
					{Field: "input", Message: fmt.Sprintf("Must be at least %d characters", minLen)},
				},
			}
		}

		if maxLen > 0 && length > maxLen {
			return ValidationResult{
				Valid: false,
				Errors: []ValidationError{
					{Field: "input", Message: fmt.Sprintf("Must not exceed %d characters", maxLen)},
				},
			}
		}

		return ValidationResult{Valid: true}
	}
}

// ValidateChoice validates input is one of allowed choices
func ValidateChoice(choices ...string) InputValidator {
	return func(input string) ValidationResult {
		input = strings.TrimSpace(strings.ToLower(input))

		for _, choice := range choices {
			if strings.ToLower(choice) == input {
				return ValidationResult{Valid: true}
			}
		}

		return ValidationResult{
			Valid: false,
			Errors: []ValidationError{
				{Field: "input", Message: fmt.Sprintf("Must be one of: %s", strings.Join(choices, ", "))},
			},
		}
	}
}

// ChainValidators chains multiple validators (all must pass)
func ChainValidators(validators ...InputValidator) InputValidator {
	return func(input string) ValidationResult {
		var allErrors []ValidationError

		for _, validator := range validators {
			result := validator(input)
			if !result.Valid {
				allErrors = append(allErrors, result.Errors...)
			}
		}

		if len(allErrors) > 0 {
			return ValidationResult{
				Valid:  false,
				Errors: allErrors,
			}
		}

		return ValidationResult{Valid: true}
	}
}

// Keyboard Builders

// YesNoKeyboard creates yes/no inline keyboard
func YesNoKeyboard(yesCallback, noCallback string) InlineKeyboardMarkup {
	return NewInlineKeyboardMarkup(
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("✅ Yes", yesCallback),
			NewInlineKeyboardButtonData("❌ No", noCallback),
		),
	)
}

// ConfirmCancelKeyboard creates confirm/cancel inline keyboard
func ConfirmCancelKeyboard(confirmCallback, cancelCallback string) InlineKeyboardMarkup {
	return NewInlineKeyboardMarkup(
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("✅ Confirm", confirmCallback),
		),
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("❌ Cancel", cancelCallback),
		),
	)
}

// PaginationKeyboard creates pagination keyboard
func PaginationKeyboard(pagination PaginationInfo, callbackPrefix string) InlineKeyboardMarkup {
	var rows [][]InlineKeyboardButton

	// Navigation row
	var navButtons []InlineKeyboardButton

	if pagination.HasPrevPage() {
		navButtons = append(navButtons,
			NewInlineKeyboardButtonData(
				"◀️ Previous",
				fmt.Sprintf("%s:page=%d", callbackPrefix, pagination.CurrentPage-1),
			),
		)
	}

	// Page indicator
	navButtons = append(navButtons,
		NewInlineKeyboardButtonData(
			fmt.Sprintf("📄 %d/%d", pagination.CurrentPage, pagination.TotalPages),
			"noop", // No-op callback
		),
	)

	if pagination.HasNextPage() {
		navButtons = append(navButtons,
			NewInlineKeyboardButtonData(
				"Next ▶️",
				fmt.Sprintf("%s:page=%d", callbackPrefix, pagination.CurrentPage+1),
			),
		)
	}

	rows = append(rows, navButtons)

	return NewInlineKeyboardMarkup(rows...)
}

// BackButton creates a back button
func BackButton() InlineKeyboardButton {
	return NewInlineKeyboardButtonData("⬅️ Back", "back")
}

// CancelButton creates a cancel button
func CancelButton() InlineKeyboardButton {
	return NewInlineKeyboardButtonData("❌ Cancel", "cancel")
}

// Text Formatting Helpers

// Bold formats text as bold (Markdown)
func Bold(text string) string {
	return fmt.Sprintf("*%s*", text)
}

// Italic formats text as italic (Markdown)
func Italic(text string) string {
	return fmt.Sprintf("_%s_", text)
}

// Code formats text as code (Markdown)
func Code(text string) string {
	return fmt.Sprintf("`%s`", text)
}

// Link creates a markdown link
func Link(text, url string) string {
	return fmt.Sprintf("[%s](%s)", text, url)
}

// Escape escapes special markdown characters
func Escape(text string) string {
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
