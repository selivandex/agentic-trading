package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"prometheus/internal/domain/market_data"
	"prometheus/pkg/errors"
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
			exchange, symbol, timeframe, market_type, open_time, close_time,
			open, high, low, close, volume, quote_volume, trades,
			taker_buy_base_volume, taker_buy_quote_volume, is_closed, event_time
		)
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare batch")
	}

	for _, candle := range candles {
		err := batch.Append(
			candle.Exchange, candle.Symbol, candle.Timeframe, candle.MarketType,
			candle.OpenTime, candle.CloseTime,
			candle.Open, candle.High, candle.Low, candle.Close,
			candle.Volume, candle.QuoteVolume, candle.Trades,
			candle.TakerBuyBaseVolume, candle.TakerBuyQuoteVolume,
			candle.IsClosed, candle.EventTime,
		)
		if err != nil {
			return errors.Wrap(err, "failed to append candle")
		}
	}

	return batch.Send()
}

// GetOHLCV retrieves OHLCV candles with query parameters
func (r *MarketDataRepository) GetOHLCV(ctx context.Context, query market_data.OHLCVQuery) ([]market_data.OHLCV, error) {
	var candles []market_data.OHLCV

	sql := `
		SELECT exchange, symbol, timeframe, market_type, open_time, close_time,
		       open, high, low, close, volume, quote_volume, trades,
		       taker_buy_base_volume, taker_buy_quote_volume, is_closed, event_time
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
		SELECT exchange, symbol, timeframe, market_type, open_time, close_time,
		       open, high, low, close, volume, quote_volume, trades,
		       taker_buy_base_volume, taker_buy_quote_volume, is_closed, event_time
		FROM ohlcv
		WHERE exchange = $1 AND symbol = $2 AND timeframe = $3
		ORDER BY open_time DESC
		LIMIT $4`

	err := r.conn.Select(ctx, &candles, sql, exchange, symbol, timeframe, limit)
	return candles, err
}

// InsertMarkPrice inserts mark price data in batch (derivatives only)
func (r *MarketDataRepository) InsertMarkPrice(ctx context.Context, markPrices []market_data.MarkPrice) error {
	if len(markPrices) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO mark_price (
			exchange, symbol, market_type, timestamp, mark_price, index_price,
			estimated_settle_price, funding_rate, next_funding_time, event_time
		)
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare mark_price batch")
	}

	for _, mp := range markPrices {
		err := batch.Append(
			mp.Exchange, mp.Symbol, mp.MarketType, mp.Timestamp,
			mp.MarkPrice, mp.IndexPrice, mp.EstimatedSettlePrice,
			mp.FundingRate, mp.NextFundingTime, mp.EventTime,
		)
		if err != nil {
			return errors.Wrap(err, "failed to append mark_price")
		}
	}

	return batch.Send()
}

// GetLatestMarkPrice retrieves the latest mark price for a symbol
func (r *MarketDataRepository) GetLatestMarkPrice(ctx context.Context, exchange, symbol string) (*market_data.MarkPrice, error) {
	var mp market_data.MarkPrice

	query := `
		SELECT exchange, symbol, market_type, timestamp, mark_price, index_price,
		       estimated_settle_price, funding_rate, next_funding_time, event_time
		FROM mark_price
		WHERE exchange = $1 AND symbol = $2
		ORDER BY timestamp DESC
		LIMIT 1`

	err := r.conn.QueryRow(ctx, query, exchange, symbol).ScanStruct(&mp)
	if err != nil {
		return nil, err
	}

	return &mp, nil
}

// GetMarkPriceHistory retrieves historical mark price data
func (r *MarketDataRepository) GetMarkPriceHistory(ctx context.Context, exchange, symbol string, limit int) ([]market_data.MarkPrice, error) {
	var markPrices []market_data.MarkPrice

	query := `
		SELECT exchange, symbol, market_type, timestamp, mark_price, index_price,
		       estimated_settle_price, funding_rate, next_funding_time, event_time
		FROM mark_price
		WHERE exchange = $1 AND symbol = $2
		ORDER BY timestamp DESC
		LIMIT $3`

	err := r.conn.Select(ctx, &markPrices, query, exchange, symbol, limit)
	return markPrices, err
}

