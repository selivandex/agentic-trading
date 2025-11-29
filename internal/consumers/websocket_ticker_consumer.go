package consumers

import (
	"context"
	"strconv"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/market_data"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

const (
	tickerBatchSize     = 100
	tickerFlushInterval = 5 * time.Second
	tickerStatsInterval = 1 * time.Minute
)

// WebSocketTickerConsumer consumes ticker events from Kafka and writes to ClickHouse
type WebSocketTickerConsumer struct {
	consumer   *kafka.Consumer
	repository market_data.Repository
	log        *logger.Logger

	mu    sync.Mutex
	batch []*market_data.Ticker

	statsMu               sync.Mutex
	totalReceived         int64
	totalProcessed        int64
	totalDeduplicated     int64
	totalErrors           int64
	lastFlushTime         time.Time
	lastStatsLogTime      time.Time
	receivedSinceLastLog  int64
	processedSinceLastLog int64
}

// NewWebSocketTickerConsumer creates a new ticker consumer
func NewWebSocketTickerConsumer(
	consumer *kafka.Consumer,
	repository market_data.Repository,
	log *logger.Logger,
) *WebSocketTickerConsumer {
	now := time.Now()
	return &WebSocketTickerConsumer{
		consumer:         consumer,
		repository:       repository,
		log:              log,
		batch:            make([]*market_data.Ticker, 0, tickerBatchSize),
		lastFlushTime:    now,
		lastStatsLogTime: now,
	}
}

// Start begins consuming ticker events
func (c *WebSocketTickerConsumer) Start(ctx context.Context) error {
	c.log.Info("Starting WebSocket ticker consumer...",
		"batch_size", tickerBatchSize,
		"flush_interval", tickerFlushInterval,
	)

	flushTicker := time.NewTicker(tickerFlushInterval)
	defer flushTicker.Stop()

	statsTicker := time.NewTicker(tickerStatsInterval)
	defer statsTicker.Stop()

	defer func() {
		c.log.Info("Closing WebSocket ticker consumer...")
		if err := c.flushBatch(context.Background()); err != nil {
			c.log.Error("Failed to flush final batch", "error", err)
		}
		c.logStats(true)
		if err := c.consumer.Close(); err != nil {
			c.log.Error("Failed to close consumer", "error", err)
		} else {
			c.log.Info("✓ WebSocket ticker consumer closed")
		}
	}()

	go c.periodicFlush(ctx, flushTicker.C)
	go c.periodicStatsLog(ctx, statsTicker.C)

	c.log.Info("🔄 Starting to read messages from Kafka...",
		"topic", "websocket.ticker",
	)

	for {
		msg, err := c.consumer.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				c.log.Info("Ticker consumer stopping (context cancelled)")
				return nil
			}
			c.log.Error("Failed to read ticker event", "error", err)
			continue
		}

		c.incrementStat(&c.totalReceived)
		c.incrementStat(&c.receivedSinceLastLog)

		c.log.Debug("📩 Received message from Kafka",
			"partition", msg.Partition,
			"offset", msg.Offset,
			"size_bytes", len(msg.Value),
		)

		processCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := c.handleTickerEvent(processCtx, msg.Value); err != nil {
			c.log.Error("Failed to handle ticker event", "error", err)
			c.incrementStat(&c.totalErrors)
		}
		cancel()

		if ctx.Err() != nil {
			c.log.Info("Ticker consumer stopping after processing current message")
			return nil
		}
	}
}

func (c *WebSocketTickerConsumer) handleTickerEvent(ctx context.Context, data []byte) error {
	var event eventspb.WebSocketTickerEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal ticker event")
	}

	ticker := c.convertProtobufToTicker(&event)
	c.addToBatch(ticker)

	c.log.Debug("Added ticker to batch",
		"exchange", event.Exchange,
		"symbol", event.Symbol,
		"last_price", event.LastPrice,
		"batch_size", len(c.batch),
	)

	if len(c.batch) >= tickerBatchSize {
		return c.flushBatch(ctx)
	}

	return nil
}

func (c *WebSocketTickerConsumer) addToBatch(ticker *market_data.Ticker) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.batch = append(c.batch, ticker)
}

