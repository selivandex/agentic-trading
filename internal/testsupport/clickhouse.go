package testsupport

import (
	"context"
	"fmt"
	"testing"
	"time"

	"prometheus/internal/adapters/clickhouse"
	"prometheus/internal/adapters/config"
	"prometheus/internal/domain/market_data"
)

// ClickHouseTestHelper manages cleanup for ClickHouse integration tests.
type ClickHouseTestHelper struct {
	client *clickhouse.Client
}

// NewClickHouseTestHelper creates a ClickHouse client for tests.
func NewClickHouseTestHelper(t *testing.T, cfg config.ClickHouseConfig) *ClickHouseTestHelper {
	t.Helper()

	client, err := clickhouse.NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to connect to clickhouse: %v", err)
	}

	helper := &ClickHouseTestHelper{client: client}
	t.Cleanup(func() { _ = client.Close() })
	return helper
}

// CreateTempTable creates a temporary table and registers cleanup.
func (h *ClickHouseTestHelper) CreateTempTable(t *testing.T, schema string) string {
	t.Helper()

	table := fmt.Sprintf("tmp_test_%d", time.Now().UnixNano())
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s) ENGINE = MergeTree() ORDER BY tuple()", table, schema)

	if err := h.client.Exec(context.Background(), query); err != nil {
		t.Fatalf("failed to create clickhouse table: %v", err)
	}

	t.Cleanup(func() {
		_ = h.client.Exec(context.Background(), fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	})

	return table
}

// CleanupTable drops the provided table immediately.
func (h *ClickHouseTestHelper) CleanupTable(ctx context.Context, table string) error {
	return h.client.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
}

// TruncateTable removes all data from the table but keeps the structure
func (h *ClickHouseTestHelper) TruncateTable(ctx context.Context, table string) error {
	return h.client.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE IF EXISTS %s", table))
}

// CleanupTableData deletes data matching a filter condition
// Example: CleanupTableData(ctx, "ohlcv", "exchange = 'test_exchange'")
func (h *ClickHouseTestHelper) CleanupTableData(ctx context.Context, table, condition string) error {
	query := fmt.Sprintf("ALTER TABLE %s DELETE WHERE %s", table, condition)
	return h.client.Exec(ctx, query)
}

// RegisterTableCleanup schedules cleanup of specific table data after test completes
// This is useful when working with shared tables that shouldn't be dropped
func (h *ClickHouseTestHelper) RegisterTableCleanup(t *testing.T, table, condition string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Use DELETE for immediate cleanup (ALTER TABLE DELETE is async)
		query := fmt.Sprintf("DELETE FROM %s WHERE %s", table, condition)
		_ = h.client.Exec(ctx, query)
	})
}

// CreateBatch is a generic function to insert test data into ClickHouse tables
// Usage: testsupport.CreateBatch(t, helper, testsupport.InsertOHLCV, candles)
func CreateBatch[T any](t *testing.T, helper *ClickHouseTestHelper, insertQuery string, items []T) {
	t.Helper()

	if len(items) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	batch, err := helper.client.Conn().PrepareBatch(ctx, insertQuery)
	if err != nil {
		t.Fatalf("failed to prepare batch: %v", err)
	}

	for _, item := range items {
		if err := batch.AppendStruct(&item); err != nil {
			t.Fatalf("failed to append item to batch: %v", err)
		}
	}

	if err := batch.Send(); err != nil {
		t.Fatalf("failed to send batch: %v", err)
	}
}

// Predefined insert queries for common tables
const (
	InsertOHLCV = `
		INSERT INTO ohlcv (
			exchange, symbol, timeframe, market_type, open_time, close_time,
			open, high, low, close, volume, quote_volume, trades,
			taker_buy_base_volume, taker_buy_quote_volume
		)
	`

	InsertTickers = `
		INSERT INTO tickers (
			exchange, symbol, timestamp, price, bid, ask,
			volume_24h, change_24h, high_24h, low_24h,
			funding_rate, open_interest
		)
	`

	InsertTrades = `
		INSERT INTO trades (
			exchange, symbol, timestamp, trade_id, price, quantity, side, is_buyer
		)
	`
)

