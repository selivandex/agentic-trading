package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special characters",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "asterisk for bold",
			input:    "This is *bold*",
			expected: "This is \\*bold\\*",
		},
		{
			name:     "underscore for italic",
			input:    "This is _italic_",
			expected: "This is \\_italic\\_",
		},
		{
			name:     "brackets",
			input:    "Array[0]",
			expected: "Array\\[0\\]",
		},
		{
			name:     "parentheses",
			input:    "Function(param)",
			expected: "Function\\(param\\)",
		},
		{
			name:     "multiple special chars",
			input:    "Price: $1,234.56 (+5%)",
			expected: "Price: $1,234\\.56 \\(\\+5%\\)",
		},
		{
			name:     "error message with special chars",
			input:    "Error: API-key invalid (code=-2015)",
			expected: "Error: API\\-key invalid \\(code\\=\\-2015\\)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeMarkdown(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeMarkdownV2(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special characters",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "backslash must be escaped first",
			input:    "Path\\to\\file",
			expected: "Path\\\\to\\\\file",
		},
		{
			name:     "all special characters",
			input:    "_*[]()~`>#+-=|{}.!",
			expected: "\\_\\*\\[\\]\\(\\)\\~\\`\\>\\#\\+\\-\\=\\|\\{\\}\\.\\!",
		},
		{
			name:     "binance error message",
			input:    "code=-2015: Invalid API-key",
			expected: "code\\=\\-2015: Invalid API\\-key",
		},
		{
			name:     "price with special chars",
			input:    "$1,234.56 (+5.2%)",
			expected: "$1,234\\.56 \\(\\+5\\.2%\\)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeMarkdownV2(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeMarkdownV2Code(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "backtick escaped",
			input:    "const x = `template`",
			expected: "const x = \\`template\\`",
		},
		{
			name:     "backslash escaped",
			input:    "path\\to\\file",
			expected: "path\\\\to\\\\file",
		},
		{
			name:     "both escaped",
			input:    "`code\\path`",
			expected: "\\`code\\\\path\\`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeMarkdownV2Code(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeMarkdownV2Link(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal URL",
			input:    "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "URL with closing parenthesis",
			input:    "https://en.wikipedia.org/wiki/Markdown_(software)",
			expected: "https://en.wikipedia.org/wiki/Markdown_(software\\)",
		},
		{
			name:     "URL with backslash",
			input:    "https://example.com\\path",
			expected: "https://example.com\\\\path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeMarkdownV2Link(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid UTF-8 with Markdown",
			input:    "Price: $100 (high)",
			expected: "Price: $100 \\(high\\)",
		},
		{
			name:     "invalid UTF-8 removed",
			input:    "Error\xff message",
			expected: "Error message",
		},
		{
			name:     "invalid UTF-8 and special chars",
			input:    "API-key\xff invalid (code=-2015)",
			expected: "API\\-key invalid \\(code\\=\\-2015\\)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeText(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeTextV2(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid UTF-8 with MarkdownV2",
			input:    "Price: $100 (high)",
			expected: "Price: $100 \\(high\\)",
		},
		{
			name:     "invalid UTF-8 removed",
			input:    "Error\xff message",
			expected: "Error message",
		},
		{
			name:     "invalid UTF-8 and special chars",
			input:    "API-key\xff invalid (code=-2015)",
			expected: "API\\-key invalid \\(code\\=\\-2015\\)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeTextV2(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
