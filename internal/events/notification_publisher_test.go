package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	eventspb "prometheus/internal/events/proto"
)

func TestExchangeDeactivatedEvent_UTF8Sanitization(t *testing.T) {
	tests := []struct {
		name             string
		label            string
		reason           string
		errorMsg         string
		expectedLabel    string
		expectedReason   string
		expectedErrorMsg string
	}{
		{
			name:             "valid UTF-8 unchanged",
			label:            "My Exchange",
			reason:           "invalid_credentials",
			errorMsg:         "API key is invalid",
			expectedLabel:    "My Exchange",
			expectedReason:   "invalid_credentials",
			expectedErrorMsg: "API key is invalid",
		},
		{
			name:             "invalid UTF-8 in error message",
			label:            "My Exchange",
			reason:           "invalid_credentials",
			errorMsg:         "Error\xff from API",
			expectedLabel:    "My Exchange",
			expectedReason:   "invalid_credentials",
			expectedErrorMsg: "Error from API",
		},
		{
			name:             "invalid UTF-8 in label",
			label:            "Exchange\xfe Label",
			reason:           "invalid_credentials",
			errorMsg:         "Error message",
			expectedLabel:    "Exchange Label",
			expectedReason:   "invalid_credentials",
			expectedErrorMsg: "Error message",
		},
		{
			name:             "invalid UTF-8 in all fields",
			label:            "Bad\xff Label",
			reason:           "error\xfe reason",
			errorMsg:         "Invalid\xfd message",
			expectedLabel:    "Bad Label",
			expectedReason:   "error reason",
			expectedErrorMsg: "Invalid message",
		},
		{
			name:             "binance API error simulation",
			label:            "Binance Account",
			reason:           "invalid_credentials",
			errorMsg:         "code=-2015\xff Invalid API-key, IP, or permissions",
			expectedLabel:    "Binance Account",
			expectedReason:   "invalid_credentials",
			expectedErrorMsg: "code=-2015 Invalid API-key, IP, or permissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Sanitize fields as PublishExchangeDeactivated does
			sanitizedLabel := SanitizeUTF8(tt.label)
			sanitizedReason := SanitizeUTF8(tt.reason)
			sanitizedErrorMsg := SanitizeUTF8(tt.errorMsg)

			// Create event with sanitized fields
			event := &eventspb.ExchangeDeactivatedEvent{
				Base:         NewBaseEvent("exchange.deactivated", "exchange_service", "user-456"),
				AccountId:    "account-123",
				Exchange:     "binance",
				Label:        sanitizedLabel,
				Reason:       sanitizedReason,
				ErrorMessage: sanitizedErrorMsg,
				IsTestnet:    false,
			}

			// Marshal to protobuf (this will fail if UTF-8 is invalid)
			data, err := proto.Marshal(event)
			require.NoError(t, err, "Should successfully marshal event")
			require.NotNil(t, data)

			// Unmarshal to verify no UTF-8 errors
			var unmarshaled eventspb.ExchangeDeactivatedEvent
			err = proto.Unmarshal(data, &unmarshaled)
			require.NoError(t, err, "Should successfully unmarshal event (no UTF-8 errors)")

			// Verify sanitization worked
			assert.Equal(t, tt.expectedLabel, unmarshaled.Label, "Label should be sanitized")
			assert.Equal(t, tt.expectedReason, unmarshaled.Reason, "Reason should be sanitized")
			assert.Equal(t, tt.expectedErrorMsg, unmarshaled.ErrorMessage, "ErrorMessage should be sanitized")

			// Verify other fields are set correctly
			assert.Equal(t, "account-123", unmarshaled.AccountId)
			assert.Equal(t, "binance", unmarshaled.Exchange)
			assert.Equal(t, false, unmarshaled.IsTestnet)
			assert.Equal(t, "exchange.deactivated", unmarshaled.Base.Type)
			assert.Equal(t, "user-456", unmarshaled.Base.UserId)
		})
	}
}
