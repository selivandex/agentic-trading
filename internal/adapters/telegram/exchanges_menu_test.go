package telegram

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"prometheus/internal/domain/exchange_account"
)

func TestExchangeTypeOptions(t *testing.T) {
	// Test that exchange type options are properly defined
	assert.Len(t, exchangeTypeOptions, 5, "Should have 5 exchange types")

	// Check that all exchange types implement MenuOption interface
	for i, opt := range exchangeTypeOptions {
		assert.NotEmpty(t, opt.GetEmoji(), "Exchange %d should have emoji", i)
		assert.NotEmpty(t, opt.GetLabel(), "Exchange %d should have label", i)
		assert.NotEmpty(t, opt.GetValue(), "Exchange %d should have value", i)
	}

	// Verify Binance
	binance := exchangeTypeOptions[0]
	assert.Equal(t, "Binance", binance.Label)
	assert.Equal(t, exchange_account.ExchangeBinance, binance.Value)

	// Verify OKX
	okx := exchangeTypeOptions[2]
	assert.Equal(t, "OKX", okx.Label)
	assert.Equal(t, exchange_account.ExchangeOKX, okx.Value)
}

func TestTestnetOptions(t *testing.T) {
	// Test that testnet options are properly defined
	assert.Len(t, testnetOptions, 2, "Should have 2 testnet options")

	// Check production option
	production := testnetOptions[0]
	assert.Equal(t, "🚀", production.Emoji)
	assert.Equal(t, "Production", production.Label)
	assert.Equal(t, "false", production.Value)
	assert.NotEmpty(t, production.Description)

	// Check testnet option
	testnet := testnetOptions[1]
	assert.Equal(t, "🧪", testnet.Emoji)
	assert.Equal(t, "Testnet", testnet.Label)
	assert.Equal(t, "true", testnet.Value)
	assert.NotEmpty(t, testnet.Description)
}

func TestExchangesFlowKeys(t *testing.T) {
	// Test that flow keys are properly defined with readable names (stored in session, not in callback_data)
	assert.Equal(t, "account_id", exchangeKeys.AccountID)
	assert.Equal(t, "exchange_type", exchangeKeys.ExchangeType)
	assert.Equal(t, "is_testnet", exchangeKeys.IsTestnet)
	assert.Equal(t, "label", exchangeKeys.Label)
	assert.Equal(t, "api_key", exchangeKeys.APIKey)
	assert.Equal(t, "secret", exchangeKeys.Secret)
	assert.Equal(t, "passphrase", exchangeKeys.Passphrase)
	assert.Equal(t, "credential_index", exchangeKeys.CredIndex)
}

func TestExchangesMenuService_GetMenuType(t *testing.T) {
	ems := &ExchangesMenuService{}
	assert.Equal(t, "exchanges", ems.GetMenuType())
}

func TestExchangesMenuService_GetScreenIDs(t *testing.T) {
	ems := &ExchangesMenuService{}

	screenIDs := ems.GetScreenIDs()

	// Should return all screen IDs with readable names
	expected := []string{
		"list", "detail", "select_exchange_type", "select_testnet",
		"enter_label_add", "enter_credential_add",
		"edit_label", "edit_credentials",
	}
	assert.Equal(t, expected, screenIDs)
}

func TestGetRequiredCredentials(t *testing.T) {
	tests := []struct {
		name         string
		exchangeType exchange_account.ExchangeType
		wantFields   int
		wantNames    []string
	}{
		{
			name:         "Binance requires 2 fields",
			exchangeType: exchange_account.ExchangeBinance,
			wantFields:   2,
			wantNames:    []string{"API Key", "Secret Key"},
		},
		{
			name:         "Bybit requires 2 fields",
			exchangeType: exchange_account.ExchangeBybit,
			wantFields:   2,
			wantNames:    []string{"API Key", "Secret Key"},
		},
		{
			name:         "OKX requires 3 fields (with passphrase)",
			exchangeType: exchange_account.ExchangeOKX,
			wantFields:   3,
			wantNames:    []string{"API Key", "Secret Key", "Passphrase"},
		},
		{
			name:         "Kucoin requires 2 fields",
			exchangeType: exchange_account.ExchangeKucoin,
			wantFields:   2,
			wantNames:    []string{"API Key", "Secret Key"},
		},
		{
			name:         "Gate requires 2 fields",
			exchangeType: exchange_account.ExchangeGate,
			wantFields:   2,
			wantNames:    []string{"API Key", "Secret Key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := exchange_account.GetRequiredCredentials(tt.exchangeType)
			assert.Len(t, fields, tt.wantFields, "Should have correct number of fields")

			for i, expectedName := range tt.wantNames {
				assert.Equal(t, expectedName, fields[i].Name, "Field %d should have correct name", i)
				assert.NotEmpty(t, fields[i].Key, "Field %d should have key", i)
				assert.NotEmpty(t, fields[i].Placeholder, "Field %d should have placeholder", i)
				assert.True(t, fields[i].IsRequired, "Field %d should be required", i)
			}
		})
	}
}

func TestGetExchangeDisplayName(t *testing.T) {
	tests := []struct {
		exchange exchange_account.ExchangeType
		want     string
	}{
		{exchange_account.ExchangeBinance, "Binance"},
		{exchange_account.ExchangeBybit, "Bybit"},
		{exchange_account.ExchangeOKX, "OKX"},
		{exchange_account.ExchangeKucoin, "Kucoin"},
		{exchange_account.ExchangeGate, "Gate.io"},
	}

	for _, tt := range tests {
		t.Run(string(tt.exchange), func(t *testing.T) {
			got := exchange_account.GetExchangeDisplayName(tt.exchange)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetExchangeEmoji(t *testing.T) {
	// Test that all exchanges have emojis
	exchanges := []exchange_account.ExchangeType{
		exchange_account.ExchangeBinance,
		exchange_account.ExchangeBybit,
		exchange_account.ExchangeOKX,
		exchange_account.ExchangeKucoin,
		exchange_account.ExchangeGate,
	}

	for _, ex := range exchanges {
		t.Run(string(ex), func(t *testing.T) {
			emoji := exchange_account.GetExchangeEmoji(ex)
			assert.NotEmpty(t, emoji, "Exchange should have emoji")
		})
	}
}
