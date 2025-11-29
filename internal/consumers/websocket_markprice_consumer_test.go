package consumers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"

	eventspb "prometheus/internal/events/proto"
)

// TestMarkPriceHandler_ConvertToDomain tests the conversion from protobuf to domain model
func TestMarkPriceHandler_ConvertToDomain(t *testing.T) {
	handler := &markPriceHandler{}

	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)
	nextFundingTime := baseTime.Add(8 * time.Hour)

	event := &eventspb.WebSocketMarkPriceEvent{
		Exchange:             "binance",
		Symbol:               "BTCUSDT",
		MarkPrice:            "50000.50",
		IndexPrice:           "50005.25",
		EstimatedSettlePrice: "50002.75",
		FundingRate:          "0.0001",
		NextFundingTime:      timestamppb.New(nextFundingTime),
		EventTime:            timestamppb.New(baseTime),
	}

	markPrice := handler.ConvertToDomain(event)

	// Verify conversion
	assert.Equal(t, "binance", markPrice.Exchange)
	assert.Equal(t, "BTCUSDT", markPrice.Symbol)
	assert.Equal(t, "futures", markPrice.MarketType) // Always futures
	assert.InDelta(t, 50000.50, markPrice.MarkPrice, 0.01)
	assert.InDelta(t, 50005.25, markPrice.IndexPrice, 0.01)
	assert.InDelta(t, 50002.75, markPrice.EstimatedSettlePrice, 0.01)
	assert.InDelta(t, 0.0001, markPrice.FundingRate, 0.00001)
	assert.Equal(t, nextFundingTime, markPrice.NextFundingTime)
	assert.Equal(t, baseTime, markPrice.Timestamp)
	assert.Equal(t, baseTime, markPrice.EventTime)
}

// TestMarkPriceHandler_ConvertToDomain_InvalidNumbers tests handling of invalid number strings
func TestMarkPriceHandler_ConvertToDomain_InvalidNumbers(t *testing.T) {
	handler := &markPriceHandler{}

	baseTime := time.Now()

	event := &eventspb.WebSocketMarkPriceEvent{
		Exchange:             "binance",
		Symbol:               "ETHUSDT",
		MarkPrice:            "invalid",
		IndexPrice:           "not-a-number",
		EstimatedSettlePrice: "bad-value",
		FundingRate:          "invalid-rate",
		NextFundingTime:      timestamppb.New(baseTime),
		EventTime:            timestamppb.New(baseTime),
	}

	markPrice := handler.ConvertToDomain(event)

	// Invalid numbers should parse as 0
	assert.Equal(t, 0.0, markPrice.MarkPrice)
	assert.Equal(t, 0.0, markPrice.IndexPrice)
	assert.Equal(t, 0.0, markPrice.EstimatedSettlePrice)
	assert.Equal(t, 0.0, markPrice.FundingRate)

	// Other fields should still be correct
	assert.Equal(t, "binance", markPrice.Exchange)
	assert.Equal(t, "ETHUSDT", markPrice.Symbol)
	assert.Equal(t, "futures", markPrice.MarketType)
}

// TestMarkPriceHandler_GetDeduplicationKey tests deduplication key generation
func TestMarkPriceHandler_GetDeduplicationKey(t *testing.T) {
	handler := &markPriceHandler{}

	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		event1      *eventspb.WebSocketMarkPriceEvent
		event2      *eventspb.WebSocketMarkPriceEvent
		shouldMatch bool
	}{
		{
			name: "same_key_different_mark_price",
			event1: &eventspb.WebSocketMarkPriceEvent{
				Exchange:  "binance",
				Symbol:    "BTCUSDT",
				MarkPrice: "50000.0",
				EventTime: timestamppb.New(baseTime),
			},
			event2: &eventspb.WebSocketMarkPriceEvent{
				Exchange:  "binance",
				Symbol:    "BTCUSDT",
				MarkPrice: "50100.0", // Different mark price - should deduplicate
				EventTime: timestamppb.New(baseTime),
			},
			shouldMatch: true,
		},
		{
			name: "different_symbol",
			event1: &eventspb.WebSocketMarkPriceEvent{
				Exchange:  "binance",
				Symbol:    "BTCUSDT",
				EventTime: timestamppb.New(baseTime),
			},
			event2: &eventspb.WebSocketMarkPriceEvent{
				Exchange:  "binance",
				Symbol:    "ETHUSDT", // Different symbol
				EventTime: timestamppb.New(baseTime),
			},
			shouldMatch: false,
		},
		{
			name: "different_timestamp",
			event1: &eventspb.WebSocketMarkPriceEvent{
				Exchange:  "binance",
				Symbol:    "BTCUSDT",
				EventTime: timestamppb.New(baseTime),
			},
			event2: &eventspb.WebSocketMarkPriceEvent{
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

			if tt.shouldMatch {
				assert.Equal(t, key1, key2, "Keys should match for deduplication")
			} else {
				assert.NotEqual(t, key1, key2, "Keys should not match")
			}
		})
	}
}
