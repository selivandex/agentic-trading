package consumers

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"

	eventspb "prometheus/internal/events/proto"
)

// TestTickerHandler_ConvertToDomain tests the conversion from protobuf to domain model
func TestTickerHandler_ConvertToDomain(t *testing.T) {
	handler := &tickerHandler{}

	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	event := &eventspb.WebSocketTickerEvent{
		Exchange:           "binance",
		Symbol:             "BTCUSDT",
		MarketType:         "futures",
		LastPrice:          "50000.50",
		OpenPrice:          "49500.00",
		HighPrice:          "50200.00",
		LowPrice:           "49400.00",
		Volume:             "1234.56",
		QuoteVolume:        "61728000.00",
		PriceChange:        "500.50",
		PriceChangePercent: "1.01",
		WeightedAvgPrice:   "49950.00",
		TradeCount:         10000,
		EventTime:          timestamppb.New(baseTime),
	}

	ticker := handler.ConvertToDomain(event)

	// Verify conversion
	assert.Equal(t, "binance", ticker.Exchange)
	assert.Equal(t, "BTCUSDT", ticker.Symbol)
	assert.Equal(t, "futures", ticker.MarketType)
	assert.InDelta(t, 50000.50, ticker.LastPrice, 0.01)
	assert.InDelta(t, 49500.00, ticker.OpenPrice, 0.01)
	assert.InDelta(t, 50200.00, ticker.HighPrice, 0.01)
	assert.InDelta(t, 49400.00, ticker.LowPrice, 0.01)
	assert.InDelta(t, 1234.56, ticker.Volume, 0.01)
	assert.InDelta(t, 61728000.00, ticker.QuoteVolume, 0.01)
	assert.InDelta(t, 500.50, ticker.PriceChange, 0.01)
	assert.InDelta(t, 1.01, ticker.PriceChangePercent, 0.01)
	assert.InDelta(t, 49950.00, ticker.WeightedAvgPrice, 0.01)
	assert.Equal(t, uint64(10000), ticker.TradeCount)
	assert.Equal(t, baseTime, ticker.Timestamp)
	assert.Equal(t, baseTime, ticker.EventTime)
}

// TestTickerHandler_ConvertToDomain_WithMarketTypeDefault tests backward compatibility
func TestTickerHandler_ConvertToDomain_WithMarketTypeDefault(t *testing.T) {
	handler := &tickerHandler{}

	baseTime := time.Now()

	// Event without market_type - should default to "futures"
	event := &eventspb.WebSocketTickerEvent{
		Exchange:  "binance",
		Symbol:    "ETHUSDT",
		LastPrice: "3000.0",
		EventTime: timestamppb.New(baseTime),
		// MarketType not set
	}

	ticker := handler.ConvertToDomain(event)

	assert.Equal(t, "futures", ticker.MarketType, "Should default to futures when market_type not set")
}

// TestTickerHandler_GetDeduplicationKey tests deduplication key generation
func TestTickerHandler_GetDeduplicationKey(t *testing.T) {
	handler := &tickerHandler{}

	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		event1      *eventspb.WebSocketTickerEvent
		event2      *eventspb.WebSocketTickerEvent
		shouldMatch bool
	}{
		{
			name: "same_key_different_price",
			event1: &eventspb.WebSocketTickerEvent{
				Exchange:  "binance",
				Symbol:    "BTCUSDT",
				LastPrice: "50000.0",
				EventTime: timestamppb.New(baseTime),
			},
			event2: &eventspb.WebSocketTickerEvent{
				Exchange:  "binance",
				Symbol:    "BTCUSDT",
				LastPrice: "50100.0", // Different price - should deduplicate
				EventTime: timestamppb.New(baseTime),
			},
			shouldMatch: true,
		},
		{
			name: "different_symbol",
			event1: &eventspb.WebSocketTickerEvent{
				Exchange:  "binance",
				Symbol:    "BTCUSDT",
				EventTime: timestamppb.New(baseTime),
			},
			event2: &eventspb.WebSocketTickerEvent{
				Exchange:  "binance",
				Symbol:    "ETHUSDT", // Different symbol
				EventTime: timestamppb.New(baseTime),
			},
			shouldMatch: false,
		},
		{
			name: "different_timestamp",
			event1: &eventspb.WebSocketTickerEvent{
				Exchange:  "binance",
				Symbol:    "BTCUSDT",
				EventTime: timestamppb.New(baseTime),
			},
			event2: &eventspb.WebSocketTickerEvent{
				Exchange:  "binance",
				Symbol:    "BTCUSDT",
				EventTime: timestamppb.New(baseTime.Add(1 * time.Second)), // Different timestamp
			},
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domain1 := handler.ConvertToDomain(tt.event1)
			domain2 := handler.ConvertToDomain(tt.event2)

			key1 := handler.GetDeduplicationKey(domain1)
			key2 := handler.GetDeduplicationKey(domain2)

			// Verify key format
			assert.Contains(t, key1, domain1.Exchange)
			assert.Contains(t, key1, domain1.Symbol)
			assert.Contains(t, key1, strconv.FormatInt(domain1.Timestamp.Unix(), 10))

			if tt.shouldMatch {
				assert.Equal(t, key1, key2, "Keys should match for deduplication")
			} else {
				assert.NotEqual(t, key1, key2, "Keys should not match")
			}
		})
	}
}
