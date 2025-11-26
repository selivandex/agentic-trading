package user

import (
	"time"

	"github.com/google/uuid"
)

// User represents a system user (Telegram user)
type User struct {
	ID               uuid.UUID `db:"id"`
	TelegramID       int64     `db:"telegram_id"`
	TelegramUsername string    `db:"telegram_username"`
	FirstName        string    `db:"first_name"`
	LastName         string    `db:"last_name"`
	LanguageCode     string    `db:"language_code"`
	IsActive         bool      `db:"is_active"`
	IsPremium        bool      `db:"is_premium"`
	Settings         Settings  `db:"settings"` // sqlx handles JSONB automatically
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// Settings contains user preferences and risk parameters
// This struct will be stored as JSONB in PostgreSQL
type Settings struct {
	DefaultAIProvider  string  `json:"default_ai_provider"` // claude, openai, etc.
	DefaultAIModel     string  `json:"default_ai_model"`    // claude-sonnet-4, gpt-4, etc.
	RiskLevel          string  `json:"risk_level"`          // conservative, moderate, aggressive
	MaxPositions       int     `json:"max_positions"`
	MaxPortfolioRisk   float64 `json:"max_portfolio_risk"`   // percentage
	MaxDailyDrawdown   float64 `json:"max_daily_drawdown"`   // percentage, circuit breaker
	MaxConsecutiveLoss int     `json:"max_consecutive_loss"` // circuit breaker trigger
	NotificationsOn    bool    `json:"notifications_on"`
	DailyReportTime    string  `json:"daily_report_time"` // HH:MM
	Timezone           string  `json:"timezone"`
	CircuitBreakerOn   bool    `json:"circuit_breaker_on"`
}

// DefaultSettings returns default user settings
func DefaultSettings() Settings {
	return Settings{
		DefaultAIProvider:  "claude",
		DefaultAIModel:     "claude-sonnet-4",
		RiskLevel:          "moderate",
		MaxPositions:       3,
		MaxPortfolioRisk:   10.0, // 10% max portfolio risk
		MaxDailyDrawdown:   5.0,  // 5% daily drawdown limit
		MaxConsecutiveLoss: 3,
		NotificationsOn:    true,
		DailyReportTime:    "09:00",
		Timezone:           "UTC",
		CircuitBreakerOn:   true,
	}
}
