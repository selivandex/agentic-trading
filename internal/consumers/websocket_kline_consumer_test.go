package consumers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/market_data"
)

func TestWebSocketKlineConsumer_DeduplicateBatch(t *testing.T) {
	consumer := &WebSocketKlineConsumer{}

	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		input          []*market_data.OHLCV
		expectedCount  int
		expectedSymbol string
		expectedClose  float64 // Expected close price (from latest event)
		description    string
	}{
		{
			name: "no_duplicates",
			input: []*market_data.OHLCV{
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "1m",
					OpenTime:  baseTime,
					Close:     50000.0,
					EventTime: baseTime.Add(10 * time.Second),
				},
				{
					Exchange:  "binance",
					Symbol:    "ETHUSDT",
					Timeframe: "1m",
					OpenTime:  baseTime,
					Close:     3000.0,
					EventTime: baseTime.Add(10 * time.Second),
				},
			},
			expectedCount: 2,
			description:   "Different symbols - no deduplication",
		},
		{
			name: "same_symbol_different_intervals",
			input: []*market_data.OHLCV{
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "1m",
					OpenTime:  baseTime,
					Close:     50000.0,
					EventTime: baseTime.Add(10 * time.Second),
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "5m",
					OpenTime:  baseTime,
					Close:     50100.0,
					EventTime: baseTime.Add(10 * time.Second),
				},
			},
			expectedCount: 2,
			description:   "Same symbol, different intervals - no deduplication",
		},
		{
			name: "duplicate_updates_keep_latest",
			input: []*market_data.OHLCV{
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "1m",
					OpenTime:  baseTime,
					Close:     50000.0,
					EventTime: baseTime.Add(5 * time.Second),
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "1m",
					OpenTime:  baseTime,
					Close:     50050.0,
					EventTime: baseTime.Add(10 * time.Second),
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "1m",
					OpenTime:  baseTime,
					Close:     50100.0,
					EventTime: baseTime.Add(15 * time.Second),
				},
			},
			expectedCount:  1,
			expectedSymbol: "BTCUSDT",
			expectedClose:  50100.0, // Should keep the latest (15s)
			description:    "Multiple updates of same candle - keep only latest",
		},
		{
			name: "mixed_duplicates_and_unique",
			input: []*market_data.OHLCV{
				// BTCUSDT 1m - 3 updates
				{Exchange: "binance", Symbol: "BTCUSDT", Timeframe: "1m", OpenTime: baseTime, Close: 50000.0, EventTime: baseTime.Add(5 * time.Second)},
				{Exchange: "binance", Symbol: "BTCUSDT", Timeframe: "1m", OpenTime: baseTime, Close: 50100.0, EventTime: baseTime.Add(10 * time.Second)},
				{Exchange: "binance", Symbol: "BTCUSDT", Timeframe: "1m", OpenTime: baseTime, Close: 50200.0, EventTime: baseTime.Add(15 * time.Second)},

				// ETHUSDT 1m - 2 updates
				{Exchange: "binance", Symbol: "ETHUSDT", Timeframe: "1m", OpenTime: baseTime, Close: 3000.0, EventTime: baseTime.Add(5 * time.Second)},
				{Exchange: "binance", Symbol: "ETHUSDT", Timeframe: "1m", OpenTime: baseTime, Close: 3010.0, EventTime: baseTime.Add(12 * time.Second)},

				// BTCUSDT 5m - 1 update (unique)
				{Exchange: "binance", Symbol: "BTCUSDT", Timeframe: "5m", OpenTime: baseTime, Close: 50150.0, EventTime: baseTime.Add(10 * time.Second)},
			},
			expectedCount: 3, // Should have 3 unique klines after dedup
			description:   "Mixed scenario - should reduce from 6 to 3",
		},
		{
			name: "different_open_times_no_dedup",
			input: []*market_data.OHLCV{
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "1m",
					OpenTime:  baseTime,
					Close:     50000.0,
					EventTime: baseTime.Add(10 * time.Second),
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "1m",
					OpenTime:  baseTime.Add(1 * time.Minute), // Different candle
					Close:     50100.0,
					EventTime: baseTime.Add(70 * time.Second),
				},
			},
			expectedCount: 2,
			description:   "Different open times - different candles, no dedup",
		},
		{
			name: "out_of_order_events_keep_latest_by_event_time",
			input: []*market_data.OHLCV{
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "1m",
					OpenTime:  baseTime,
					Close:     50200.0,
					EventTime: baseTime.Add(30 * time.Second), // Latest
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "1m",
					OpenTime:  baseTime,
					Close:     50000.0,
					EventTime: baseTime.Add(5 * time.Second), // Oldest
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "1m",
					OpenTime:  baseTime,
					Close:     50100.0,
					EventTime: baseTime.Add(15 * time.Second), // Middle
				},
			},
			expectedCount:  1,
			expectedSymbol: "BTCUSDT",
			expectedClose:  50200.0, // Should keep event with event_time = 30s
			description:    "Out-of-order events - keep latest by event_time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := consumer.deduplicateBatch(tt.input)

			// Check count
			assert.Equal(t, tt.expectedCount, len(result), tt.description)

			// If specific symbol/close expected, verify it
			if tt.expectedSymbol != "" {
				found := false
				for _, ohlcv := range result {
					if ohlcv.Symbol == tt.expectedSymbol {
						found = true
						assert.Equal(t, tt.expectedClose, ohlcv.Close,
							"Should keep the latest update (by event_time)")
					}
				}
				assert.True(t, found, "Expected symbol should be in result")
			}

			// Verify no data loss - all unique klines should be present
			if tt.name == "mixed_duplicates_and_unique" {
				symbols := make(map[string]bool)
				for _, ohlcv := range result {
					key := ohlcv.Symbol + "_" + ohlcv.Timeframe
					symbols[key] = true
				}
				assert.Equal(t, 3, len(symbols), "Should have 3 unique symbol-interval pairs")
				assert.True(t, symbols["BTCUSDT_1m"], "Should have BTCUSDT 1m")
				assert.True(t, symbols["ETHUSDT_1m"], "Should have ETHUSDT 1m")
				assert.True(t, symbols["BTCUSDT_5m"], "Should have BTCUSDT 5m")
			}
		})
	}
}

