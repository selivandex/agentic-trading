package exchanges

import (
	"time"

	"github.com/shopspring/decimal"
)

// MarketType defines supported exchange market segments.
type MarketType string

const (
	MarketTypeSpot         MarketType = "spot"
	MarketTypeLinearPerp   MarketType = "linear_perp"
	MarketTypeInversePerp  MarketType = "inverse_perp"
	MarketTypeDeliveryFut  MarketType = "delivery_futures"
	MarketTypeUnknown      MarketType = "unknown"
)

// OrderSide defines buy or sell direction.
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// PositionSide helps differentiate hedged positions.
type PositionSide string

const (
	PositionSideLong  PositionSide = "long"
	PositionSideShort PositionSide = "short"
	PositionSideBoth  PositionSide = "both"
)

// OrderType defines supported order execution types.
type OrderType string

const (
	OrderTypeMarket     OrderType = "market"
	OrderTypeLimit      OrderType = "limit"
	OrderTypeStopMarket OrderType = "stop_market"
	OrderTypeStopLimit  OrderType = "stop_limit"
)

// OrderStatus enumerates exchange level order lifecycle.
type OrderStatus string

const (
	OrderStatusNew       OrderStatus = "new"
	OrderStatusOpen      OrderStatus = "open"
	OrderStatusPartial   OrderStatus = "partial"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusCanceled  OrderStatus = "canceled"
	OrderStatusRejected  OrderStatus = "rejected"
	OrderStatusExpired   OrderStatus = "expired"
	OrderStatusUnknown   OrderStatus = "unknown"
)

// TimeInForce enumerates supported order time policies.
type TimeInForce string

const (
	TimeInForceGTC TimeInForce = "GTC"
	TimeInForceIOC TimeInForce = "IOC"
	TimeInForceFOK TimeInForce = "FOK"
)

// MarginMode defines futures margin configuration.
type MarginMode string

const (
	MarginCross    MarginMode = "cross"
	MarginIsolated MarginMode = "isolated"
)

// OrderRequest is the unified payload for order placement.
type OrderRequest struct {
	Symbol        string
	Market        MarketType
	Side          OrderSide
	Type          OrderType
	Quantity      decimal.Decimal
	Price         decimal.Decimal
	StopPrice     decimal.Decimal
	TimeInForce   TimeInForce
	ReduceOnly    bool
	MarginMode    MarginMode
	PositionSide  PositionSide
	ClientOrderID string
	Tag           string
	Extra         map[string]string
}

// Order represents a normalized exchange order.
type Order struct {
	ID            string
	ClientOrderID string
	Symbol        string
	Market        MarketType
	Type          OrderType
	Side          OrderSide
	Status        OrderStatus
	Price         decimal.Decimal
	StopPrice     decimal.Decimal
	Quantity      decimal.Decimal
	Filled        decimal.Decimal
	AvgFillPrice  decimal.Decimal
	ReduceOnly    bool
	TimeInForce   TimeInForce
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// OrderBookEntry represents a single level on the order book.
type OrderBookEntry struct {
	Price  decimal.Decimal
	Amount decimal.Decimal
}

// OrderBook captures aggregated bid/ask data.
type OrderBook struct {
	Symbol    string
	Bids      []OrderBookEntry
	Asks      []OrderBookEntry
	Timestamp time.Time
}

// OHLCV candle.
type OHLCV struct {
	Symbol    string
	Timeframe string
	OpenTime  time.Time
	CloseTime time.Time
	Open      decimal.Decimal
	High      decimal.Decimal
	Low       decimal.Decimal
	Close     decimal.Decimal
	Volume    decimal.Decimal
}

// Trade describes a recent trade print.
type Trade struct {
	ID        string
	Symbol    string
	Price     decimal.Decimal
	Amount    decimal.Decimal
	Side      OrderSide
	Timestamp time.Time
}

// Ticker contains 24h stats for a symbol.
type Ticker struct {
	Symbol       string
	LastPrice    decimal.Decimal
	BidPrice     decimal.Decimal
	AskPrice     decimal.Decimal
	High24h      decimal.Decimal
	Low24h       decimal.Decimal
	VolumeBase   decimal.Decimal
	VolumeQuote  decimal.Decimal
	Change24hPct decimal.Decimal
	Timestamp    time.Time
}

// FundingRate contains current funding information.
type FundingRate struct {
	Symbol    string
	Rate      decimal.Decimal
	NextTime  time.Time
	Timestamp time.Time
}

// OpenInterest captures OI snapshot.
type OpenInterest struct {
	Symbol    string
	Amount    decimal.Decimal
	Timestamp time.Time
}

// Balance describes wallet balances.
type Balance struct {
	Total     decimal.Decimal
	Available decimal.Decimal
	Currency  string
	Details   []BalanceDetail
}

// BalanceDetail holds per-asset balance.
type BalanceDetail struct {
	Currency  string
	Total     decimal.Decimal
	Available decimal.Decimal
	Borrowed  decimal.Decimal
}

// Position represents a futures/derivatives position.
type Position struct {
	Symbol           string
	Market           MarketType
	Side             PositionSide
	Size             decimal.Decimal
	EntryPrice       decimal.Decimal
	MarkPrice        decimal.Decimal
	LiquidationPrice decimal.Decimal
	MarginMode       MarginMode
	Leverage         decimal.Decimal
	UnrealizedPnL    decimal.Decimal
	RealizedPnL      decimal.Decimal
	UpdatedAt        time.Time
}

// BracketLeg defines a helper for bracket order legs.
type BracketLeg struct {
	Amount      decimal.Decimal
	Price       decimal.Decimal
	Type        OrderType
	TimeInForce TimeInForce
	Tag         string
}

// BracketOrderRequest orchestrates entry + exit orders.
type BracketOrderRequest struct {
	Entry      OrderRequest
	StopLoss   *BracketLeg
	TakeProfit []BracketLeg
}

// BracketOrderResponse summarizes placed orders.
type BracketOrderResponse struct {
	Entry      *Order
	StopLoss   *Order
	TakeProfit []*Order
}

// LadderStep represents one rung in a ladder order.
type LadderStep struct {
	Price  decimal.Decimal
	Amount decimal.Decimal
	Tag    string
}

// LadderOrderRequest breaks a large order into steps.
type LadderOrderRequest struct {
	Template OrderRequest
	Steps    []LadderStep
}

// LadderOrderResponse lists all created ladder orders.
type LadderOrderResponse struct {
	Orders []*Order
}