// Client exposes the raw ClickHouse client for queries.
func (h *ClickHouseTestHelper) Client() *clickhouse.Client {
	return h.client
}

// ========================================
// Fixture Builders for ClickHouse Tests
// ========================================

// OHLCVFixture provides builder pattern for creating test OHLCV candles
type OHLCVFixture struct {
	candle market_data.OHLCV
}

// NewOHLCVFixture creates a default OHLCV candle for testing
// Default: BTC/USDT on Binance, spot market, with realistic values
func NewOHLCVFixture() *OHLCVFixture {
	now := time.Now().Truncate(time.Minute)
	return &OHLCVFixture{
		candle: market_data.OHLCV{
			Exchange:            "binance",
			Symbol:              "BTC/USDT",
			Timeframe:           "1h",
			MarketType:          "spot",
			OpenTime:            now,
			CloseTime:           now.Add(1 * time.Hour),
			Open:                50000.0,
			High:                51000.0,
			Low:                 49500.0,
			Close:               50500.0,
			Volume:              100.0,
			QuoteVolume:         5000000.0,
			Trades:              1000,
			TakerBuyBaseVolume:  55.0, // 55% buy pressure
			TakerBuyQuoteVolume: 2750000.0,
			IsClosed:            true,
			EventTime:           now,
		},
	}
}

// WithExchange sets the exchange
func (f *OHLCVFixture) WithExchange(exchange string) *OHLCVFixture {
	f.candle.Exchange = exchange
	return f
}

// WithSymbol sets the symbol
func (f *OHLCVFixture) WithSymbol(symbol string) *OHLCVFixture {
	f.candle.Symbol = symbol
	return f
}

// WithTimeframe sets the timeframe
func (f *OHLCVFixture) WithTimeframe(timeframe string) *OHLCVFixture {
	f.candle.Timeframe = timeframe
	return f
}

// WithMarketType sets the market type
func (f *OHLCVFixture) WithMarketType(marketType string) *OHLCVFixture {
	f.candle.MarketType = marketType
	return f
}

// WithOpenTime sets the open time and calculates close time based on timeframe
func (f *OHLCVFixture) WithOpenTime(t time.Time) *OHLCVFixture {
	f.candle.OpenTime = t
	f.candle.CloseTime = t.Add(parseTimeframeDuration(f.candle.Timeframe))
	return f
}

// WithTimes sets both open and close times explicitly
func (f *OHLCVFixture) WithTimes(openTime, closeTime time.Time) *OHLCVFixture {
	f.candle.OpenTime = openTime
	f.candle.CloseTime = closeTime
	return f
}

// WithPrices sets OHLC prices
func (f *OHLCVFixture) WithPrices(open, high, low, close float64) *OHLCVFixture {
	f.candle.Open = open
	f.candle.High = high
	f.candle.Low = low
	f.candle.Close = close
	return f
}

// WithVolume sets base and quote volumes
func (f *OHLCVFixture) WithVolume(baseVolume, quoteVolume float64) *OHLCVFixture {
	f.candle.Volume = baseVolume
	f.candle.QuoteVolume = quoteVolume
	return f
}

// WithTrades sets the number of trades
func (f *OHLCVFixture) WithTrades(trades uint64) *OHLCVFixture {
	f.candle.Trades = trades
	return f
}

// WithBuyPressure sets taker buy volumes to achieve specific buy pressure percentage
// buyPressurePct should be between 0 and 100 (e.g., 60 for 60% buy pressure)
func (f *OHLCVFixture) WithBuyPressure(buyPressurePct float64) *OHLCVFixture {
	f.candle.TakerBuyBaseVolume = f.candle.Volume * (buyPressurePct / 100.0)
	f.candle.TakerBuyQuoteVolume = f.candle.QuoteVolume * (buyPressurePct / 100.0)
	return f
}

// WithTakerBuyVolumes sets taker buy volumes explicitly
func (f *OHLCVFixture) WithTakerBuyVolumes(baseVolume, quoteVolume float64) *OHLCVFixture {
	f.candle.TakerBuyBaseVolume = baseVolume
	f.candle.TakerBuyQuoteVolume = quoteVolume
	return f
}