func TestWebSocketKlineConsumer_DeduplicateBatch_NoDataLoss(t *testing.T) {
	consumer := &WebSocketKlineConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	// Simulate real scenario: multiple symbols, intervals, with updates
	input := []*market_data.OHLCV{
		// BTCUSDT 1m - 5 updates (real-time stream scenario)
		{Exchange: "binance", Symbol: "BTCUSDT", Timeframe: "1m", OpenTime: baseTime, Close: 50000.0, EventTime: baseTime.Add(1 * time.Second)},
		{Exchange: "binance", Symbol: "BTCUSDT", Timeframe: "1m", OpenTime: baseTime, Close: 50010.0, EventTime: baseTime.Add(2 * time.Second)},
		{Exchange: "binance", Symbol: "BTCUSDT", Timeframe: "1m", OpenTime: baseTime, Close: 50020.0, EventTime: baseTime.Add(3 * time.Second)},
		{Exchange: "binance", Symbol: "BTCUSDT", Timeframe: "1m", OpenTime: baseTime, Close: 50030.0, EventTime: baseTime.Add(4 * time.Second)},
		{Exchange: "binance", Symbol: "BTCUSDT", Timeframe: "1m", OpenTime: baseTime, Close: 50040.0, EventTime: baseTime.Add(5 * time.Second)},

		// BTCUSDT 5m - 3 updates
		{Exchange: "binance", Symbol: "BTCUSDT", Timeframe: "5m", OpenTime: baseTime, Close: 50050.0, EventTime: baseTime.Add(2 * time.Second)},
		{Exchange: "binance", Symbol: "BTCUSDT", Timeframe: "5m", OpenTime: baseTime, Close: 50060.0, EventTime: baseTime.Add(4 * time.Second)},
		{Exchange: "binance", Symbol: "BTCUSDT", Timeframe: "5m", OpenTime: baseTime, Close: 50070.0, EventTime: baseTime.Add(6 * time.Second)},

		// ETHUSDT 1m - 2 updates
		{Exchange: "binance", Symbol: "ETHUSDT", Timeframe: "1m", OpenTime: baseTime, Close: 3000.0, EventTime: baseTime.Add(1 * time.Second)},
		{Exchange: "binance", Symbol: "ETHUSDT", Timeframe: "1m", OpenTime: baseTime, Close: 3005.0, EventTime: baseTime.Add(3 * time.Second)},

		// SOLUSDT 1m - 1 update (no duplicates)
		{Exchange: "binance", Symbol: "SOLUSDT", Timeframe: "1m", OpenTime: baseTime, Close: 100.0, EventTime: baseTime.Add(2 * time.Second)},
	}

	result := consumer.deduplicateBatch(input)

	// Should reduce from 11 events to 4 unique klines
	require.Equal(t, 4, len(result), "Should have 4 unique klines after dedup")

	// Verify each unique kline has the latest close price
	priceMap := make(map[string]float64)
	for _, ohlcv := range result {
		key := ohlcv.Symbol + "_" + ohlcv.Timeframe
		priceMap[key] = ohlcv.Close
	}

	assert.Equal(t, 50040.0, priceMap["BTCUSDT_1m"], "BTCUSDT 1m should have latest close (50040)")
	assert.Equal(t, 50070.0, priceMap["BTCUSDT_5m"], "BTCUSDT 5m should have latest close (50070)")
	assert.Equal(t, 3005.0, priceMap["ETHUSDT_1m"], "ETHUSDT 1m should have latest close (3005)")
	assert.Equal(t, 100.0, priceMap["SOLUSDT_1m"], "SOLUSDT 1m should have close (100)")
}

