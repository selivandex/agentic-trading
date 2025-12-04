package websocket

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UserDataStreamer is a generic interface for exchange-specific User Data WebSocket implementations
// Handles real-time order updates, position changes, balance updates, and margin calls
type UserDataStreamer interface {
	// Authentication & ListenKey management
	CreateListenKey(ctx context.Context, apiKey, secret string) (string, time.Time, error)
	RenewListenKey(ctx context.Context, apiKey, secret, listenKey string) error
	DeleteListenKey(ctx context.Context, apiKey, secret, listenKey string) error

	// Connection lifecycle
	Connect(ctx context.Context, listenKey, apiKey, secret string) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	// Status
	IsConnected() bool
	GetStats() UserDataStats
}

// UserDataEventHandler processes events from User Data WebSocket
// All methods should be non-blocking and idempotent
type UserDataEventHandler interface {
	OnOrderUpdate(ctx context.Context, event *OrderUpdateEvent) error
	OnPositionUpdate(ctx context.Context, event *PositionUpdateEvent) error
	OnBalanceUpdate(ctx context.Context, event *BalanceUpdateEvent) error
	OnMarginCall(ctx context.Context, event *MarginCallEvent) error
	OnAccountConfigUpdate(ctx context.Context, event *AccountConfigEvent) error
	OnError(err error)
}

// OrderUpdateEvent represents a real-time order status change
type OrderUpdateEvent struct {
	UserID          uuid.UUID
	AccountID       uuid.UUID
	Exchange        string
	OrderID         string
	ClientOrderID   string
	Symbol          string
	Side            string // BUY, SELL
	PositionSide    string // LONG, SHORT, BOTH
	Type            string // LIMIT, MARKET, STOP, etc.
	Status          string // NEW, PARTIALLY_FILLED, FILLED, CANCELED, REJECTED
	ExecutionType   string // NEW, TRADE, CANCELED, EXPIRED
	OriginalQty     string
	FilledQty       string
	AvgPrice        string
	StopPrice       string
	LastFilledQty   string
	LastFilledPrice string
	Commission      string
	CommissionAsset string
	TradeTime       time.Time
	EventTime       time.Time
}

// PositionUpdateEvent represents a position change
type PositionUpdateEvent struct {
	UserID            uuid.UUID
	AccountID         uuid.UUID
	Exchange          string
	Symbol            string
	Side              string // LONG, SHORT
	Amount            string
	EntryPrice        string
	MarkPrice         string
	UnrealizedPnL     string
	MaintenanceMargin string
	PositionSide      string // LONG, SHORT, BOTH (for hedge mode)
	EventTime         time.Time
}

// BalanceUpdateEvent represents a balance change
type BalanceUpdateEvent struct {
	UserID             uuid.UUID
	AccountID          uuid.UUID
	Exchange           string
	Asset              string
	WalletBalance      string
	CrossWalletBalance string
	AvailableBalance   string
	EventTime          time.Time
	ReasonType         string // DEPOSIT, WITHDRAW, ORDER, FUNDING_FEE, etc.
}

// MarginCallEvent represents a liquidation risk warning (CRITICAL!)
type MarginCallEvent struct {
	UserID             uuid.UUID
	AccountID          uuid.UUID
	Exchange           string
	CrossWalletBalance string
	PositionsAtRisk    []PositionAtRisk
	EventTime          time.Time
}

// PositionAtRisk describes a position that may be liquidated
type PositionAtRisk struct {
	Symbol            string
	Side              string
	Amount            string
	MarginType        string
	UnrealizedPnL     string
	MaintenanceMargin string
	PositionSide      string
}

// AccountConfigEvent represents leverage or margin mode changes
type AccountConfigEvent struct {
	UserID    uuid.UUID
	AccountID uuid.UUID
	Exchange  string
	Symbol    string
	Leverage  int
	EventTime time.Time
}

// UserDataStats tracks User Data WebSocket connection health
type UserDataStats struct {
	AccountID          uuid.UUID
	Exchange           string
	ConnectedSince     time.Time
	MessagesReceived   int64
	OrderUpdates       int64
	PositionUpdates    int64
	BalanceUpdates     int64
	MarginCalls        int64
	ErrorCount         int64
	ReconnectCount     int
	LastError          error
	ListenKeyExpiresAt time.Time
}


