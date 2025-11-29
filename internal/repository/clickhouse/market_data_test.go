package clickhouse

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/market_data"
	"prometheus/internal/testsupport"
)

func TestMarketDataRepository_InsertAndGetOHLCV(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testsupport.LoadDatabaseConfigsFromEnv(t)
	helper := testsupport.NewClickHouseTestHelper(t, cfg.ClickHouse)

	// Create repository
	repo := NewMarketDataRepository(helper.Client().Conn())
	ctx := context.Background()

	// Register cleanup for test data
	helper.RegisterTableCleanup(t, "ohlcv", "exchange IN ('binance', 'bybit', 'okx', 'test_exchange')")

	t.Run("InsertOHLCV_Success", func(t *testing.T) {
		// Use fixtures for clean test data creation
		btcCandle := testsupport.NewOHLCVFixture().
			WithSymbol("BTC/USDT").
			WithPrices(50000, 51000, 49500, 50500).
			WithVolume(100.5, 5025000).
			WithTrades(1500).
			WithBuyPressure(55). // 55% buy pressure
			Build()

		ethCandle := testsupport.NewOHLCVFixture().
			WithSymbol("ETH/USDT").
			WithPrices(3000, 3100, 2950, 3050).
			WithVolume(500, 1500000).
			WithTrades(800).
			WithBuyPressure(56). // 56% buy pressure
			Build()

		candles := []market_data.OHLCV{btcCandle, ethCandle}

		// Test the repository insert method
		err := repo.InsertOHLCV(ctx, candles)
		require.NoError(t, err)

		// Verify data was inserted
		var count uint64
		err = helper.Client().Query(ctx, &count, "SELECT count() FROM ohlcv WHERE exchange = 'binance' AND symbol IN ('BTC/USDT', 'ETH/USDT')")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, uint64(2))
	})

	t.Run("InsertOHLCV_EmptySlice", func(t *testing.T) {
		err := repo.InsertOHLCV(ctx, []market_data.OHLCV{})
		require.NoError(t, err)
	})

	t.Run("GetLatestOHLCV_Success", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Minute)

		// Setup: Create test data using fixtures
		candles := testsupport.NewOHLCVFixture().
			WithExchange("bybit").
			WithSymbol("SOL/USDT").
			WithTimeframe("5m").
			WithOpenTime(baseTime.Add(-10*time.Minute)).
			WithPrices(100, 102, 99, 101).
			WithVolume(1000, 100500).
			WithTrades(250).
			WithBuyPressure(60). // 60% buy pressure
			BuildMany(2)         // Create 2 sequential candles

		// Insert using generic batch helper
		testsupport.CreateBatch(t, helper, testsupport.InsertOHLCV, candles)

		// Test: Get latest 1 candle
		result, err := repo.GetLatestOHLCV(ctx, "bybit", "SOL/USDT", "5m", 1)
		require.NoError(t, err)
		require.Len(t, result, 1)

		// Verify: Should return the most recent candle
		assert.Equal(t, "bybit", result[0].Exchange)
		assert.Equal(t, "SOL/USDT", result[0].Symbol)
		assert.Equal(t, "5m", result[0].Timeframe)
		assert.Equal(t, int64(250), result[0].Trades)

		// Test: Get latest 2 candles
		result, err = repo.GetLatestOHLCV(ctx, "bybit", "SOL/USDT", "5m", 2)
		require.NoError(t, err)
		require.Len(t, result, 2)
	})

	t.Run("GetOHLCV_WithFilters", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Hour)

		// Insert test data across multiple hours
		candles := []market_data.OHLCV{
			{
				Exchange:   "okx",
				Symbol:     "BNB/USDT",
				Timeframe:  "1h",
				MarketType: "spot",
				OpenTime:   baseTime.Add(-3 * time.Hour),
				CloseTime:  baseTime.Add(-2 * time.Hour),
				Open:       300.0,
				High:       310.0,
				Low:        295.0,
				Close:      305.0,
				Volume:     500.0,
			},
			{
				Exchange:   "okx",
				Symbol:     "BNB/USDT",
				Timeframe:  "1h",
				MarketType: "spot",
				OpenTime:   baseTime.Add(-2 * time.Hour),
				CloseTime:  baseTime.Add(-1 * time.Hour),
				Open:       305.0,
				High:       315.0,
				Low:        302.0,
				Close:      310.0,
				Volume:     600.0,
			},
			{
				Exchange:   "okx",
				Symbol:     "BNB/USDT",
				Timeframe:  "1h",
				MarketType: "spot",
				OpenTime:   baseTime.Add(-1 * time.Hour),
				CloseTime:  baseTime,
				Open:       310.0,
				High:       320.0,
				Low:        308.0,
				Close:      315.0,
				Volume:     700.0,
			},
		}

		err := repo.InsertOHLCV(ctx, candles)
		require.NoError(t, err)

		// Query with time range
		query := market_data.OHLCVQuery{
			Exchange:  "okx",
			Symbol:    "BNB/USDT",
			Timeframe: "1h",
			StartTime: baseTime.Add(-2*time.Hour - 1*time.Minute), // Include second candle
			EndTime:   baseTime.Add(-30 * time.Minute),            // Exclude last candle
			Limit:     10,
		}

		result, err := repo.GetOHLCV(ctx, query)
		require.NoError(t, err)
		require.Len(t, result, 1) // Should only return the second candle
		assert.Equal(t, 310.0, result[0].Close)
	})

	t.Run("TakerBuyVolume_Calculation", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Minute)

		candle := market_data.OHLCV{
			Exchange:            "binance",
			Symbol:              "XRP/USDT",
			Timeframe:           "1m",
			MarketType:          "spot",
			OpenTime:            baseTime,
			CloseTime:           baseTime.Add(1 * time.Minute),
			Open:                0.5,
			High:                0.52,
			Low:                 0.49,
			Close:               0.51,
			Volume:              1000000.0,
			QuoteVolume:         500000.0,
			Trades:              5000,
			TakerBuyBaseVolume:  650000.0, // 65% buying pressure
			TakerBuyQuoteVolume: 325000.0,
		}

		err := repo.InsertOHLCV(ctx, []market_data.OHLCV{candle})
		require.NoError(t, err)

		result, err := repo.GetLatestOHLCV(ctx, "binance", "XRP/USDT", "1m", 1)
		require.NoError(t, err)
		require.Len(t, result, 1)

		// Verify taker buy volume fields
		assert.Equal(t, 650000.0, result[0].TakerBuyBaseVolume)
		assert.Equal(t, 325000.0, result[0].TakerBuyQuoteVolume)

		// Calculate buy pressure percentage
		buyPressure := (result[0].TakerBuyBaseVolume / result[0].Volume) * 100
		assert.InDelta(t, 65.0, buyPressure, 0.1) // 65% buy pressure
	})
}