func TestWebSocketKlineConsumer_DeduplicateBatch_EmptyBatch(t *testing.T) {
	consumer := &WebSocketKlineConsumer{}

	result := consumer.deduplicateBatch([]*market_data.OHLCV{})
	assert.Empty(t, result, "Empty batch should return empty result")
}

func TestWebSocketKlineConsumer_DeduplicateBatch_PreservesAllFields(t *testing.T) {
	consumer := &WebSocketKlineConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	input := []*market_data.OHLCV{
		{
			Exchange:            "binance",
			Symbol:              "BTCUSDT",
			Timeframe:           "1m",
			MarketType:          "futures",
			OpenTime:            baseTime,
			CloseTime:           baseTime.Add(1 * time.Minute),
			Open:                50000.0,
			High:                50100.0,
			Low:                 49900.0,
			Close:               50050.0,
			Volume:              100.5,
			QuoteVolume:         5000000.0,
			Trades:              1234,
			TakerBuyBaseVolume:  60.3,
			TakerBuyQuoteVolume: 3000000.0,
			IsClosed:            false,
			EventTime:           baseTime.Add(30 * time.Second),
		},
		{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Timeframe: "1m",
			OpenTime:  baseTime,
			Close:     50075.0,                        // Updated close
			EventTime: baseTime.Add(45 * time.Second), // Later event
		},
	}

	result := consumer.deduplicateBatch(input)

	require.Len(t, result, 1, "Should deduplicate to 1 kline")

	kline := result[0]
	assert.Equal(t, "binance", kline.Exchange)
	assert.Equal(t, "BTCUSDT", kline.Symbol)
	assert.Equal(t, "1m", kline.Timeframe)
	assert.Equal(t, 50075.0, kline.Close, "Should have latest close price")
	assert.Equal(t, baseTime.Add(45*time.Second), kline.EventTime, "Should have latest event time")
}

