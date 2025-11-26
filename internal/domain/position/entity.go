package position

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Position represents an open trading position
type Position struct {
	ID                uuid.UUID `db:"id"`
	UserID            uuid.UUID `db:"user_id"`
	TradingPairID     uuid.UUID `db:"trading_pair_id"`
	ExchangeAccountID uuid.UUID `db:"exchange_account_id"`

	Symbol     string       `db:"symbol"`
	MarketType string       `db:"market_type"`
	Side       PositionSide `db:"side"` // long, short

	// Size & prices
	Size             decimal.Decimal `db:"size"`
	EntryPrice       decimal.Decimal `db:"entry_price"`
	CurrentPrice     decimal.Decimal `db:"current_price"`
	LiquidationPrice decimal.Decimal `db:"liquidation_price"` // Futures only

	// Leverage (futures)
	Leverage   int    `db:"leverage"`
	MarginMode string `db:"margin_mode"` // cross, isolated

	// PnL
	UnrealizedPnL    decimal.Decimal `db:"unrealized_pnl"`
	UnrealizedPnLPct decimal.Decimal `db:"unrealized_pnl_pct"`
	RealizedPnL      decimal.Decimal `db:"realized_pnl"`

	// Risk management
	StopLossPrice   decimal.Decimal `db:"stop_loss_price"`
	TakeProfitPrice decimal.Decimal `db:"take_profit_price"`
	TrailingStopPct decimal.Decimal `db:"trailing_stop_pct"`

	// Orders
	StopLossOrderID   *uuid.UUID `db:"stop_loss_order_id"`
	TakeProfitOrderID *uuid.UUID `db:"take_profit_order_id"`

	// Metadata
	OpenReasoning string `db:"open_reasoning"`

	Status    PositionStatus `db:"status"`
	OpenedAt  time.Time      `db:"opened_at"`
	ClosedAt  *time.Time     `db:"closed_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

// PositionSide defines long or short
type PositionSide string

const (
	PositionLong  PositionSide = "long"
	PositionShort PositionSide = "short"
)

// Valid checks if position side is valid
func (s PositionSide) Valid() bool {
	return s == PositionLong || s == PositionShort
}

// String returns string representation
func (s PositionSide) String() string {
	return string(s)
}

// PositionStatus defines position lifecycle status
type PositionStatus string

const (
	PositionOpen       PositionStatus = "open"
	PositionClosed     PositionStatus = "closed"
	PositionLiquidated PositionStatus = "liquidated"
)

// Valid checks if position status is valid
func (s PositionStatus) Valid() bool {
	switch s {
	case PositionOpen, PositionClosed, PositionLiquidated:
		return true
	}
	return false
}

// String returns string representation
func (s PositionStatus) String() string {
	return string(s)
}

// IsOpen returns true if position is open
func (s PositionStatus) IsOpen() bool {
	return s == PositionOpen
}