// WithIsClosed sets whether the candle is closed/final
func (f *OHLCVFixture) WithIsClosed(isClosed bool) *OHLCVFixture {
	f.candle.IsClosed = isClosed
	return f
}

// WithEventTime sets the exchange event timestamp
func (f *OHLCVFixture) WithEventTime(t time.Time) *OHLCVFixture {
	f.candle.EventTime = t
	return f
}

// Bullish creates a bullish candle (close > open, high buy pressure)
func (f *OHLCVFixture) Bullish() *OHLCVFixture {
	basePrice := f.candle.Open
	f.candle.High = basePrice * 1.03
	f.candle.Low = basePrice * 0.99
	f.candle.Close = basePrice * 1.02
	return f.WithBuyPressure(65.0) // 65% buy pressure
}

// Bearish creates a bearish candle (close < open, low buy pressure)
func (f *OHLCVFixture) Bearish() *OHLCVFixture {
	basePrice := f.candle.Open
	f.candle.High = basePrice * 1.01
	f.candle.Low = basePrice * 0.97
	f.candle.Close = basePrice * 0.98
	return f.WithBuyPressure(35.0) // 35% buy pressure
}

// Build returns the constructed OHLCV candle
func (f *OHLCVFixture) Build() market_data.OHLCV {
	return f.candle
}

// BuildMany creates multiple candles with sequential timestamps
func (f *OHLCVFixture) BuildMany(count int) []market_data.OHLCV {
	candles := make([]market_data.OHLCV, count)
	duration := parseTimeframeDuration(f.candle.Timeframe)

	for i := 0; i < count; i++ {
		candle := f.candle
		candle.OpenTime = f.candle.OpenTime.Add(time.Duration(i) * duration)
		candle.CloseTime = candle.OpenTime.Add(duration)
		candle.EventTime = f.candle.EventTime.Add(time.Duration(i) * duration)
		candles[i] = candle
	}

	return candles
}

// TickerFixture provides builder pattern for creating test tickers
type TickerFixture struct {
	ticker market_data.Ticker
}

// NewTickerFixture creates a default ticker for testing
func NewTickerFixture() *TickerFixture {
	now := time.Now().Truncate(time.Second)
	return &TickerFixture{
		ticker: market_data.Ticker{
			Exchange:           "binance",
			Symbol:             "BTC/USDT",
			MarketType:         "spot",
			Timestamp:          now,
			LastPrice:          50000.0,
			OpenPrice:          49000.0,
			HighPrice:          51000.0,
			LowPrice:           48500.0,
			Volume:             10000.0,
			QuoteVolume:        500000000.0, // 500M quote volume
			PriceChange:        1000.0,      // +1000 USD
			PriceChangePercent: 2.04,        // +2.04%
			WeightedAvgPrice:   49800.0,
			TradeCount:         50000,
			EventTime:          now,
		},
	}
}

// WithExchange sets the exchange
func (f *TickerFixture) WithExchange(exchange string) *TickerFixture {
	f.ticker.Exchange = exchange
	return f
}

// WithSymbol sets the symbol
func (f *TickerFixture) WithSymbol(symbol string) *TickerFixture {
	f.ticker.Symbol = symbol
	return f
}

// WithMarketType sets the market type
func (f *TickerFixture) WithMarketType(marketType string) *TickerFixture {
	f.ticker.MarketType = marketType
	return f
}

// WithTimestamp sets the timestamp
func (f *TickerFixture) WithTimestamp(t time.Time) *TickerFixture {
	f.ticker.Timestamp = t
	f.ticker.EventTime = t
	return f
}

// WithPrice sets the last price and adjusts other price fields proportionally
func (f *TickerFixture) WithPrice(price float64) *TickerFixture {
	f.ticker.LastPrice = price
	f.ticker.OpenPrice = price * 0.98
	f.ticker.HighPrice = price * 1.02
	f.ticker.LowPrice = price * 0.97
	f.ticker.WeightedAvgPrice = price * 0.996
	f.ticker.PriceChange = price - f.ticker.OpenPrice
	f.ticker.PriceChangePercent = (f.ticker.PriceChange / f.ticker.OpenPrice) * 100
	return f
}

