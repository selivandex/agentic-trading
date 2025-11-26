package model

// Model represents an LLM or tool-using model.
type Model interface {
	Name() string
	Provider() string
	MaxTokens() int
}

// BasicModel is a lightweight implementation for static metadata.
type BasicModel struct {
	ID         string
	ProviderID string
	Tokens     int
}

// Name returns the model identifier.
func (m BasicModel) Name() string { return m.ID }

// Provider returns the provider identifier.
func (m BasicModel) Provider() string { return m.ProviderID }

// MaxTokens returns the max token limit.
func (m BasicModel) MaxTokens() int { return m.Tokens }

var _ Model = BasicModel{}
