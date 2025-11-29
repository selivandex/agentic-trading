package consumers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/market_data"
)

func TestWebSocketMarkPriceConsumer_DeduplicateBatch(t *testing.T) {
	consumer := &WebSocketMarkPriceConsumer{}

	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name              string
		input             []*market_data.MarkPrice
		expectedCount     int
		expectedSymbol    string
		expectedMarkPrice float64 // Expected mark price (from latest event)
		description       string
	}{
		{
			name: "no_duplicates",
			input: []*market_data.MarkPrice{
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timestamp: baseTime,
					MarkPrice: 50000.0,
					EventTime: baseTime.Add(10 * time.Second),
				},
				{
					Exchange:  "binance",
					Symbol:    "ETHUSDT",
					Timestamp: baseTime,
					MarkPrice: 3000.0,
					EventTime: baseTime.Add(10 * time.Second),
				},
			},
			expectedCount: 2,
			description:   "Different symbols - no deduplication",
		},
		{
			name: "same_symbol_different_timestamps",
			input: []*market_data.MarkPrice{
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timestamp: baseTime,
					MarkPrice: 50000.0,
					EventTime: baseTime.Add(10 * time.Second),
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timestamp: baseTime.Add(1 * time.Second),
					MarkPrice: 50100.0,
					EventTime: baseTime.Add(11 * time.Second),
				},
			},
			expectedCount: 2,
			description:   "Same symbol, different timestamps - no deduplication",
		},
		{
			name: "duplicate_updates_keep_latest",
			input: []*market_data.MarkPrice{
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timestamp: baseTime,
					MarkPrice: 50000.0,
					EventTime: baseTime.Add(5 * time.Second),
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timestamp: baseTime,
					MarkPrice: 50050.0,
					EventTime: baseTime.Add(10 * time.Second),
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timestamp: baseTime,
					MarkPrice: 50100.0,
					EventTime: baseTime.Add(15 * time.Second),
				},
			},
			expectedCount:     1,
			expectedSymbol:    "BTCUSDT",
			expectedMarkPrice: 50100.0, // Should keep the latest (15s)
			description:       "Multiple updates of same mark price - keep only latest",
		},
		{
			name: "mixed_duplicates_and_unique",
			input: []*market_data.MarkPrice{
				// BTCUSDT - 3 updates at same timestamp
				{Exchange: "binance", Symbol: "BTCUSDT", Timestamp: baseTime, MarkPrice: 50000.0, EventTime: baseTime.Add(5 * time.Second)},
				{Exchange: "binance", Symbol: "BTCUSDT", Timestamp: baseTime, MarkPrice: 50100.0, EventTime: baseTime.Add(10 * time.Second)},
				{Exchange: "binance", Symbol: "BTCUSDT", Timestamp: baseTime, MarkPrice: 50200.0, EventTime: baseTime.Add(15 * time.Second)},

				// ETHUSDT - 2 updates at same timestamp
				{Exchange: "binance", Symbol: "ETHUSDT", Timestamp: baseTime, MarkPrice: 3000.0, EventTime: baseTime.Add(5 * time.Second)},
				{Exchange: "binance", Symbol: "ETHUSDT", Timestamp: baseTime, MarkPrice: 3010.0, EventTime: baseTime.Add(12 * time.Second)},

				// BTCUSDT at different timestamp - unique
				{Exchange: "binance", Symbol: "BTCUSDT", Timestamp: baseTime.Add(1 * time.Second), MarkPrice: 50150.0, EventTime: baseTime.Add(10 * time.Second)},
			},
			expectedCount: 3, // Should have 3 unique mark prices after dedup
			description:   "Mixed scenario - should reduce from 6 to 3",
		},
		{
			name: "out_of_order_events_keep_latest_by_event_time",
			input: []*market_data.MarkPrice{
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timestamp: baseTime,
					MarkPrice: 50200.0,
					EventTime: baseTime.Add(30 * time.Second), // Latest
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timestamp: baseTime,
					MarkPrice: 50000.0,
					EventTime: baseTime.Add(5 * time.Second), // Oldest
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timestamp: baseTime,
					MarkPrice: 50100.0,
					EventTime: baseTime.Add(15 * time.Second), // Middle
				},
			},
			expectedCount:     1,
			expectedSymbol:    "BTCUSDT",
			expectedMarkPrice: 50200.0, // Should keep event with event_time = 30s
			description:       "Out-of-order events - keep latest by event_time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := consumer.deduplicateBatch(tt.input)

			// Check count
			assert.Equal(t, tt.expectedCount, len(result), tt.description)

			// If specific symbol/mark price expected, verify it
			if tt.expectedSymbol != "" {
				found := false
				for _, mp := range result {
					if mp.Symbol == tt.expectedSymbol {
						found = true
						assert.Equal(t, tt.expectedMarkPrice, mp.MarkPrice,
							"Should keep the latest update (by event_time)")
					}
				}
				assert.True(t, found, "Expected symbol should be in result")
			}

			// Verify no data loss - all unique mark prices should be present
			if tt.name == "mixed_duplicates_and_unique" {
				keys := make(map[string]bool)
				for _, mp := range result {
					key := mp.Symbol + "_" + mp.Timestamp.Format(time.RFC3339Nano)
					keys[key] = true
				}
				assert.Equal(t, 3, len(keys), "Should have 3 unique symbol-timestamp pairs")
			}
		})
	}
}

