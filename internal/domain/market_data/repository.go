package market_data

import (
	"context"
)

// Repository defines the interface for market data access (ClickHouse)
type Repository interface {
	// OHLCV operations
	InsertOHLCV(ctx context.Context, candles []OHLCV) error
	GetOHLCV(ctx context.Context, query OHLCVQuery) ([]OHLCV, error)
	GetLatestOHLCV(ctx context.Context, exchange, symbol, timeframe string, limit int) ([]OHLCV, error)

	// Mark Price operations (derivatives only)
	InsertMarkPrice(ctx context.Context, markPrices []MarkPrice) error
	GetLatestMarkPrice(ctx context.Context, exchange, symbol string) (*MarkPrice, error)
	GetMarkPriceHistory(ctx context.Context, exchange, symbol string, limit int) ([]MarkPrice, error)

	// Ticker operations (24hr statistics)
	InsertTicker(ctx context.Context, tickers []Ticker) error
	GetLatestTicker(ctx context.Context, exchange, symbol string) (*Ticker, error)

	// Order book operations
	InsertOrderBook(ctx context.Context, snapshots []OrderBookSnapshot) error
	GetLatestOrderBook(ctx context.Context, exchange, symbol string) (*OrderBookSnapshot, error)

	// Trades operations (aggregated trades)
	InsertTrades(ctx context.Context, trades []Trade) error
	GetRecentTrades(ctx context.Context, exchange, symbol string, limit int) ([]Trade, error)

	// Liquidation operations (derivatives only)
	InsertLiquidations(ctx context.Context, liquidations []Liquidation) error
	GetRecentLiquidations(ctx context.Context, exchange, symbol string, limit int) ([]Liquidation, error)

	// Funding rate operations (deprecated, use MarkPrice)
	InsertFundingRate(ctx context.Context, fundingRate *FundingRate) error
	GetLatestFundingRate(ctx context.Context, exchange, symbol string) (*FundingRate, error)

	// Open interest operations (REST API only, not WebSocket)
	InsertOpenInterest(ctx context.Context, oi *OpenInterest) error
	GetLatestOpenInterest(ctx context.Context, exchange, symbol string) (*OpenInterest, error)
}
