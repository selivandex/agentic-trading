package consumers

import (
	"context"
	"strconv"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	kafkaadapter "prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/market_data"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/logger"
)

const (
	// Batch size for ClickHouse writes
	klineBatchSize = 50
	// Flush interval if batch is not full
	klineFlushInterval = 10 * time.Second
	// Stats logging interval
	klineStatsInterval = 1 * time.Minute
)

// WebSocketKlineConsumer consumes kline events from Kafka and writes to ClickHouse in batches
type WebSocketKlineConsumer struct {
	*WebSocketBaseConsumer[*eventspb.WebSocketKlineEvent, market_data.OHLCV]
	repository market_data.Repository
	handler    *klineHandler
}

// klineHandler implements EventHandler interface
type klineHandler struct {
	repository market_data.Repository

	// Additional statistics for klines
	statsMu       sync.Mutex
	totalFinal    int64 // Final (closed) candles
	totalNonFinal int64 // Non-final (updating) candles
}

// NewWebSocketKlineConsumer creates a new WebSocket kline consumer
func NewWebSocketKlineConsumer(
	consumer *kafkaadapter.Consumer,
	repository market_data.Repository,
	log *logger.Logger,
) *WebSocketKlineConsumer {
	handler := &klineHandler{repository: repository}

	baseConsumer := NewWebSocketBaseConsumer[*eventspb.WebSocketKlineEvent, market_data.OHLCV](
		consumer,
		repository,
		log,
		BatchProcessingConfig{
			ConsumerName:  "WebSocket Kline Consumer",
			BatchSize:     klineBatchSize,
			FlushInterval: klineFlushInterval,
			StatsInterval: klineStatsInterval,
		},
		handler,
	)

	return &WebSocketKlineConsumer{
		WebSocketBaseConsumer: baseConsumer,
		repository:            repository,
		handler:               handler,
	}
}

// UnmarshalEvent implements EventHandler interface
func (h *klineHandler) UnmarshalEvent(data []byte) (*eventspb.WebSocketKlineEvent, error) {
	var event eventspb.WebSocketKlineEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ConvertToDomain implements EventHandler interface
func (h *klineHandler) ConvertToDomain(event *eventspb.WebSocketKlineEvent) market_data.OHLCV {
	// Helper to parse string to float64
	parseFloat := func(s string) float64 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	// Use market_type from event, fallback to "futures" for backward compatibility
	marketType := event.MarketType
	if marketType == "" {
		marketType = "futures"
	}

	// Track stats by candle type
	h.statsMu.Lock()
	if event.IsFinal {
		h.totalFinal++
	} else {
		h.totalNonFinal++
	}
	h.statsMu.Unlock()

	return market_data.OHLCV{
		Exchange:            event.Exchange,
		Symbol:              event.Symbol,
		Timeframe:           event.Interval,
		MarketType:          marketType,
		OpenTime:            event.OpenTime.AsTime(),
		CloseTime:           event.CloseTime.AsTime(),
		Open:                parseFloat(event.Open),
		High:                parseFloat(event.High),
		Low:                 parseFloat(event.Low),
		Close:               parseFloat(event.Close),
		Volume:              parseFloat(event.Volume),
		QuoteVolume:         parseFloat(event.QuoteVolume),
		Trades:              uint64(event.TradeCount),
		TakerBuyBaseVolume:  0, // Not available in basic kline event
		TakerBuyQuoteVolume: 0, // Not available in basic kline event
		IsClosed:            event.IsFinal,
		EventTime:           event.EventTime.AsTime(),
	}
}

// GetDeduplicationKey implements EventHandler interface
func (h *klineHandler) GetDeduplicationKey(item market_data.OHLCV) string {
	// Create unique key: exchange_symbol_interval_opentime
	return item.Exchange + "_" + item.Symbol + "_" + item.Timeframe + "_" +
		strconv.FormatInt(item.OpenTime.Unix(), 10)
}

// GetEventTime implements EventHandler interface
func (h *klineHandler) GetEventTime(item market_data.OHLCV) time.Time {
	return item.EventTime
}

// InsertBatch implements EventHandler interface
func (h *klineHandler) InsertBatch(ctx context.Context, batch []market_data.OHLCV) error {
	return h.repository.InsertOHLCV(ctx, batch)
}

// GetCustomStats implements EventHandler interface
func (h *klineHandler) GetCustomStats() map[string]interface{} {
	h.statsMu.Lock()
	defer h.statsMu.Unlock()

	finalPercent := 0.0
	total := h.totalFinal + h.totalNonFinal
	if total > 0 {
		finalPercent = float64(h.totalFinal) / float64(total) * 100
	}

	return map[string]interface{}{
		"total_final":     h.totalFinal,
		"total_non_final": h.totalNonFinal,
		"final_percent":   int64(finalPercent),
	}
}
