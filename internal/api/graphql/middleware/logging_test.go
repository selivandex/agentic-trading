package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeVariables(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "redacts password field",
			input: map[string]interface{}{
				"email":    "test@example.com",
				"password": "secret123",
			},
			expected: map[string]interface{}{
				"email":    "test@example.com",
				"password": "***REDACTED***",
			},
		},
		{
			name: "redacts multiple sensitive fields",
			input: map[string]interface{}{
				"username":     "john",
				"password":     "secret123",
				"token":        "eyJhbGc...",
				"apiKey":       "key123",
				"apiSecret":    "secret456",
				"accessToken":  "access123",
				"refreshToken": "refresh456",
			},
			expected: map[string]interface{}{
				"username":     "john",
				"password":     "***REDACTED***",
				"token":        "***REDACTED***",
				"apiKey":       "***REDACTED***",
				"apiSecret":    "***REDACTED***",
				"accessToken":  "***REDACTED***",
				"refreshToken": "***REDACTED***",
			},
		},
		{
			name: "handles nested objects",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"name":     "John",
					"password": "secret123",
				},
				"credentials": map[string]interface{}{
					"apiKey":    "key123",
					"apiSecret": "secret456",
				},
			},
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"name":     "John",
					"password": "***REDACTED***",
				},
				"credentials": map[string]interface{}{
					"apiKey":    "***REDACTED***",
					"apiSecret": "***REDACTED***",
				},
			},
		},
		{
			name: "handles arrays with nested objects",
			input: map[string]interface{}{
				"accounts": []interface{}{
					map[string]interface{}{
						"name":      "Account 1",
						"apiKey":    "key1",
						"apiSecret": "secret1",
					},
					map[string]interface{}{
						"name":      "Account 2",
						"apiKey":    "key2",
						"apiSecret": "secret2",
					},
				},
			},
			expected: map[string]interface{}{
				"accounts": []interface{}{
					map[string]interface{}{
						"name":      "Account 1",
						"apiKey":    "***REDACTED***",
						"apiSecret": "***REDACTED***",
					},
					map[string]interface{}{
						"name":      "Account 2",
						"apiKey":    "***REDACTED***",
						"apiSecret": "***REDACTED***",
					},
				},
			},
		},
		{
			name: "preserves non-sensitive data",
			input: map[string]interface{}{
				"userId":   "123",
				"email":    "test@example.com",
				"amount":   100.5,
				"active":   true,
				"metadata": map[string]interface{}{"key": "value"},
			},
			expected: map[string]interface{}{
				"userId":   "123",
				"email":    "test@example.com",
				"amount":   100.5,
				"active":   true,
				"metadata": map[string]interface{}{"key": "value"},
			},
		},
		{
			name:     "handles empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name: "handles deeply nested structures",
			input: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": map[string]interface{}{
							"password": "deep-secret",
							"data":     "safe-data",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": map[string]interface{}{
							"password": "***REDACTED***",
							"data":     "safe-data",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeVariables(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeArray(t *testing.T) {
	sensitiveFields := map[string]bool{
		"password": true,
		"token":    true,
	}

	tests := []struct {
		name     string
		input    []interface{}
		expected []interface{}
	}{
		{
			name: "sanitizes objects in array",
			input: []interface{}{
				map[string]interface{}{
					"name":     "Item 1",
					"password": "secret1",
				},
				map[string]interface{}{
					"name":     "Item 2",
					"password": "secret2",
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					"name":     "Item 1",
					"password": "***REDACTED***",
				},
				map[string]interface{}{
					"name":     "Item 2",
					"password": "***REDACTED***",
				},
			},
		},
		{
			name:     "handles array of primitives",
			input:    []interface{}{"string", 123, true, 45.67},
			expected: []interface{}{"string", 123, true, 45.67},
		},
		{
			name: "handles nested arrays",
			input: []interface{}{
				[]interface{}{
					map[string]interface{}{
						"token": "secret",
						"value": 100,
					},
				},
			},
			expected: []interface{}{
				[]interface{}{
					map[string]interface{}{
						"token": "***REDACTED***",
						"value": 100,
					},
				},
			},
		},
		{
			name:     "handles empty array",
			input:    []interface{}{},
			expected: []interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeArray(tt.input, sensitiveFields)
			assert.Equal(t, tt.expected, result)
		})
	}
}