func TestWebSocketMarkPriceConsumer_DeduplicateBatch_NoDataLoss(t *testing.T) {
	consumer := &WebSocketMarkPriceConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	// Simulate real scenario: multiple symbols with rapid updates
	input := []*market_data.MarkPrice{
		// BTCUSDT - 5 updates at same timestamp (real-time stream)
		{Exchange: "binance", Symbol: "BTCUSDT", Timestamp: baseTime, MarkPrice: 50000.0, FundingRate: 0.0001, EventTime: baseTime.Add(1 * time.Second)},
		{Exchange: "binance", Symbol: "BTCUSDT", Timestamp: baseTime, MarkPrice: 50010.0, FundingRate: 0.0001, EventTime: baseTime.Add(2 * time.Second)},
		{Exchange: "binance", Symbol: "BTCUSDT", Timestamp: baseTime, MarkPrice: 50020.0, FundingRate: 0.0001, EventTime: baseTime.Add(3 * time.Second)},
		{Exchange: "binance", Symbol: "BTCUSDT", Timestamp: baseTime, MarkPrice: 50030.0, FundingRate: 0.0002, EventTime: baseTime.Add(4 * time.Second)},
		{Exchange: "binance", Symbol: "BTCUSDT", Timestamp: baseTime, MarkPrice: 50040.0, FundingRate: 0.0002, EventTime: baseTime.Add(5 * time.Second)},

		// ETHUSDT - 3 updates at same timestamp
		{Exchange: "binance", Symbol: "ETHUSDT", Timestamp: baseTime, MarkPrice: 3000.0, EventTime: baseTime.Add(1 * time.Second)},
		{Exchange: "binance", Symbol: "ETHUSDT", Timestamp: baseTime, MarkPrice: 3005.0, EventTime: baseTime.Add(3 * time.Second)},
		{Exchange: "binance", Symbol: "ETHUSDT", Timestamp: baseTime, MarkPrice: 3010.0, EventTime: baseTime.Add(5 * time.Second)},

		// SOLUSDT - 1 update (no duplicates)
		{Exchange: "binance", Symbol: "SOLUSDT", Timestamp: baseTime, MarkPrice: 100.0, EventTime: baseTime.Add(2 * time.Second)},
	}

	result := consumer.deduplicateBatch(input)

	// Should reduce from 9 events to 3 unique mark prices
	require.Equal(t, 3, len(result), "Should have 3 unique mark prices after dedup")

	// Verify each unique mark price has the latest value
	priceMap := make(map[string]float64)
	for _, mp := range result {
		priceMap[mp.Symbol] = mp.MarkPrice
	}

	assert.Equal(t, 50040.0, priceMap["BTCUSDT"], "BTCUSDT should have latest mark price (50040)")
	assert.Equal(t, 3010.0, priceMap["ETHUSDT"], "ETHUSDT should have latest mark price (3010)")
	assert.Equal(t, 100.0, priceMap["SOLUSDT"], "SOLUSDT should have mark price (100)")
}

func TestWebSocketMarkPriceConsumer_DeduplicateBatch_EmptyBatch(t *testing.T) {
	consumer := &WebSocketMarkPriceConsumer{}

	result := consumer.deduplicateBatch([]*market_data.MarkPrice{})
	assert.Empty(t, result, "Empty batch should return empty result")
}

