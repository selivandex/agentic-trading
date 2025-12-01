package exchange_account

// CredentialField represents a single credential field required by an exchange
type CredentialField struct {
	Name        string // Display name: "API Key", "Secret", "Passphrase"
	Key         string // Session key: "api_key", "secret", "passphrase"
	Placeholder string // Placeholder text for input prompts
	IsRequired  bool   // Whether this field is required
}

// GetRequiredCredentials returns the list of required credential fields for a given exchange type
// This is used to dynamically build credential input flows in the Telegram bot
func GetRequiredCredentials(exchange ExchangeType) []CredentialField {
	switch exchange {
	case ExchangeOKX:
		// OKX requires API Key, Secret, and Passphrase
		return []CredentialField{
			{
				Name:        "API Key",
				Key:         "api_key",
				Placeholder: "Enter your OKX API Key",
				IsRequired:  true,
			},
			{
				Name:        "Secret Key",
				Key:         "secret",
				Placeholder: "Enter your OKX Secret Key",
				IsRequired:  true,
			},
			{
				Name:        "Passphrase",
				Key:         "passphrase",
				Placeholder: "Enter your OKX API Passphrase",
				IsRequired:  true,
			},
		}
	default:
		// Binance, Bybit, Kucoin, Gate require only API Key and Secret
		return []CredentialField{
			{
				Name:        "API Key",
				Key:         "api_key",
				Placeholder: "Enter your API Key",
				IsRequired:  true,
			},
			{
				Name:        "Secret Key",
				Key:         "secret",
				Placeholder: "Enter your Secret Key",
				IsRequired:  true,
			},
		}
	}
}

// GetExchangeDisplayName returns a human-readable display name for the exchange
func GetExchangeDisplayName(exchange ExchangeType) string {
	switch exchange {
	case ExchangeBinance:
		return "Binance"
	case ExchangeBybit:
		return "Bybit"
	case ExchangeOKX:
		return "OKX"
	case ExchangeKucoin:
		return "Kucoin"
	case ExchangeGate:
		return "Gate.io"
	default:
		return string(exchange)
	}
}

// GetExchangeEmoji returns an emoji for the exchange
func GetExchangeEmoji(exchange ExchangeType) string {
	switch exchange {
	case ExchangeBinance:
		return "🟡"
	case ExchangeBybit:
		return "🟠"
	case ExchangeOKX:
		return "⚫"
	case ExchangeKucoin:
		return "🟢"
	case ExchangeGate:
		return "🔵"
	default:
		return "📊"
	}
}
