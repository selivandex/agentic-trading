package consumers

import (
	"context"
	"strconv"
	"time"

	"google.golang.org/protobuf/proto"

	kafkaadapter "prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/market_data"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/logger"
)

const (
	// Batch size for mark price writes
	markPriceBatchSize = 100
	// Flush interval if batch is not full
	markPriceFlushInterval = 5 * time.Second
	// Stats logging interval
	markPriceStatsInterval = 1 * time.Minute
)

// WebSocketMarkPriceConsumer consumes mark price events from Kafka and writes to ClickHouse
type WebSocketMarkPriceConsumer struct {
	*WebSocketBaseConsumer[*eventspb.WebSocketMarkPriceEvent, market_data.MarkPrice]
	repository market_data.Repository
}

// markPriceHandler implements EventHandler interface
type markPriceHandler struct {
	repository market_data.Repository
}

// NewWebSocketMarkPriceConsumer creates a new mark price consumer
func NewWebSocketMarkPriceConsumer(
	consumer *kafkaadapter.Consumer,
	repository market_data.Repository,
	log *logger.Logger,
) *WebSocketMarkPriceConsumer {
	handler := &markPriceHandler{repository: repository}

	baseConsumer := NewWebSocketBaseConsumer[*eventspb.WebSocketMarkPriceEvent, market_data.MarkPrice](
		consumer,
		repository,
		log,
		BatchProcessingConfig{
			ConsumerName:  "WebSocket Mark Price Consumer",
			BatchSize:     markPriceBatchSize,
			FlushInterval: markPriceFlushInterval,
			StatsInterval: markPriceStatsInterval,
		},
		handler,
	)

	return &WebSocketMarkPriceConsumer{
		WebSocketBaseConsumer: baseConsumer,
		repository:            repository,
	}
}

// UnmarshalEvent implements EventHandler interface
func (h *markPriceHandler) UnmarshalEvent(data []byte) (*eventspb.WebSocketMarkPriceEvent, error) {
	var event eventspb.WebSocketMarkPriceEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ConvertToDomain implements EventHandler interface
func (h *markPriceHandler) ConvertToDomain(event *eventspb.WebSocketMarkPriceEvent) market_data.MarkPrice {
	// Helper to parse string to float64
	parseFloat := func(s string) float64 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	// Mark price is always futures (it doesn't exist for spot)
	return market_data.MarkPrice{
		Exchange:             event.Exchange,
		Symbol:               event.Symbol,
		MarketType:           "futures", // Mark price only exists for futures
		Timestamp:            event.EventTime.AsTime(),
		MarkPrice:            parseFloat(event.MarkPrice),
		IndexPrice:           parseFloat(event.IndexPrice),
		EstimatedSettlePrice: parseFloat(event.EstimatedSettlePrice),
		FundingRate:          parseFloat(event.FundingRate),
		NextFundingTime:      event.NextFundingTime.AsTime(),
		EventTime:            event.EventTime.AsTime(),
	}
}

// GetDeduplicationKey implements EventHandler interface
func (h *markPriceHandler) GetDeduplicationKey(item market_data.MarkPrice) string {
	// Create unique key: exchange_symbol_timestamp
	return item.Exchange + "_" + item.Symbol + "_" + strconv.FormatInt(item.Timestamp.Unix(), 10)
}

// GetEventTime implements EventHandler interface
func (h *markPriceHandler) GetEventTime(item market_data.MarkPrice) time.Time {
	return item.EventTime
}

// InsertBatch implements EventHandler interface
func (h *markPriceHandler) InsertBatch(ctx context.Context, batch []market_data.MarkPrice) error {
	return h.repository.InsertMarkPrice(ctx, batch)
}

// GetCustomStats implements EventHandler interface
func (h *markPriceHandler) GetCustomStats() map[string]interface{} {
	return map[string]interface{}{} // No custom stats for mark price
}