func (c *WebSocketTickerConsumer) flushBatch(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.batch) == 0 {
		return nil
	}

	originalSize := len(c.batch)
	start := time.Now()

	dedupedBatch := c.deduplicateBatch(c.batch)
	deduplicatedCount := originalSize - len(dedupedBatch)

	if deduplicatedCount > 0 {
		c.statsMu.Lock()
		c.totalDeduplicated += int64(deduplicatedCount)
		c.statsMu.Unlock()

		c.log.Debug("Deduplicated batch before write",
			"original_size", originalSize,
			"deduplicated_size", len(dedupedBatch),
			"removed", deduplicatedCount,
		)
	}

	if err := c.repository.InsertTicker(ctx, dedupedBatch); err != nil {
		c.log.Error("Failed to insert ticker batch", "error", err)
		return errors.Wrap(err, "insert batch to ClickHouse")
	}

	duration := time.Since(start)

	c.log.Info("✅ Flushed ticker batch to ClickHouse",
		"original_size", originalSize,
		"written_size", len(dedupedBatch),
		"deduplicated", deduplicatedCount,
		"duration_ms", duration.Milliseconds(),
	)

	c.statsMu.Lock()
	c.totalProcessed += int64(len(dedupedBatch))
	c.processedSinceLastLog += int64(len(dedupedBatch))
	c.lastFlushTime = time.Now()
	c.statsMu.Unlock()

	c.batch = c.batch[:0]

	return nil
}

func (c *WebSocketTickerConsumer) deduplicateBatch(batch []*market_data.Ticker) []market_data.Ticker {
	if len(batch) == 0 {
		return []market_data.Ticker{}
	}

	seen := make(map[string]*market_data.Ticker, len(batch))

	for _, ticker := range batch {
		key := ticker.Exchange + "_" + ticker.Symbol + "_" + strconv.FormatInt(ticker.Timestamp.Unix(), 10)

		existing, exists := seen[key]
		if !exists {
			seen[key] = ticker
		} else {
			if ticker.EventTime.After(existing.EventTime) {
				seen[key] = ticker
			}
		}
	}

	result := make([]market_data.Ticker, 0, len(seen))
	for _, ticker := range seen {
		result = append(result, *ticker)
	}

	return result
}

func (c *WebSocketTickerConsumer) periodicFlush(ctx context.Context, ticker <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker:
			if err := c.flushBatch(ctx); err != nil {
				c.log.Error("Periodic flush failed", "error", err)
			}
		}
	}
}

func (c *WebSocketTickerConsumer) periodicStatsLog(ctx context.Context, ticker <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker:
			c.logStats(false)
		}
	}
}

func (c *WebSocketTickerConsumer) logStats(isFinal bool) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	prefix := "📊"
	if isFinal {
		prefix = "📊 [FINAL]"
	}

	timeSinceLastLog := time.Since(c.lastStatsLogTime)
	receivedRate := float64(c.receivedSinceLastLog) / timeSinceLastLog.Seconds()
	processedRate := float64(c.processedSinceLastLog) / timeSinceLastLog.Seconds()

	c.log.Info(prefix+" Ticker consumer stats",
		"total_received", c.totalReceived,
		"total_processed", c.totalProcessed,
		"total_deduplicated", c.totalDeduplicated,
		"total_errors", c.totalErrors,
		"pending_batch", len(c.batch),
		"received_per_sec", int64(receivedRate),
		"processed_per_sec", int64(processedRate),
		"time_since_last_flush", time.Since(c.lastFlushTime).Round(time.Second),
	)

	c.receivedSinceLastLog = 0
	c.processedSinceLastLog = 0
	c.lastStatsLogTime = time.Now()
}

func (c *WebSocketTickerConsumer) incrementStat(counter *int64) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	*counter++
}

func (c *WebSocketTickerConsumer) convertProtobufToTicker(event *eventspb.WebSocketTickerEvent) *market_data.Ticker {
	parseFloat := func(s string) float64 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	// Use market_type from event, fallback to "futures" for backward compatibility
	marketType := event.MarketType
	if marketType == "" {
		marketType = "futures"
	}

	return &market_data.Ticker{
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
