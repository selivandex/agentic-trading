package websocket

import (
	"context"
	"time"
)

// StreamType defines the type of WebSocket stream
type StreamType string
type Interval string

const (
	StreamTypeKline       StreamType = "kline"
	StreamTypeTicker      StreamType = "ticker"
	StreamTypeDepth       StreamType = "depth"
	StreamTypeTrade       StreamType = "trade"
	StreamTypeFunding     StreamType = "funding"
	StreamTypeMarkPrice   StreamType = "markPrice"
	StreamTypeLiquidation StreamType = "liquidation"
)

const (
	Interval1m  Interval = "1m"
	Interval3m  Interval = "3m"
	Interval5m  Interval = "5m"
	Interval15m Interval = "15m"
	Interval30m Interval = "30m"
	Interval1h  Interval = "1h"
	Interval2h  Interval = "2h"
	Interval4h  Interval = "4h"
	Interval6h  Interval = "6h"
	Interval8h  Interval = "8h"
	Interval12h Interval = "12h"
	Interval1d  Interval = "1d"
	Interval3d  Interval = "3d"
	Interval1w  Interval = "1w"
	Interval1M  Interval = "1M"
)

// MarketType defines the market type (spot or futures)
type MarketType string

const (
	MarketTypeSpot    MarketType = "spot"
	MarketTypeFutures MarketType = "futures"
)

// StreamConfig defines configuration for a single WebSocket stream
type StreamConfig struct {
	Type       StreamType
	Symbol     string
	Interval   Interval   // For kline streams (e.g., "1m", "5m", "1h")
	MarketType MarketType // spot or futures
}

// ConnectionConfig defines configuration for WebSocket connection
type ConnectionConfig struct {
	Streams          []StreamConfig
	ReconnectBackoff time.Duration
	MaxReconnects    int
	PingInterval     time.Duration
	ReadBufferSize   int
	WriteBufferSize  int
}

// KlineEvent represents a kline/candlestick event from any exchange
type KlineEvent struct {
	Exchange    string
	Symbol      string
	MarketType  string // "spot" or "futures"
	Interval    string
	OpenTime    time.Time
	CloseTime   time.Time
	Open        string
	High        string
	Low         string
	Close       string
	Volume      string
	QuoteVolume string
	TradeCount  int64
	IsFinal     bool // Whether the candle is closed
	EventTime   time.Time
}

// TickerEvent represents a 24hr ticker statistics event
type TickerEvent struct {
	Exchange           string
	Symbol             string
	MarketType         string // "spot" or "futures"
	PriceChange        string
	PriceChangePercent string
	WeightedAvgPrice   string
	LastPrice          string
	LastQty            string
	OpenPrice          string
	HighPrice          string
	LowPrice           string
	Volume             string
	QuoteVolume        string
	OpenTime           time.Time
	CloseTime          time.Time
	FirstTradeID       int64
	LastTradeID        int64
	TradeCount         int64
	EventTime          time.Time
}

// DepthEvent represents order book depth event
type DepthEvent struct {
	Exchange     string
	Symbol       string
	MarketType   string // "spot" or "futures"
	Bids         []PriceLevel
	Asks         []PriceLevel
	LastUpdateID int64
	EventTime    time.Time
}

// PriceLevel represents a single level in the order book
type PriceLevel struct {
	Price    string
	Quantity string
}

// TradeEvent represents a single trade event
type TradeEvent struct {
	Exchange      string
	Symbol        string
	MarketType    string // "spot" or "futures"
	TradeID       int64
	Price         string
	Quantity      string
	BuyerOrderID  int64
	SellerOrderID int64
	TradeTime     time.Time
	IsBuyerMaker  bool
	EventTime     time.Time
}

// FundingRateEvent represents funding rate event for perpetual contracts
type FundingRateEvent struct {
	Exchange        string
	Symbol          string
	FundingRate     string
	FundingTime     time.Time
	NextFundingTime time.Time
	EventTime       time.Time
}

// MarkPriceEvent represents mark price event for derivatives
type MarkPriceEvent struct {
	Exchange             string
	Symbol               string
	MarkPrice            string
	IndexPrice           string
	EstimatedSettlePrice string
	FundingRate          string
	NextFundingTime      time.Time
	EventTime            time.Time
}

// LiquidationEvent represents a liquidation event (futures only)
type LiquidationEvent struct {
	Exchange   string
	Symbol     string
	MarketType string // "futures", "linear_perp", "inverse_perp"
	Side       string // "long", "short"
	OrderType  string // "LIMIT", "MARKET"
	Price      string
	Quantity   string
	Value      string // price * quantity in USD
	EventTime  time.Time
}

// Client defines the interface for exchange WebSocket clients
type Client interface {
	// Connect establishes WebSocket connection(s) based on config
	Connect(ctx context.Context, config ConnectionConfig) error

	// Start begins receiving events and publishing them to handlers
	Start(ctx context.Context) error

	// Stop gracefully closes all connections
	Stop(ctx context.Context) error

	// IsConnected returns true if client is connected
	IsConnected() bool

	// GetStats returns connection statistics
	GetStats() Stats
}

// Stats represents WebSocket client statistics
type Stats struct {
	ConnectedSince   time.Time
	ReconnectCount   int
	MessagesReceived int64
	MessagesSent     int64
	ErrorCount       int64
	LastError        error
	ActiveStreams    int
}

// EventHandler is called when events are received from WebSocket
type EventHandler interface {
	OnKline(event *KlineEvent) error
	OnTicker(event *TickerEvent) error
	OnDepth(event *DepthEvent) error
	OnTrade(event *TradeEvent) error
	OnFundingRate(event *FundingRateEvent) error
	OnMarkPrice(event *MarkPriceEvent) error
	OnLiquidation(event *LiquidationEvent) error
	OnError(err error)
}