// WithPrices sets OHLC prices explicitly
func (f *TickerFixture) WithPrices(last, open, high, low float64) *TickerFixture {
	f.ticker.LastPrice = last
	f.ticker.OpenPrice = open
	f.ticker.HighPrice = high
	f.ticker.LowPrice = low
	f.ticker.PriceChange = last - open
	f.ticker.PriceChangePercent = (f.ticker.PriceChange / open) * 100
	return f
}

// WithVolume sets base and quote volumes
func (f *TickerFixture) WithVolume(baseVolume, quoteVolume float64) *TickerFixture {
	f.ticker.Volume = baseVolume
	f.ticker.QuoteVolume = quoteVolume
	return f
}

// Build returns the constructed ticker
func (f *TickerFixture) Build() market_data.Ticker {
	return f.ticker
}

// TradeFixture provides builder pattern for creating test trades
type TradeFixture struct {
	trade market_data.Trade
}

// NewTradeFixture creates a default trade for testing
func NewTradeFixture() *TradeFixture {
	return &TradeFixture{
		trade: market_data.Trade{
			Exchange:     "binance",
			Symbol:       "BTC/USDT",
			MarketType:   "spot",
			Timestamp:    time.Now().Truncate(time.Millisecond),
			TradeID:      12345,
			AggTradeID:   12345,
			Price:        50000.0,
			Quantity:     0.1,
			FirstTradeID: 12345,
			LastTradeID:  12345,
			IsBuyerMaker: false, // Default to buy trade (taker is buyer)
			EventTime:    time.Now().Truncate(time.Millisecond),
		},
	}
}

// WithExchange sets the exchange
func (f *TradeFixture) WithExchange(exchange string) *TradeFixture {
	f.trade.Exchange = exchange
	return f
}

// WithSymbol sets the symbol
func (f *TradeFixture) WithSymbol(symbol string) *TradeFixture {
	f.trade.Symbol = symbol
	return f
}

// WithMarketType sets the market type
func (f *TradeFixture) WithMarketType(marketType string) *TradeFixture {
	f.trade.MarketType = marketType
	return f
}

// WithTimestamp sets the timestamp
func (f *TradeFixture) WithTimestamp(t time.Time) *TradeFixture {
	f.trade.Timestamp = t
	f.trade.EventTime = t
	return f
}

// WithTradeID sets the trade ID
func (f *TradeFixture) WithTradeID(id int64) *TradeFixture {
	f.trade.TradeID = id
	f.trade.AggTradeID = id
	f.trade.FirstTradeID = id
	f.trade.LastTradeID = id
	return f
}

// WithPrice sets the price
func (f *TradeFixture) WithPrice(price float64) *TradeFixture {
	f.trade.Price = price
	return f
}

// WithQuantity sets the quantity
func (f *TradeFixture) WithQuantity(qty float64) *TradeFixture {
	f.trade.Quantity = qty
	return f
}

// AsBuy makes this a buy trade (taker is buyer)
func (f *TradeFixture) AsBuy() *TradeFixture {
	f.trade.IsBuyerMaker = false // Buyer is taker (aggressor buy)
	return f
}

// AsSell makes this a sell trade (taker is seller)
func (f *TradeFixture) AsSell() *TradeFixture {
	f.trade.IsBuyerMaker = true // Seller is taker (aggressor sell)
	return f
}

// Build returns the constructed trade
func (f *TradeFixture) Build() market_data.Trade {
	return f.trade
}

// BuildMany creates multiple trades with sequential timestamps and IDs
func (f *TradeFixture) BuildMany(count int) []market_data.Trade {
	trades := make([]market_data.Trade, count)

	for i := 0; i < count; i++ {
		trade := f.trade
		trade.Timestamp = f.trade.Timestamp.Add(time.Duration(i) * time.Millisecond)
		trade.EventTime = f.trade.EventTime.Add(time.Duration(i) * time.Millisecond)
		trade.TradeID = f.trade.TradeID + int64(i)
		trade.AggTradeID = f.trade.AggTradeID + int64(i)
		trades[i] = trade
	}

	return trades
}

