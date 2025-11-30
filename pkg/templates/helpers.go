package templates

import (
	"strings"
)

// EscapeMarkdown escapes special characters for Telegram Markdown format
// This is a shared helper used by both templates and telegram package
func EscapeMarkdown(text string) string {
	// Characters that need to be escaped in Markdown: _ * [ ] ( ) ~ ` > # + - = | { } . !
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

// EscapeMarkdownV2 escapes special characters for Telegram MarkdownV2 format
// According to Telegram docs:
// - Any character with code between 1 and 126 can be escaped with preceding '\'
// - Inside pre and code entities, all '`' and '\' must be escaped
// - Inside (...) of inline link, all ')' and '\' must be escaped
// - In all other places: '_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!' must be escaped
func EscapeMarkdownV2(text string) string {
	// Characters that must be escaped in MarkdownV2
	replacer := strings.NewReplacer(
		"\\", "\\\\", // Backslash must be escaped first
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

// EscapeMarkdownV2Code escapes special characters inside code/pre blocks
// Inside code and pre entities, only '`' and '\' must be escaped
func EscapeMarkdownV2Code(code string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"`", "\\`",
	)
	return replacer.Replace(code)
}

// EscapeMarkdownV2Link escapes special characters inside inline link URLs
// Inside (...) part of inline link, only ')' and '\' must be escaped
func EscapeMarkdownV2Link(url string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		")", "\\)",
	)
	return replacer.Replace(url)
}

// SafeText sanitizes text for safe use in Telegram messages:
// 1. Removes invalid UTF-8 sequences
// 2. Escapes special Markdown characters
func SafeText(text string) string {
	// Ensure valid UTF-8
	text = strings.ToValidUTF8(text, "")
	// Escape special Markdown characters
	return EscapeMarkdown(text)
}

// SafeTextV2 sanitizes text for safe use in Telegram MarkdownV2 messages:
// 1. Removes invalid UTF-8 sequences
// 2. Escapes special MarkdownV2 characters
func SafeTextV2(text string) string {
	// Ensure valid UTF-8
	text = strings.ToValidUTF8(text, "")
	// Escape special MarkdownV2 characters
	return EscapeMarkdownV2(text)
}
