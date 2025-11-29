package consumers

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"

	eventspb "prometheus/internal/events/proto"
)

// TestTradeHandler_ConvertToDomain tests the conversion from protobuf to domain model
func TestTradeHandler_ConvertToDomain(t *testing.T) {
	handler := &tradeHandler{}

	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)
	tradeTime := baseTime.Add(5 * time.Second)

	event := &eventspb.WebSocketTradeEvent{
		Exchange:     "binance",
		Symbol:       "BTCUSDT",
		MarketType:   "futures",
		TradeId:      123456,
		Price:        "50000.50",
		Quantity:     "1.25",
		TradeTime:    timestamppb.New(tradeTime),
		IsBuyerMaker: true,
		EventTime:    timestamppb.New(baseTime),
	}

	trade := handler.ConvertToDomain(event)

	// Verify conversion
	assert.Equal(t, "binance", trade.Exchange)
	assert.Equal(t, "BTCUSDT", trade.Symbol)
	assert.Equal(t, "futures", trade.MarketType)
	assert.Equal(t, int64(123456), trade.TradeID)
	assert.Equal(t, int64(123456), trade.AggTradeID)
	assert.Equal(t, int64(123456), trade.FirstTradeID)
	assert.Equal(t, int64(123456), trade.LastTradeID)
	assert.InDelta(t, 50000.50, trade.Price, 0.01)
	assert.InDelta(t, 1.25, trade.Quantity, 0.01)
	assert.True(t, trade.IsBuyerMaker)
	assert.Equal(t, tradeTime, trade.Timestamp)
	assert.Equal(t, baseTime, trade.EventTime)
}

// TestTradeHandler_ConvertToDomain_WithMarketTypeDefault tests backward compatibility
func TestTradeHandler_ConvertToDomain_WithMarketTypeDefault(t *testing.T) {
	handler := &tradeHandler{}

	baseTime := time.Now()

	// Event without market_type - should default to "futures"
	event := &eventspb.WebSocketTradeEvent{
		Exchange:  "binance",
		Symbol:    "ETHUSDT",
		TradeId:   999,
		Price:     "3000.0",
		Quantity:  "10.0",
		TradeTime: timestamppb.New(baseTime),
		EventTime: timestamppb.New(baseTime),
		// MarketType not set
	}

	trade := handler.ConvertToDomain(event)

	assert.Equal(t, "futures", trade.MarketType, "Should default to futures when market_type not set")
}

// TestTradeHandler_GetDeduplicationKey tests deduplication key generation
func TestTradeHandler_GetDeduplicationKey(t *testing.T) {
	handler := &tradeHandler{}

	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		event1      *eventspb.WebSocketTradeEvent
		event2      *eventspb.WebSocketTradeEvent
		shouldMatch bool
	}{
		{
			name: "same_trade_id",
			event1: &eventspb.WebSocketTradeEvent{
				Exchange:  "binance",
				Symbol:    "BTCUSDT",
				TradeId:   123456,
				Price:     "50000.0",
				TradeTime: timestamppb.New(baseTime),
				EventTime: timestamppb.New(baseTime),
			},
			event2: &eventspb.WebSocketTradeEvent{
				Exchange:  "binance",
				Symbol:    "BTCUSDT",
				TradeId:   123456, // Same trade ID
				Price:     "50100.0",
				TradeTime: timestamppb.New(baseTime),
				EventTime: timestamppb.New(baseTime.Add(1 * time.Second)),
			},
			shouldMatch: true,
		},
		{
			name: "different_trade_id",
			event1: &eventspb.WebSocketTradeEvent{
				Exchange:  "binance",
				Symbol:    "BTCUSDT",
				TradeId:   123456,
				TradeTime: timestamppb.New(baseTime),
				EventTime: timestamppb.New(baseTime),
			},
			event2: &eventspb.WebSocketTradeEvent{
				Exchange:  "binance",
				Symbol:    "BTCUSDT",
				TradeId:   123457, // Different trade ID
				TradeTime: timestamppb.New(baseTime),
				EventTime: timestamppb.New(baseTime),
			},
			shouldMatch: false,
		},
		{
			name: "different_symbol_same_trade_id",
			event1: &eventspb.WebSocketTradeEvent{
				Exchange:  "binance",
				Symbol:    "BTCUSDT",
				TradeId:   123456,
				TradeTime: timestamppb.New(baseTime),
				EventTime: timestamppb.New(baseTime),
			},
			event2: &eventspb.WebSocketTradeEvent{
				Exchange:  "binance",
				Symbol:    "ETHUSDT", // Different symbol
				TradeId:   123456,
				TradeTime: timestamppb.New(baseTime),
				EventTime: timestamppb.New(baseTime),
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
			assert.Contains(t, key1, strconv.FormatInt(domain1.TradeID, 10))

			if tt.shouldMatch {
				assert.Equal(t, key1, key2, "Keys should match for deduplication")
			} else {
				assert.NotEqual(t, key1, key2, "Keys should not match")
			}
		})
	}
}