// MarkPriceFixture provides builder pattern for creating test mark prices
type MarkPriceFixture struct {
	markPrice market_data.MarkPrice
}

// NewMarkPriceFixture creates a default mark price for testing
func NewMarkPriceFixture() *MarkPriceFixture {
	now := time.Now().Truncate(time.Second)
	return &MarkPriceFixture{
		markPrice: market_data.MarkPrice{
			Exchange:             "binance",
			Symbol:               "BTC/USDT",
			MarketType:           "linear_perp",
			Timestamp:            now,
			MarkPrice:            50000.0,
			IndexPrice:           49995.0,
			EstimatedSettlePrice: 50005.0,
			FundingRate:          0.0001,
			NextFundingTime:      now.Add(8 * time.Hour),
			EventTime:            now,
		},
	}
}

// WithExchange sets the exchange
func (f *MarkPriceFixture) WithExchange(exchange string) *MarkPriceFixture {
	f.markPrice.Exchange = exchange
	return f
}

// WithSymbol sets the symbol
func (f *MarkPriceFixture) WithSymbol(symbol string) *MarkPriceFixture {
	f.markPrice.Symbol = symbol
	return f
}

// WithMarketType sets the market type
func (f *MarkPriceFixture) WithMarketType(marketType string) *MarkPriceFixture {
	f.markPrice.MarketType = marketType
	return f
}

// WithTimestamp sets the timestamp
func (f *MarkPriceFixture) WithTimestamp(t time.Time) *MarkPriceFixture {
	f.markPrice.Timestamp = t
	return f
}

// WithMarkPrice sets the mark price
func (f *MarkPriceFixture) WithMarkPrice(price float64) *MarkPriceFixture {
	f.markPrice.MarkPrice = price
	return f
}

// WithIndexPrice sets the index price
func (f *MarkPriceFixture) WithIndexPrice(price float64) *MarkPriceFixture {
	f.markPrice.IndexPrice = price
	return f
}

// WithFundingRate sets the funding rate
func (f *MarkPriceFixture) WithFundingRate(rate float64) *MarkPriceFixture {
	f.markPrice.FundingRate = rate
	return f
}

// Build returns the constructed mark price
func (f *MarkPriceFixture) Build() market_data.MarkPrice {
	return f.markPrice
}

// BuildMany creates multiple mark prices with sequential timestamps
func (f *MarkPriceFixture) BuildMany(count int) []market_data.MarkPrice {
	markPrices := make([]market_data.MarkPrice, count)

	for i := 0; i < count; i++ {
		mp := f.markPrice
		mp.Timestamp = f.markPrice.Timestamp.Add(time.Duration(i) * time.Second)
		mp.EventTime = f.markPrice.EventTime.Add(time.Duration(i) * time.Second)
		markPrices[i] = mp
	}

	return markPrices
}

// OrderBookSnapshotFixture provides builder pattern for creating test orderbook snapshots
type OrderBookSnapshotFixture struct {
	snapshot market_data.OrderBookSnapshot
}

// NewOrderBookSnapshotFixture creates a default orderbook snapshot for testing
func NewOrderBookSnapshotFixture() *OrderBookSnapshotFixture {
	return &OrderBookSnapshotFixture{
		snapshot: market_data.OrderBookSnapshot{
			Exchange:   "binance",
			Symbol:     "BTC/USDT",
			MarketType: "spot",
			Timestamp:  time.Now().Truncate(time.Second),
			Bids:       `[[49990.0, 1.5], [49980.0, 2.0], [49970.0, 3.5]]`,
			Asks:       `[[50010.0, 1.2], [50020.0, 2.5], [50030.0, 3.0]]`,
			BidDepth:   7.0, // Total bid volume
			AskDepth:   6.7, // Total ask volume
			EventTime:  time.Now().Truncate(time.Second),
		},
	}
}

// WithExchange sets the exchange
func (f *OrderBookSnapshotFixture) WithExchange(exchange string) *OrderBookSnapshotFixture {
	f.snapshot.Exchange = exchange
	return f
}

