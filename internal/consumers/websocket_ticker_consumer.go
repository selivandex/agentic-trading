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
	tickerBatchSize     = 100
	tickerFlushInterval = 5 * time.Second
	tickerStatsInterval = 1 * time.Minute
)

// WebSocketTickerConsumer consumes ticker events from Kafka and writes to ClickHouse
type WebSocketTickerConsumer struct {
	*WebSocketBaseConsumer[*eventspb.WebSocketTickerEvent, market_data.Ticker]
	repository market_data.Repository
}

// tickerHandler implements EventHandler interface
type tickerHandler struct {
	repository market_data.Repository
}

// NewWebSocketTickerConsumer creates a new ticker consumer
func NewWebSocketTickerConsumer(
	consumer *kafkaadapter.Consumer,
	repository market_data.Repository,
	log *logger.Logger,
) *WebSocketTickerConsumer {
	handler := &tickerHandler{repository: repository}

	baseConsumer := NewWebSocketBaseConsumer[*eventspb.WebSocketTickerEvent, market_data.Ticker](
		consumer,
		repository,
		log,
		BatchProcessingConfig{
			ConsumerName:  "WebSocket Ticker Consumer",
			BatchSize:     tickerBatchSize,
			FlushInterval: tickerFlushInterval,
			StatsInterval: tickerStatsInterval,
		},
		handler,
	)

	return &WebSocketTickerConsumer{
		WebSocketBaseConsumer: baseConsumer,
		repository:            repository,
	}
}

// UnmarshalEvent implements EventHandler interface
func (h *tickerHandler) UnmarshalEvent(data []byte) (*eventspb.WebSocketTickerEvent, error) {
	var event eventspb.WebSocketTickerEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ConvertToDomain implements EventHandler interface
func (h *tickerHandler) ConvertToDomain(event *eventspb.WebSocketTickerEvent) market_data.Ticker {
	parseFloat := func(s string) float64 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	// Use market_type from event, fallback to "futures" for backward compatibility
	marketType := event.MarketType
	if marketType == "" {
		marketType = "futures"
	}

	return market_data.Ticker{
		Exchange:           event.Exchange,
		Symbol:             event.Symbol,
		MarketType:         marketType,
		Timestamp:          event.EventTime.AsTime(),
		LastPrice:          parseFloat(event.LastPrice),
		OpenPrice:          parseFloat(event.OpenPrice),
		HighPrice:          parseFloat(event.HighPrice),
		LowPrice:           parseFloat(event.LowPrice),
		Volume:             parseFloat(event.Volume),
		QuoteVolume:        parseFloat(event.QuoteVolume),
		PriceChange:        parseFloat(event.PriceChange),
		PriceChangePercent: parseFloat(event.PriceChangePercent),
		WeightedAvgPrice:   parseFloat(event.WeightedAvgPrice),
		TradeCount:         uint64(event.TradeCount),
		EventTime:          event.EventTime.AsTime(),
	}
}

// GetDeduplicationKey implements EventHandler interface
func (h *tickerHandler) GetDeduplicationKey(item market_data.Ticker) string {
	return item.Exchange + "_" + item.Symbol + "_" + strconv.FormatInt(item.Timestamp.Unix(), 10)
}

// GetEventTime implements EventHandler interface
func (h *tickerHandler) GetEventTime(item market_data.Ticker) time.Time {
	return item.EventTime
}

// InsertBatch implements EventHandler interface
func (h *tickerHandler) InsertBatch(ctx context.Context, batch []market_data.Ticker) error {
	return h.repository.InsertTicker(ctx, batch)
}

// GetCustomStats implements EventHandler interface
func (h *tickerHandler) GetCustomStats() map[string]interface{} {
	return map[string]interface{}{} // No custom stats for ticker
}