func TestWebSocketKlineConsumer_DeduplicateBatch_DifferentExchanges(t *testing.T) {
	consumer := &WebSocketKlineConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	input := []*market_data.OHLCV{
		{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Timeframe: "1m",
			OpenTime:  baseTime,
			Close:     50000.0,
			EventTime: baseTime.Add(10 * time.Second),
		},
		{
			Exchange:  "bybit",
			Symbol:    "BTCUSDT",
			Timeframe: "1m",
			OpenTime:  baseTime,
			Close:     50010.0,
			EventTime: baseTime.Add(10 * time.Second),
		},
	}

	result := consumer.deduplicateBatch(input)

	assert.Equal(t, 2, len(result), "Different exchanges should not deduplicate")
}

func TestWebSocketKlineConsumer_DeduplicateBatch_FinalVsNonFinal(t *testing.T) {
	consumer := &WebSocketKlineConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	input := []*market_data.OHLCV{
		// Non-final updates
		{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Timeframe: "1m",
			OpenTime:  baseTime,
			Close:     50000.0,
			IsClosed:  false,
			EventTime: baseTime.Add(10 * time.Second),
		},
		{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Timeframe: "1m",
			OpenTime:  baseTime,
			Close:     50050.0,
			IsClosed:  false,
			EventTime: baseTime.Add(30 * time.Second),
		},
		// Final close
		{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Timeframe: "1m",
			OpenTime:  baseTime,
			Close:     50075.0,
			IsClosed:  true,
			EventTime: baseTime.Add(60 * time.Second),
		},
	}

	result := consumer.deduplicateBatch(input)

	require.Len(t, result, 1, "Should keep only 1 kline")
	assert.True(t, result[0].IsClosed, "Should keep the final (closed) candle")
	assert.Equal(t, 50075.0, result[0].Close, "Should have final close price")
}

func TestWebSocketKlineConsumer_DeduplicateBatch_Performance(t *testing.T) {
	consumer := &WebSocketKlineConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	// Create large batch with many duplicates (realistic scenario)
	// 3 symbols × 5 intervals × 20 updates = 300 events → should dedup to 15
	symbols := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}
	intervals := []string{"1m", "5m", "15m", "1h", "4h"}
	updates := 20

	input := make([]*market_data.OHLCV, 0, len(symbols)*len(intervals)*updates)

	for _, symbol := range symbols {
		for _, interval := range intervals {
			for i := 0; i < updates; i++ {
				input = append(input, &market_data.OHLCV{
					Exchange:  "binance",
					Symbol:    symbol,
					Timeframe: interval,
					OpenTime:  baseTime,
					Close:     50000.0 + float64(i*10),
					EventTime: baseTime.Add(time.Duration(i+1) * time.Second),
				})
			}
		}
	}

	require.Len(t, input, 300, "Should have 300 input events")

	start := time.Now()
	result := consumer.deduplicateBatch(input)
	duration := time.Since(start)

	expectedUnique := len(symbols) * len(intervals) // 3 × 5 = 15
	assert.Equal(t, expectedUnique, len(result), "Should deduplicate 300 → 15 events")
	assert.Less(t, duration.Milliseconds(), int64(10), "Deduplication should be fast (<10ms)")

	t.Logf("✅ Deduplicated %d → %d events in %v", len(input), len(result), duration)
}

func TestWebSocketKlineConsumer_ConvertBatchToValues(t *testing.T) {
	consumer := &WebSocketKlineConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	input := []*market_data.OHLCV{
		{Exchange: "binance", Symbol: "BTCUSDT", Timeframe: "1m", OpenTime: baseTime, Close: 50000.0},
		{Exchange: "binance", Symbol: "ETHUSDT", Timeframe: "5m", OpenTime: baseTime, Close: 3000.0},
	}

	result := consumer.convertBatchToValues(input)

	require.Len(t, result, 2)
	assert.Equal(t, "BTCUSDT", result[0].Symbol)
	assert.Equal(t, "ETHUSDT", result[1].Symbol)
	assert.Equal(t, 50000.0, result[0].Close)
	assert.Equal(t, 3000.0, result[1].Close)
}