func TestMarketDataRepository_Ticker(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testsupport.LoadDatabaseConfigsFromEnv(t)
	helper := testsupport.NewClickHouseTestHelper(t, cfg.ClickHouse)

	repo := NewMarketDataRepository(helper.Client().Conn())
	ctx := context.Background()

	t.Run("InsertAndGetLatestTicker", func(t *testing.T) {
		ticker := &market_data.Ticker{
			Exchange:     "binance",
			Symbol:       "BTC/USDT",
			Timestamp:    time.Now().Truncate(time.Second),
			Price:        50000.0,
			Bid:          49995.0,
			Ask:          50005.0,
			Volume24h:    10000.0,
			Change24h:    2.5,
			High24h:      51000.0,
			Low24h:       49000.0,
			FundingRate:  0.0001,
			OpenInterest: 1000000.0,
		}

		err := repo.InsertTicker(ctx, ticker)
		require.NoError(t, err)

		result, err := repo.GetLatestTicker(ctx, "binance", "BTC/USDT")
		require.NoError(t, err)
		assert.Equal(t, ticker.Price, result.Price)
		assert.Equal(t, ticker.Volume24h, result.Volume24h)
	})
}

func TestMarketDataRepository_Trades(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testsupport.LoadDatabaseConfigsFromEnv(t)
	helper := testsupport.NewClickHouseTestHelper(t, cfg.ClickHouse)

	repo := NewMarketDataRepository(helper.Client().Conn())
	ctx := context.Background()

	t.Run("InsertAndGetRecentTrades", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Second)

		trades := []market_data.Trade{
			{
				Exchange:  "binance",
				Symbol:    "ETH/USDT",
				Timestamp: baseTime.Add(-2 * time.Second),
				TradeID:   "trade1",
				Price:     3000.0,
				Quantity:  1.5,
				Side:      "buy",
				IsBuyer:   true,
			},
			{
				Exchange:  "binance",
				Symbol:    "ETH/USDT",
				Timestamp: baseTime.Add(-1 * time.Second),
				TradeID:   "trade2",
				Price:     3001.0,
				Quantity:  2.0,
				Side:      "sell",
				IsBuyer:   false,
			},
		}

		err := repo.InsertTrades(ctx, trades)
		require.NoError(t, err)

		result, err := repo.GetRecentTrades(ctx, "binance", "ETH/USDT", 10)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(result), 2)
	})
}

func TestMarketDataRepository_FundingRate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testsupport.LoadDatabaseConfigsFromEnv(t)
	helper := testsupport.NewClickHouseTestHelper(t, cfg.ClickHouse)

	repo := NewMarketDataRepository(helper.Client().Conn())
	ctx := context.Background()

	t.Run("InsertAndGetLatestFundingRate", func(t *testing.T) {
		fundingRate := &market_data.FundingRate{
			Exchange:        "binance",
			Symbol:          "BTC/USDT",
			Timestamp:       time.Now().Truncate(time.Second),
			FundingRate:     0.0001,
			NextFundingTime: time.Now().Add(8 * time.Hour).Truncate(time.Hour),
			MarkPrice:       50000.0,
			IndexPrice:      49995.0,
		}

		err := repo.InsertFundingRate(ctx, fundingRate)
		require.NoError(t, err)

		result, err := repo.GetLatestFundingRate(ctx, "binance", "BTC/USDT")
		require.NoError(t, err)
		assert.Equal(t, fundingRate.FundingRate, result.FundingRate)
		assert.Equal(t, fundingRate.MarkPrice, result.MarkPrice)
	})
}