// InsertTicker inserts tickers in batch (24hr statistics)
func (r *MarketDataRepository) InsertTicker(ctx context.Context, tickers []market_data.Ticker) error {
	if len(tickers) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO tickers (
			exchange, symbol, market_type, timestamp, last_price, open_price,
			high_price, low_price, volume, quote_volume, price_change,
			price_change_percent, weighted_avg_price, trade_count, event_time
		)
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare tickers batch")
	}

	for _, ticker := range tickers {
		err := batch.Append(
			ticker.Exchange, ticker.Symbol, ticker.MarketType, ticker.Timestamp,
			ticker.LastPrice, ticker.OpenPrice, ticker.HighPrice, ticker.LowPrice,
			ticker.Volume, ticker.QuoteVolume, ticker.PriceChange,
			ticker.PriceChangePercent, ticker.WeightedAvgPrice, ticker.TradeCount,
			ticker.EventTime,
		)
		if err != nil {
			return errors.Wrap(err, "failed to append ticker")
		}
	}

	return batch.Send()
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

// InsertOrderBook inserts order book snapshots in batch
func (r *MarketDataRepository) InsertOrderBook(ctx context.Context, snapshots []market_data.OrderBookSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO orderbook_snapshots (
			exchange, symbol, market_type, timestamp, bids, asks,
			bid_depth, ask_depth, event_time
		)
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare orderbook batch")
	}

	for _, snapshot := range snapshots {
		err := batch.Append(
			snapshot.Exchange, snapshot.Symbol, snapshot.MarketType, snapshot.Timestamp,
			snapshot.Bids, snapshot.Asks, snapshot.BidDepth, snapshot.AskDepth,
			snapshot.EventTime,
		)
		if err != nil {
			return errors.Wrap(err, "failed to append orderbook")
		}
	}

	return batch.Send()
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

// InsertTrades inserts aggregated trades in batch
func (r *MarketDataRepository) InsertTrades(ctx context.Context, trades []market_data.Trade) error {
	if len(trades) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO trades (
			exchange, symbol, market_type, timestamp, trade_id, agg_trade_id,
			price, quantity, first_trade_id, last_trade_id, is_buyer_maker, event_time
		)
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare trades batch")
	}

	for _, trade := range trades {
		err := batch.Append(
			trade.Exchange, trade.Symbol, trade.MarketType, trade.Timestamp,
			trade.TradeID, trade.AggTradeID, trade.Price, trade.Quantity,
			trade.FirstTradeID, trade.LastTradeID, trade.IsBuyerMaker, trade.EventTime,
		)
		if err != nil {
			return errors.Wrap(err, "failed to append trade")
		}
	}

	return batch.Send()
}

// InsertLiquidations inserts liquidations in batch (derivatives only)
func (r *MarketDataRepository) InsertLiquidations(ctx context.Context, liquidations []market_data.Liquidation) error {
	if len(liquidations) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO liquidations (
			exchange, symbol, market_type, timestamp, side, order_type,
			price, quantity, value, event_time
		)
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare liquidations batch")
	}

	for _, liq := range liquidations {
		err := batch.Append(
			liq.Exchange, liq.Symbol, liq.MarketType, liq.Timestamp,
			liq.Side, liq.OrderType, liq.Price, liq.Quantity, liq.Value, liq.EventTime,
		)
		if err != nil {
			return errors.Wrap(err, "failed to append liquidation")
		}
	}

	return batch.Send()
}

// GetRecentLiquidations retrieves recent liquidations
func (r *MarketDataRepository) GetRecentLiquidations(ctx context.Context, exchange, symbol string, limit int) ([]market_data.Liquidation, error) {
	var liquidations []market_data.Liquidation

	query := `
		SELECT exchange, symbol, market_type, timestamp, side, order_type,
		       price, quantity, value, event_time
		FROM liquidations
		WHERE exchange = $1 AND symbol = $2
		ORDER BY timestamp DESC
		LIMIT $3`

	err := r.conn.Select(ctx, &liquidations, query, exchange, symbol, limit)
	return liquidations, err
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

// InsertOpenInterest inserts an open interest snapshot
func (r *MarketDataRepository) InsertOpenInterest(ctx context.Context, oi *market_data.OpenInterest) error {
	query := `
		INSERT INTO open_interest (
			exchange, symbol, timestamp, amount
		) VALUES (
			$1, $2, $3, $4
		)`

	return r.conn.Exec(ctx, query,
		oi.Exchange, oi.Symbol, oi.Timestamp, oi.Amount,
	)
}

// GetLatestOpenInterest retrieves the latest open interest for a symbol
func (r *MarketDataRepository) GetLatestOpenInterest(ctx context.Context, exchange, symbol string) (*market_data.OpenInterest, error) {
	var oi market_data.OpenInterest

	query := `
		SELECT * FROM open_interest
		WHERE exchange = $1 AND symbol = $2
		ORDER BY timestamp DESC
		LIMIT 1`

	err := r.conn.QueryRow(ctx, query, exchange, symbol).ScanStruct(&oi)
	if err != nil {
		return nil, err
	}

	return &oi, nil
}
