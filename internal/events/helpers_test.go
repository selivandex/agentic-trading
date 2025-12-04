package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeUTF8(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid UTF-8 string unchanged",
			input:    "Hello, World! 你好世界",
			expected: "Hello, World! 你好世界",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name: "invalid UTF-8 bytes removed",
			// \xff is invalid UTF-8
			input:    "Hello\xffWorld",
			expected: "HelloWorld",
		},
		{
			name: "multiple invalid UTF-8 sequences",
			// Multiple invalid bytes
			input:    "Start\xffMiddle\xfeEnd\xfd",
			expected: "StartMiddleEnd",
		},
		{
			name: "mixed valid and invalid UTF-8",
			// Valid emoji followed by invalid bytes
			input:    "Test 🚀\xff error\xfe message",
			expected: "Test 🚀 error message",
		},
		{
			name: "binance error with invalid UTF-8",
			// Simulate Binance API error that could contain invalid bytes
			input:    "code=-2015\xff Invalid API-key",
			expected: "code=-2015 Invalid API-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeUTF8(tt.input)
			assert.Equal(t, tt.expected, result, "SanitizeUTF8 should remove invalid UTF-8 sequences")
		})
	}
}