// WithSymbol sets the symbol
func (f *OrderBookSnapshotFixture) WithSymbol(symbol string) *OrderBookSnapshotFixture {
	f.snapshot.Symbol = symbol
	return f
}

// WithMarketType sets the market type
func (f *OrderBookSnapshotFixture) WithMarketType(marketType string) *OrderBookSnapshotFixture {
	f.snapshot.MarketType = marketType
	return f
}

// WithTimestamp sets the timestamp
func (f *OrderBookSnapshotFixture) WithTimestamp(t time.Time) *OrderBookSnapshotFixture {
	f.snapshot.Timestamp = t
	f.snapshot.EventTime = t
	return f
}

// WithBids sets the bids JSON array
func (f *OrderBookSnapshotFixture) WithBids(bids string) *OrderBookSnapshotFixture {
	f.snapshot.Bids = bids
	return f
}

// WithAsks sets the asks JSON array
func (f *OrderBookSnapshotFixture) WithAsks(asks string) *OrderBookSnapshotFixture {
	f.snapshot.Asks = asks
	return f
}

// WithDepth sets the total bid and ask depth
func (f *OrderBookSnapshotFixture) WithDepth(bidDepth, askDepth float64) *OrderBookSnapshotFixture {
	f.snapshot.BidDepth = bidDepth
	f.snapshot.AskDepth = askDepth
	return f
}

// Build returns the constructed orderbook snapshot
func (f *OrderBookSnapshotFixture) Build() market_data.OrderBookSnapshot {
	return f.snapshot
}

// LiquidationFixture provides builder pattern for creating test liquidations
type LiquidationFixture struct {
	liquidation market_data.Liquidation
}

// NewLiquidationFixture creates a default liquidation for testing
func NewLiquidationFixture() *LiquidationFixture {
	now := time.Now().Truncate(time.Second)
	return &LiquidationFixture{
		liquidation: market_data.Liquidation{
			Exchange:   "binance",
			Symbol:     "BTC/USDT",
			MarketType: "linear_perp",
			Timestamp:  now,
			Side:       "LONG",
			OrderType:  "MARKET",
			Price:      49000.0,
			Quantity:   2.5,
			Value:      122500.0, // 49000 * 2.5
			EventTime:  now,
		},
	}
}

// WithExchange sets the exchange
func (f *LiquidationFixture) WithExchange(exchange string) *LiquidationFixture {
	f.liquidation.Exchange = exchange
	return f
}

// WithSymbol sets the symbol
func (f *LiquidationFixture) WithSymbol(symbol string) *LiquidationFixture {
	f.liquidation.Symbol = symbol
	return f
}

// WithMarketType sets the market type
func (f *LiquidationFixture) WithMarketType(marketType string) *LiquidationFixture {
	f.liquidation.MarketType = marketType
	return f
}

// WithTimestamp sets the timestamp
func (f *LiquidationFixture) WithTimestamp(t time.Time) *LiquidationFixture {
	f.liquidation.Timestamp = t
	f.liquidation.EventTime = t
	return f
}

// WithSide sets the side (LONG or SHORT)
func (f *LiquidationFixture) WithSide(side string) *LiquidationFixture {
	f.liquidation.Side = side
	return f
}

// WithPrice sets the price and recalculates value
func (f *LiquidationFixture) WithPrice(price float64) *LiquidationFixture {
	f.liquidation.Price = price
	f.liquidation.Value = price * f.liquidation.Quantity
	return f
}

// WithQuantity sets the quantity and recalculates value
func (f *LiquidationFixture) WithQuantity(quantity float64) *LiquidationFixture {
	f.liquidation.Quantity = quantity
	f.liquidation.Value = f.liquidation.Price * quantity
	return f
}

// AsLongLiquidation makes this a long position liquidation
func (f *LiquidationFixture) AsLongLiquidation() *LiquidationFixture {
	f.liquidation.Side = "LONG"
	return f
}

// AsShortLiquidation makes this a short position liquidation
func (f *LiquidationFixture) AsShortLiquidation() *LiquidationFixture {
	f.liquidation.Side = "SHORT"
	return f
}

// Build returns the constructed liquidation
func (f *LiquidationFixture) Build() market_data.Liquidation {
	return f.liquidation
}

