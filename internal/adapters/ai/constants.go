package ai

// ProviderName represents an AI provider identifier
type ProviderName string

// Provider name constants
const (
	ProviderNameAnthropic ProviderName = "anthropic"
	ProviderNameOpenAI    ProviderName = "openai"
	ProviderNameGoogle    ProviderName = "google"
	ProviderNameDeepSeek  ProviderName = "deepseek"
)

// String returns the string representation of the provider name
func (p ProviderName) String() string {
	return string(p)
}

// IsValid checks if the provider name is supported
func (p ProviderName) IsValid() bool {
	switch p {
	case ProviderNameAnthropic, ProviderNameOpenAI, ProviderNameGoogle, ProviderNameDeepSeek:
		return true
	default:
		return false
	}
}

// AllProviderNames returns all supported provider names
func AllProviderNames() []ProviderName {
	return []ProviderName{
		ProviderNameAnthropic,
		ProviderNameOpenAI,
		ProviderNameGoogle,
		ProviderNameDeepSeek,
	}
}
