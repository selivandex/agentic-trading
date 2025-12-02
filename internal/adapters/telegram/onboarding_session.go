package telegram

import (
	"time"

	"github.com/google/uuid"
)

// OnboardingState represents different states in the onboarding flow
type OnboardingState string

const (
	StateIdle                OnboardingState = "idle"
	StateSelectExchange      OnboardingState = "select_exchange"
	StateAwaitingCapital     OnboardingState = "awaiting_capital"
	StateAwaitingRiskProfile OnboardingState = "awaiting_risk_profile"
	StateAwaitingExchange    OnboardingState = "awaiting_exchange"
	StateProcessing          OnboardingState = "processing"
	StateComplete            OnboardingState = "complete"
	StateError               OnboardingState = "error"
)

// OnboardingSession stores the state of an ongoing onboarding process
// Used by portfolio initialization workflow
type OnboardingSession struct {
	TelegramID        int64           `json:"telegram_id"`
	UserID            uuid.UUID       `json:"user_id"`
	StrategyID        *uuid.UUID      `json:"strategy_id,omitempty"` // Pre-created strategy ID from invest flow
	State             OnboardingState `json:"state"`
	Capital           float64         `json:"capital"`
	RiskProfile       string          `json:"risk_profile"` // conservative, moderate, aggressive
	MarketType        string          `json:"market_type"`  // spot, futures
	ExchangeAccountID *uuid.UUID      `json:"exchange_account_id,omitempty"`
	PreferredAssets   []string        `json:"preferred_assets,omitempty"`
	LastMessageID     int             `json:"last_msg_id"` // For updating messages (inline keyboard flow)
	StartedAt         time.Time       `json:"started_at"`
}