// BuildMany creates multiple liquidations with sequential timestamps
func (f *LiquidationFixture) BuildMany(count int) []market_data.Liquidation {
	liquidations := make([]market_data.Liquidation, count)

	for i := 0; i < count; i++ {
		liq := f.liquidation
		liq.Timestamp = f.liquidation.Timestamp.Add(time.Duration(i) * time.Second)
		liq.EventTime = f.liquidation.EventTime.Add(time.Duration(i) * time.Second)
		liquidations[i] = liq
	}

	return liquidations
}

// OpenInterestFixture provides builder pattern for creating test open interest
type OpenInterestFixture struct {
	openInterest market_data.OpenInterest
}

// NewOpenInterestFixture creates a default open interest for testing
func NewOpenInterestFixture() *OpenInterestFixture {
	return &OpenInterestFixture{
		openInterest: market_data.OpenInterest{
			Exchange:  "binance",
			Symbol:    "BTC/USDT",
			Timestamp: time.Now().Truncate(time.Minute),
			Amount:    1000000.0, // 1M USD worth of open contracts
		},
	}
}

// WithExchange sets the exchange
func (f *OpenInterestFixture) WithExchange(exchange string) *OpenInterestFixture {
	f.openInterest.Exchange = exchange
	return f
}

// WithSymbol sets the symbol
func (f *OpenInterestFixture) WithSymbol(symbol string) *OpenInterestFixture {
	f.openInterest.Symbol = symbol
	return f
}

// WithTimestamp sets the timestamp
func (f *OpenInterestFixture) WithTimestamp(t time.Time) *OpenInterestFixture {
	f.openInterest.Timestamp = t
	return f
}

// WithAmount sets the open interest amount
func (f *OpenInterestFixture) WithAmount(amount float64) *OpenInterestFixture {
	f.openInterest.Amount = amount
	return f
}

// Build returns the constructed open interest
func (f *OpenInterestFixture) Build() market_data.OpenInterest {
	return f.openInterest
}

// ========================================
// Generic Helper Functions
// ========================================

// parseTimeframeDuration converts timeframe string to duration
func parseTimeframeDuration(timeframe string) time.Duration {
	switch timeframe {
	case "1m":
		return 1 * time.Minute
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "1h":
		return 1 * time.Hour
	case "4h":
		return 4 * time.Hour
	case "1d":
		return 24 * time.Hour
	default:
		return 1 * time.Hour
	}
}

// BuilderFunc is a generic function type for building test fixtures
type BuilderFunc[T any] func() T

// BuildMany is a generic helper to create multiple instances using a builder
func BuildMany[T any](builder BuilderFunc[T], count int) []T {
	items := make([]T, count)
	for i := 0; i < count; i++ {
		items[i] = builder()
	}
	return items
}

// BuildManyWith is a generic helper to create multiple instances with custom modifications
func BuildManyWith[T any](base T, count int, modifier func(T, int) T) []T {
	items := make([]T, count)
	for i := 0; i < count; i++ {
		items[i] = modifier(base, i)
	}
	return items
}

// OHLCVSequence creates a sequence of candles with incrementing time
func OHLCVSequence(base market_data.OHLCV, count int, timeframe string) []market_data.OHLCV {
	duration := parseTimeframeDuration(timeframe)
	return BuildManyWith(base, count, func(candle market_data.OHLCV, i int) market_data.OHLCV {
		candle.OpenTime = base.OpenTime.Add(time.Duration(i) * duration)
		candle.CloseTime = candle.OpenTime.Add(duration)
		return candle
	})
}

// TradeSequence creates a sequence of trades with incrementing time and IDs
func TradeSequence(base market_data.Trade, count int) []market_data.Trade {
	return BuildManyWith(base, count, func(trade market_data.Trade, i int) market_data.Trade {
		trade.Timestamp = base.Timestamp.Add(time.Duration(i) * time.Millisecond)
		trade.EventTime = base.EventTime.Add(time.Duration(i) * time.Millisecond)
		trade.TradeID = base.TradeID + int64(i)
		trade.AggTradeID = base.AggTradeID + int64(i)
		return trade
	})
}