func TestWebSocketMarkPriceConsumer_DeduplicateBatch_PreservesAllFields(t *testing.T) {
	consumer := &WebSocketMarkPriceConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	input := []*market_data.MarkPrice{
		{
			Exchange:             "binance",
			Symbol:               "BTCUSDT",
			MarketType:           "futures",
			Timestamp:            baseTime,
			MarkPrice:            50000.0,
			IndexPrice:           49950.0,
			EstimatedSettlePrice: 49980.0,
			FundingRate:          0.0001,
			NextFundingTime:      baseTime.Add(8 * time.Hour),
			EventTime:            baseTime.Add(30 * time.Second),
		},
		{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Timestamp: baseTime,
			MarkPrice: 50075.0,                        // Updated mark price
			EventTime: baseTime.Add(45 * time.Second), // Later event
		},
	}

	result := consumer.deduplicateBatch(input)

	require.Len(t, result, 1, "Should deduplicate to 1 mark price")

	mp := result[0]
	assert.Equal(t, "binance", mp.Exchange)
	assert.Equal(t, "BTCUSDT", mp.Symbol)
	assert.Equal(t, 50075.0, mp.MarkPrice, "Should have latest mark price")
	assert.Equal(t, baseTime.Add(45*time.Second), mp.EventTime, "Should have latest event time")
}

func TestWebSocketMarkPriceConsumer_DeduplicateBatch_DifferentExchanges(t *testing.T) {
	consumer := &WebSocketMarkPriceConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	input := []*market_data.MarkPrice{
		{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Timestamp: baseTime,
			MarkPrice: 50000.0,
			EventTime: baseTime.Add(10 * time.Second),
		},
		{
			Exchange:  "bybit",
			Symbol:    "BTCUSDT",
			Timestamp: baseTime,
			MarkPrice: 50010.0,
			EventTime: baseTime.Add(10 * time.Second),
		},
	}

	result := consumer.deduplicateBatch(input)

	assert.Equal(t, 2, len(result), "Different exchanges should not deduplicate")
}

func TestWebSocketMarkPriceConsumer_DeduplicateBatch_FundingRateUpdates(t *testing.T) {
	consumer := &WebSocketMarkPriceConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	input := []*market_data.MarkPrice{
		{
			Exchange:    "binance",
			Symbol:      "BTCUSDT",
			Timestamp:   baseTime,
			MarkPrice:   50000.0,
			FundingRate: 0.0001,
			EventTime:   baseTime.Add(10 * time.Second),
		},
		{
			Exchange:    "binance",
			Symbol:      "BTCUSDT",
			Timestamp:   baseTime,
			MarkPrice:   50050.0,
			FundingRate: 0.0002, // Funding rate changed
			EventTime:   baseTime.Add(30 * time.Second),
		},
	}

	result := consumer.deduplicateBatch(input)

	require.Len(t, result, 1, "Should keep only 1 mark price")
	assert.Equal(t, 0.0002, result[0].FundingRate, "Should have latest funding rate")
	assert.Equal(t, 50050.0, result[0].MarkPrice, "Should have latest mark price")
}

func TestWebSocketMarkPriceConsumer_DeduplicateBatch_Performance(t *testing.T) {
	consumer := &WebSocketMarkPriceConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	// Create large batch with many duplicates (realistic scenario)
	// 10 symbols × 50 updates = 500 events → should dedup to 10
	symbols := []string{
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "ADAUSDT",
		"DOTUSDT", "MATICUSDT", "LINKUSDT", "AVAXUSDT", "ATOMUSDT",
	}
	updates := 50

	input := make([]*market_data.MarkPrice, 0, len(symbols)*updates)

	for _, symbol := range symbols {
		for i := 0; i < updates; i++ {
			input = append(input, &market_data.MarkPrice{
				Exchange:  "binance",
				Symbol:    symbol,
				Timestamp: baseTime,
				MarkPrice: 50000.0 + float64(i*10),
				EventTime: baseTime.Add(time.Duration(i+1) * time.Second),
			})
		}
	}

	require.Len(t, input, 500, "Should have 500 input events")

	start := time.Now()
	result := consumer.deduplicateBatch(input)
	duration := time.Since(start)

	expectedUnique := len(symbols) // 10
	assert.Equal(t, expectedUnique, len(result), "Should deduplicate 500 → 10 events")
	assert.Less(t, duration.Milliseconds(), int64(10), "Deduplication should be fast (<10ms)")

	t.Logf("✅ Deduplicated %d → %d events in %v", len(input), len(result), duration)
}
