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
	liquidationBatchSize     = 50 // Smaller batch - liquidations are less frequent but important
	liquidationFlushInterval = 5 * time.Second
	liquidationStatsInterval = 1 * time.Minute
)

// WebSocketLiquidationConsumer consumes liquidation events from Kafka and writes to ClickHouse
type WebSocketLiquidationConsumer struct {
	*WebSocketBaseConsumer[*eventspb.WebSocketLiquidationEvent, market_data.Liquidation]
	repository market_data.Repository
}

// liquidationHandler implements EventHandler interface
type liquidationHandler struct {
	repository market_data.Repository
}

// NewWebSocketLiquidationConsumer creates a new liquidation consumer
func NewWebSocketLiquidationConsumer(
	consumer *kafkaadapter.Consumer,
	repository market_data.Repository,
	log *logger.Logger,
) *WebSocketLiquidationConsumer {
	handler := &liquidationHandler{repository: repository}

	baseConsumer := NewWebSocketBaseConsumer[*eventspb.WebSocketLiquidationEvent, market_data.Liquidation](
		consumer,
		repository,
		log,
		BatchProcessingConfig{
			ConsumerName:  "WebSocket Liquidation Consumer",
			BatchSize:     liquidationBatchSize,
			FlushInterval: liquidationFlushInterval,
			StatsInterval: liquidationStatsInterval,
		},
		handler,
	)

	return &WebSocketLiquidationConsumer{
		WebSocketBaseConsumer: baseConsumer,
		repository:            repository,
	}
}

// UnmarshalEvent implements EventHandler interface
func (h *liquidationHandler) UnmarshalEvent(data []byte) (*eventspb.WebSocketLiquidationEvent, error) {
	var event eventspb.WebSocketLiquidationEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ConvertToDomain implements EventHandler interface
func (h *liquidationHandler) ConvertToDomain(event *eventspb.WebSocketLiquidationEvent) market_data.Liquidation {
	price, _ := strconv.ParseFloat(event.Price, 64)
	quantity, _ := strconv.ParseFloat(event.Quantity, 64)
	value, _ := strconv.ParseFloat(event.Value, 64)

	return market_data.Liquidation{
		Exchange:   event.Exchange,
		Symbol:     event.Symbol,
		MarketType: event.MarketType,
		Timestamp:  event.EventTime.AsTime(),
		Side:       event.Side,
		OrderType:  event.OrderType,
		Price:      price,
		Quantity:   quantity,
		Value:      value,
		EventTime:  event.EventTime.AsTime(),
	}
}

// GetDeduplicationKey implements EventHandler interface
// Liquidations don't need deduplication (no unique ID)
func (h *liquidationHandler) GetDeduplicationKey(item market_data.Liquidation) string {
	return "" // No deduplication for liquidations
}

// GetEventTime implements EventHandler interface
func (h *liquidationHandler) GetEventTime(item market_data.Liquidation) time.Time {
	return item.EventTime
}

// InsertBatch implements EventHandler interface
func (h *liquidationHandler) InsertBatch(ctx context.Context, batch []market_data.Liquidation) error {
	return h.repository.InsertLiquidations(ctx, batch)
}

// GetCustomStats implements EventHandler interface
func (h *liquidationHandler) GetCustomStats() map[string]interface{} {
	return map[string]interface{}{} // No custom stats for liquidations
}
