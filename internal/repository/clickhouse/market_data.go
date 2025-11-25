package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"prometheus/internal/domain/market_data"
)

// Compile-time check
var _ market_data.Repository = (*MarketDataRepository)(nil)

// MarketDataRepository implements market_data.Repository using ClickHouse
type MarketDataRepository struct {
	conn driver.Conn
}

// NewMarketDataRepository creates a new market data repository
func NewMarketDataRepository(conn driver.Conn) *MarketDataRepository {
	return &MarketDataRepository{conn: conn}
}

// InsertOHLCV inserts OHLCV candles in batch
func (r *MarketDataRepository) InsertOHLCV(ctx context.Context, candles []market_data.OHLCV) error {
	if len(candles) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO ohlcv (
			exchange, symbol, timeframe, open_time,
			open, high, low, close, volume, quote_volume, trades
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	for _, candle := range candles {
		err := batch.Append(
			candle.Exchange, candle.Symbol, candle.Timeframe, candle.OpenTime,
			candle.Open, candle.High, candle.Low, candle.Close,
			candle.Volume, candle.QuoteVolume, candle.Trades,
		)
		if err != nil {
			return fmt.Errorf("failed to append candle: %w", err)
		}
	}

	return batch.Send()
}

// GetOHLCV retrieves OHLCV candles with query parameters
func (r *MarketDataRepository) GetOHLCV(ctx context.Context, query market_data.OHLCVQuery) ([]market_data.OHLCV, error) {
	var candles []market_data.OHLCV

	sql := `
		SELECT exchange, symbol, timeframe, open_time, open, high, low, close, volume, quote_volume, trades
		FROM ohlcv
		WHERE symbol = $1 AND timeframe = $2`

	args := []interface{}{query.Symbol, query.Timeframe}

	if query.Exchange != "" {
		sql += ` AND exchange = $3`
		args = append(args, query.Exchange)
	}

	if !query.StartTime.IsZero() {
		sql += fmt.Sprintf(` AND open_time >= $%d`, len(args)+1)
		args = append(args, query.StartTime)
	}

	if !query.EndTime.IsZero() {
		sql += fmt.Sprintf(` AND open_time <= $%d`, len(args)+1)
		args = append(args, query.EndTime)
	}

	sql += ` ORDER BY open_time DESC`

	if query.Limit > 0 {
		sql += fmt.Sprintf(` LIMIT $%d`, len(args)+1)
		args = append(args, query.Limit)
	}

	err := r.conn.Select(ctx, &candles, sql, args...)
	return candles, err
}

// GetLatestOHLCV retrieves latest N candles
func (r *MarketDataRepository) GetLatestOHLCV(ctx context.Context, exchange, symbol, timeframe string, limit int) ([]market_data.OHLCV, error) {
	var candles []market_data.OHLCV

	sql := `
		SELECT exchange, symbol, timeframe, open_time, open, high, low, close, volume, quote_volume, trades
		FROM ohlcv
		WHERE exchange = $1 AND symbol = $2 AND timeframe = $3
		ORDER BY open_time DESC
		LIMIT $4`

	err := r.conn.Select(ctx, &candles, sql, exchange, symbol, timeframe, limit)
	return candles, err
}

// InsertTicker inserts a ticker
func (r *MarketDataRepository) InsertTicker(ctx context.Context, ticker *market_data.Ticker) error {
	query := `
		INSERT INTO tickers (
			exchange, symbol, timestamp, price, bid, ask,
			volume_24h, change_24h, high_24h, low_24h,
			funding_rate, open_interest
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)`

	return r.conn.Exec(ctx, query,
		ticker.Exchange, ticker.Symbol, ticker.Timestamp,
		ticker.Price, ticker.Bid, ticker.Ask,
		ticker.Volume24h, ticker.Change24h, ticker.High24h, ticker.Low24h,
		ticker.FundingRate, ticker.OpenInterest,
	)
}

// GetLatestTicker retrieves the latest ticker for a symbol
func (r *MarketDataRepository) GetLatestTicker(ctx context.Context, exchange, symbol string) (*market_data.Ticker, error) {
	var ticker market_data.Ticker

	query := `
		SELECT * FROM tickers
		WHERE exchange = $1 AND symbol = $2
		ORDER BY timestamp DESC
		LIMIT 1`

	err := r.conn.QueryRow(ctx, query, exchange, symbol).ScanStruct(&ticker)
	if err != nil {
		return nil, err
	}

	return &ticker, nil
}

// InsertOrderBook inserts an order book snapshot
func (r *MarketDataRepository) InsertOrderBook(ctx context.Context, snapshot *market_data.OrderBookSnapshot) error {
	query := `
		INSERT INTO orderbook_snapshots (
			exchange, symbol, timestamp, bids, asks, bid_depth, ask_depth
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)`

	return r.conn.Exec(ctx, query,
		snapshot.Exchange, snapshot.Symbol, snapshot.Timestamp,
		snapshot.Bids, snapshot.Asks, snapshot.BidDepth, snapshot.AskDepth,
	)
}

// GetLatestOrderBook retrieves the latest order book snapshot
func (r *MarketDataRepository) GetLatestOrderBook(ctx context.Context, exchange, symbol string) (*market_data.OrderBookSnapshot, error) {
	var snapshot market_data.OrderBookSnapshot

	query := `
		SELECT * FROM orderbook_snapshots
		WHERE exchange = $1 AND symbol = $2
		ORDER BY timestamp DESC
		LIMIT 1`

	err := r.conn.QueryRow(ctx, query, exchange, symbol).ScanStruct(&snapshot)
	if err != nil {
		return nil, err
	}

	return &snapshot, nil
}

// InsertTrades inserts trades in batch
func (r *MarketDataRepository) InsertTrades(ctx context.Context, trades []market_data.Trade) error {
	if len(trades) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO trades (exchange, symbol, timestamp, trade_id, price, quantity, side, is_buyer)
	`)
	if err != nil {
		return err
	}

	for _, trade := range trades {
		err := batch.Append(
			trade.Exchange, trade.Symbol, trade.Timestamp, trade.TradeID,
			trade.Price, trade.Quantity, trade.Side, trade.IsBuyer,
		)
		if err != nil {
			return err
		}
	}

	return batch.Send()
}

// GetRecentTrades retrieves recent trades
func (r *MarketDataRepository) GetRecentTrades(ctx context.Context, exchange, symbol string, limit int) ([]market_data.Trade, error) {
	var trades []market_data.Trade

	query := `
		SELECT * FROM trades
		WHERE exchange = $1 AND symbol = $2
		ORDER BY timestamp DESC
		LIMIT $3`

	err := r.conn.Select(ctx, &trades, query, exchange, symbol, limit)
	return trades, err
}
