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

	// Position sizing limits
	MaxPositionSizeUSD  float64  `json:"max_position_size_usd"`  // max single position in USD
	MaxTotalExposureUSD float64  `json:"max_total_exposure_usd"` // max total portfolio exposure
	MinPositionSizeUSD  float64  `json:"min_position_size_usd"`  // min position to avoid dust
	MaxLeverageMultiple float64  `json:"max_leverage_multiple"`  // max leverage (1.0 = no leverage)
	AllowedExchanges    []string `json:"allowed_exchanges"`      // whitelist of exchanges
}

// RiskTolerance provides a convenient accessor for risk-related settings
type RiskTolerance struct {
	MaxPositionSizeUSD  float64
	MaxTotalExposureUSD float64
	MaxPortfolioRiskPct float64
	MaxDailyDrawdownPct float64
	MaxLeverage         float64
	MaxPositions        int
}

// GetRiskTolerance returns risk tolerance settings for the user
func (u *User) GetRiskTolerance() RiskTolerance {
	return RiskTolerance{
		MaxPositionSizeUSD:  u.Settings.MaxPositionSizeUSD,
		MaxTotalExposureUSD: u.Settings.MaxTotalExposureUSD,
		MaxPortfolioRiskPct: u.Settings.MaxPortfolioRisk,
		MaxDailyDrawdownPct: u.Settings.MaxDailyDrawdown,
		MaxLeverage:         u.Settings.MaxLeverageMultiple,
		MaxPositions:        u.Settings.MaxPositions,
	}
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

		// Conservative defaults for position sizing
		MaxPositionSizeUSD:  1000.0, // $1000 max per position
		MaxTotalExposureUSD: 5000.0, // $5000 max total
		MinPositionSizeUSD:  10.0,   // $10 minimum
		MaxLeverageMultiple: 1.0,    // No leverage by default
		AllowedExchanges:    []string{"binance", "bybit"},
	}
}
