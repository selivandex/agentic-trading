package trading_pair

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TradingPair represents a configured trading pair for a user
type TradingPair struct {
	ID                uuid.UUID       `db:"id"`
	UserID            uuid.UUID       `db:"user_id"`
	ExchangeAccountID uuid.UUID       `db:"exchange_account_id"`
	Symbol            string          `db:"symbol"`      // BTC/USDT
	MarketType        MarketType      `db:"market_type"` // spot, futures

	// Budget & Risk
	Budget            decimal.Decimal `db:"budget"`             // USDT allocated
	MaxPositionSize   decimal.Decimal `db:"max_position_size"`
	MaxLeverage       int             `db:"max_leverage"`      // For futures
	StopLossPercent   decimal.Decimal `db:"stop_loss_percent"`
	TakeProfitPercent decimal.Decimal `db:"take_profit_percent"`

	// Strategy
	AIProvider   string       `db:"ai_provider"`
	StrategyMode StrategyMode `db:"strategy_mode"` // auto, semi_auto, signals
	Timeframes   []string     `db:"timeframes"`    // ["1h", "4h", "1d"]

	// State
	IsActive     bool      `db:"is_active"`
	IsPaused     bool      `db:"is_paused"`
	PausedReason string    `db:"paused_reason"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// MarketType defines market types
type MarketType string

const (
	MarketSpot    MarketType = "spot"
	MarketFutures MarketType = "futures"
)

// Valid checks if market type is valid
func (m MarketType) Valid() bool {
	return m == MarketSpot || m == MarketFutures
}

// String returns string representation
func (m MarketType) String() string {
	return string(m)
}

// StrategyMode defines trading automation level
type StrategyMode string

const (
	StrategyAuto     StrategyMode = "auto"       // Full automation
	StrategySemiAuto StrategyMode = "semi_auto"  // Needs confirmation
	StrategySignals  StrategyMode = "signals"    // Signals only, no execution
)

// Valid checks if strategy mode is valid
func (s StrategyMode) Valid() bool {
	switch s {
	case StrategyAuto, StrategySemiAuto, StrategySignals:
		return true
	}
	return false
}

// String returns string representation
func (s StrategyMode) String() string {
	return string(s)
}
