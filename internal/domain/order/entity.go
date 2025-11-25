package order

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Order represents a trading order
type Order struct {
	ID                uuid.UUID `db:"id"`
	UserID            uuid.UUID `db:"user_id"`
	TradingPairID     uuid.UUID `db:"trading_pair_id"`
	ExchangeAccountID uuid.UUID `db:"exchange_account_id"`

	// Exchange data
	ExchangeOrderID string `db:"exchange_order_id"`
	Symbol          string `db:"symbol"`
	MarketType      string `db:"market_type"`

	// Order details
	Side   OrderSide   `db:"side"` // buy, sell
	Type   OrderType   `db:"type"` // market, limit, stop_market, stop_limit
	Status OrderStatus `db:"status"`

	// Prices & amounts
	Price        decimal.Decimal `db:"price"`
	Amount       decimal.Decimal `db:"amount"`
	FilledAmount decimal.Decimal `db:"filled_amount"`
	AvgFillPrice decimal.Decimal `db:"avg_fill_price"`

	// Stop orders
	StopPrice decimal.Decimal `db:"stop_price"`

	// Futures specific
	ReduceOnly bool `db:"reduce_only"`

	// Metadata
	AgentID       string     `db:"agent_id"`        // Which agent created
	Reasoning     string     `db:"reasoning"`       // AI reasoning
	ParentOrderID *uuid.UUID `db:"parent_order_id"` // For SL/TP orders

	// Fees
	Fee         decimal.Decimal `db:"fee"`
	FeeCurrency string          `db:"fee_currency"`

	// Timestamps
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	FilledAt  *time.Time `db:"filled_at"`
}

// OrderSide defines buy or sell
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// Valid checks if order side is valid
func (s OrderSide) Valid() bool {
	return s == OrderSideBuy || s == OrderSideSell
}

// String returns string representation
func (s OrderSide) String() string {
	return string(s)
}

// OrderType defines order execution type
type OrderType string

const (
	OrderTypeMarket     OrderType = "market"
	OrderTypeLimit      OrderType = "limit"
	OrderTypeStopMarket OrderType = "stop_market"
	OrderTypeStopLimit  OrderType = "stop_limit"
)

// Valid checks if order type is valid
func (t OrderType) Valid() bool {
	switch t {
	case OrderTypeMarket, OrderTypeLimit, OrderTypeStopMarket, OrderTypeStopLimit:
		return true
	}
	return false
}

// String returns string representation
func (t OrderType) String() string {
	return string(t)
}

// OrderStatus defines order lifecycle status
type OrderStatus string

const (
	OrderStatusPending  OrderStatus = "pending"
	OrderStatusOpen     OrderStatus = "open"
	OrderStatusFilled   OrderStatus = "filled"
	OrderStatusPartial  OrderStatus = "partial"
	OrderStatusCanceled OrderStatus = "canceled"
	OrderStatusRejected OrderStatus = "rejected"
	OrderStatusExpired  OrderStatus = "expired"
)

// Valid checks if order status is valid
func (s OrderStatus) Valid() bool {
	switch s {
	case OrderStatusPending, OrderStatusOpen, OrderStatusFilled,
		OrderStatusPartial, OrderStatusCanceled, OrderStatusRejected, OrderStatusExpired:
		return true
	}
	return false
}

// String returns string representation
func (s OrderStatus) String() string {
	return string(s)
}

// IsTerminal returns true if order is in terminal state
func (s OrderStatus) IsTerminal() bool {
	switch s {
	case OrderStatusFilled, OrderStatusCanceled, OrderStatusRejected, OrderStatusExpired:
		return true
	}
	return false
}

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Order represents a trading order
type Order struct {
	ID                uuid.UUID `db:"id"`
	UserID            uuid.UUID `db:"user_id"`
	TradingPairID     uuid.UUID `db:"trading_pair_id"`
	ExchangeAccountID uuid.UUID `db:"exchange_account_id"`

	// Exchange data
	ExchangeOrderID string `db:"exchange_order_id"`
	Symbol          string `db:"symbol"`
	MarketType      string `db:"market_type"`

	// Order details
	Side   OrderSide   `db:"side"` // buy, sell
	Type   OrderType   `db:"type"` // market, limit, stop_market, stop_limit
	Status OrderStatus `db:"status"`

	// Prices & amounts
	Price        decimal.Decimal `db:"price"`
	Amount       decimal.Decimal `db:"amount"`
	FilledAmount decimal.Decimal `db:"filled_amount"`
	AvgFillPrice decimal.Decimal `db:"avg_fill_price"`

	// Stop orders
	StopPrice decimal.Decimal `db:"stop_price"`

	// Futures specific
	ReduceOnly bool `db:"reduce_only"`

	// Metadata
	AgentID       string     `db:"agent_id"`        // Which agent created
	Reasoning     string     `db:"reasoning"`       // AI reasoning
	ParentOrderID *uuid.UUID `db:"parent_order_id"` // For SL/TP orders

	// Fees
	Fee         decimal.Decimal `db:"fee"`
	FeeCurrency string          `db:"fee_currency"`

	// Timestamps
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	FilledAt  *time.Time `db:"filled_at"`
}

// OrderSide defines buy or sell
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// Valid checks if order side is valid
func (s OrderSide) Valid() bool {
	return s == OrderSideBuy || s == OrderSideSell
}

// String returns string representation
func (s OrderSide) String() string {
	return string(s)
}

// OrderType defines order execution type
type OrderType string

const (
	OrderTypeMarket     OrderType = "market"
	OrderTypeLimit      OrderType = "limit"
	OrderTypeStopMarket OrderType = "stop_market"
	OrderTypeStopLimit  OrderType = "stop_limit"
)

// Valid checks if order type is valid
func (t OrderType) Valid() bool {
	switch t {
	case OrderTypeMarket, OrderTypeLimit, OrderTypeStopMarket, OrderTypeStopLimit:
		return true
	}
	return false
}

// String returns string representation
func (t OrderType) String() string {
	return string(t)
}

// OrderStatus defines order lifecycle status
type OrderStatus string

const (
	OrderStatusPending  OrderStatus = "pending"
	OrderStatusOpen     OrderStatus = "open"
	OrderStatusFilled   OrderStatus = "filled"
	OrderStatusPartial  OrderStatus = "partial"
	OrderStatusCanceled OrderStatus = "canceled"
	OrderStatusRejected OrderStatus = "rejected"
	OrderStatusExpired  OrderStatus = "expired"
)

// Valid checks if order status is valid
func (s OrderStatus) Valid() bool {
	switch s {
	case OrderStatusPending, OrderStatusOpen, OrderStatusFilled,
		OrderStatusPartial, OrderStatusCanceled, OrderStatusRejected, OrderStatusExpired:
		return true
	}
	return false
}

// String returns string representation
func (s OrderStatus) String() string {
	return string(s)
}

// IsTerminal returns true if order is in terminal state
func (s OrderStatus) IsTerminal() bool {
	switch s {
	case OrderStatusFilled, OrderStatusCanceled, OrderStatusRejected, OrderStatusExpired:
		return true
	}
	return false
}
