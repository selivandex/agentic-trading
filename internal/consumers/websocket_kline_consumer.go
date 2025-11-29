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
	// Batch size for ClickHouse writes
	klineBatchSize = 50
	// Flush interval if batch is not full
	klineFlushInterval = 10 * time.Second
	// Stats logging interval
	klineStatsInterval = 1 * time.Minute
)

// WebSocketKlineConsumer consumes kline events from Kafka and writes to ClickHouse in batches
type WebSocketKlineConsumer struct {
	consumer   *kafka.Consumer
	repository market_data.Repository
	log        *logger.Logger

	// Batching
	mu    sync.Mutex
	batch []*market_data.OHLCV

	// Statistics
	statsMu               sync.Mutex
	totalReceived         int64
	totalProcessed        int64
	totalDeduplicated     int64 // Removed during batch dedup
	totalFinal            int64 // Final (closed) candles
	totalNonFinal         int64 // Non-final (updating) candles
	totalErrors           int64
	lastFlushTime         time.Time
	lastStatsLogTime      time.Time
	receivedSinceLastLog  int64
	processedSinceLastLog int64
}

// NewWebSocketKlineConsumer creates a new WebSocket kline consumer
func NewWebSocketKlineConsumer(
	consumer *kafka.Consumer,
	repository market_data.Repository,
	log *logger.Logger,
) *WebSocketKlineConsumer {
	now := time.Now()
	return &WebSocketKlineConsumer{
		consumer:         consumer,
		repository:       repository,
		log:              log,
		batch:            make([]*market_data.OHLCV, 0, klineBatchSize),
		lastFlushTime:    now,
		lastStatsLogTime: now,
	}
}

// Start begins consuming kline events
func (c *WebSocketKlineConsumer) Start(ctx context.Context) error {
	c.log.Info("Starting WebSocket kline consumer...",
		"batch_size", klineBatchSize,
		"flush_interval", klineFlushInterval,
	)

	// Start periodic flush goroutine
	flushTicker := time.NewTicker(klineFlushInterval)
	defer flushTicker.Stop()

	// Start periodic stats logging goroutine
	statsTicker := time.NewTicker(klineStatsInterval)
	defer statsTicker.Stop()

	// Ensure consumer is closed on exit
	defer func() {
		c.log.Info("Closing WebSocket kline consumer...")

		// Final flush
		if err := c.flushBatch(context.Background()); err != nil {
			c.log.Error("Failed to flush final batch", "error", err)
		}

		// Final stats
		c.logStats(true)

		if err := c.consumer.Close(); err != nil {
			c.log.Error("Failed to close consumer", "error", err)
		} else {
			c.log.Info("✓ WebSocket kline consumer closed")
		}
	}()

	// Start background goroutines
	go c.periodicFlush(ctx, flushTicker.C)
	go c.periodicStatsLog(ctx, statsTicker.C)

	c.log.Info("🔄 Starting to read messages from Kafka...",
		"topic", "websocket.kline",
	)

	// Consume messages from Kafka
	for {
		msg, err := c.consumer.ReadMessage(ctx)
		if err != nil {
			// Check if error is due to context cancellation
			if ctx.Err() != nil {
				c.log.Info("WebSocket kline consumer stopping (context cancelled)")
				return nil
			}
			c.log.Error("Failed to read kline event", "error", err)
			continue
		}

		c.incrementStat(&c.totalReceived)
		c.incrementStat(&c.receivedSinceLastLog)

		c.log.Debug("📩 Received message from Kafka",
			"partition", msg.Partition,
			"offset", msg.Offset,
			"size_bytes", len(msg.Value),
		)

		// Process event with timeout
		processCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := c.handleKlineEvent(processCtx, msg.Value); err != nil {
			c.log.Error("Failed to handle kline event",
				"topic", msg.Topic,
				"error", err,
			)
			c.incrementStat(&c.totalErrors)
		}
		cancel()

		// Check if we should stop after processing current message
		if ctx.Err() != nil {
			c.log.Info("WebSocket kline consumer stopping after processing current message")
			return nil
		}
	}
}

// handleKlineEvent processes a single kline event
func (c *WebSocketKlineConsumer) handleKlineEvent(ctx context.Context, data []byte) error {
	var event eventspb.WebSocketKlineEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal kline event")
	}

	// Track stats by candle type
	if event.IsFinal {
		c.incrementStat(&c.totalFinal)
	} else {
		c.incrementStat(&c.totalNonFinal)
	}

	// Convert to domain model
	ohlcv := c.convertProtobufToOHLCV(&event)

	// Add to batch (deduplication happens before flush)
	c.addToBatch(ohlcv)

	c.log.Debug("Added kline to batch",
		"exchange", event.Exchange,
		"symbol", event.Symbol,
		"interval", event.Interval,
		"is_final", event.IsFinal,
		"batch_size", len(c.batch),
	)

	// Flush if batch is full
	if len(c.batch) >= klineBatchSize {
		return c.flushBatch(ctx)
	}

	return nil
}

// addToBatch adds a kline to the batch (thread-safe)
func (c *WebSocketKlineConsumer) addToBatch(ohlcv *market_data.OHLCV) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.batch = append(c.batch, ohlcv)
}

// flushBatch writes the batch to ClickHouse with deduplication
func (c *WebSocketKlineConsumer) flushBatch(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.batch) == 0 {
		return nil
	}

	originalSize := len(c.batch)
	start := time.Now()

	// Deduplicate batch: keep only latest event for each (exchange, symbol, interval, open_time)
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

	// Write deduplicated batch to ClickHouse
	if err := c.repository.InsertOHLCV(ctx, c.convertBatchToValues(dedupedBatch)); err != nil {
		c.log.Error("Failed to insert batch to ClickHouse",
			"error", err,
			"batch_size", len(dedupedBatch),
		)
		return errors.Wrap(err, "insert batch to ClickHouse")
	}

	duration := time.Since(start)

	// Count final vs non-final in deduplicated batch
	finalCount := 0
	for _, ohlcv := range dedupedBatch {
		if ohlcv.IsClosed {
			finalCount++
		}
	}

	c.log.Info("✅ Flushed kline batch to ClickHouse",
		"original_size", originalSize,
		"written_size", len(dedupedBatch),
		"deduplicated", deduplicatedCount,
		"final_candles", finalCount,
		"updating_candles", len(dedupedBatch)-finalCount,
		"duration_ms", duration.Milliseconds(),
		"candles_per_sec", int64(float64(len(dedupedBatch))/duration.Seconds()),
	)

	// Update stats
	c.statsMu.Lock()
	c.totalProcessed += int64(len(dedupedBatch))
	c.processedSinceLastLog += int64(len(dedupedBatch))
	c.lastFlushTime = time.Now()
	c.statsMu.Unlock()

	// Clear batch
	c.batch = c.batch[:0]

	return nil
}

// deduplicateBatch keeps only the latest event for each unique kline
// For same (exchange, symbol, interval, open_time), keeps the one with latest event_time
func (c *WebSocketKlineConsumer) deduplicateBatch(batch []*market_data.OHLCV) []*market_data.OHLCV {
	if len(batch) == 0 {
		return batch
	}

	// Map: key -> latest OHLCV
	seen := make(map[string]*market_data.OHLCV, len(batch))

	for _, ohlcv := range batch {
		// Create unique key: exchange_symbol_interval_opentime
		key := ohlcv.Exchange + "_" + ohlcv.Symbol + "_" + ohlcv.Timeframe + "_" +
			strconv.FormatInt(ohlcv.OpenTime.Unix(), 10)

		existing, exists := seen[key]
		if !exists {
			// First time seeing this kline
			seen[key] = ohlcv
		} else {
			// Keep the one with latest event_time (most recent update)
			if ohlcv.EventTime.After(existing.EventTime) {
				seen[key] = ohlcv
			}
		}
	}

	// Convert map back to slice
	result := make([]*market_data.OHLCV, 0, len(seen))
	for _, ohlcv := range seen {
		result = append(result, ohlcv)
	}

	return result
}

// convertBatchToValues converts batch of pointers to values
func (c *WebSocketKlineConsumer) convertBatchToValues(batch []*market_data.OHLCV) []market_data.OHLCV {
	values := make([]market_data.OHLCV, len(batch))
	for i, ohlcv := range batch {
		values[i] = *ohlcv
	}
	return values
}

// periodicFlush flushes batch periodically
func (c *WebSocketKlineConsumer) periodicFlush(ctx context.Context, ticker <-chan time.Time) {
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

// periodicStatsLog logs statistics periodically
func (c *WebSocketKlineConsumer) periodicStatsLog(ctx context.Context, ticker <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker:
			c.logStats(false)
		}
	}
}

// logStats logs current statistics
func (c *WebSocketKlineConsumer) logStats(isFinal bool) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	prefix := "📊"
	if isFinal {
		prefix = "📊 [FINAL]"
	}

	timeSinceLastLog := time.Since(c.lastStatsLogTime)
	receivedRate := float64(c.receivedSinceLastLog) / timeSinceLastLog.Seconds()
	processedRate := float64(c.processedSinceLastLog) / timeSinceLastLog.Seconds()

	finalPercent := 0.0
	if c.totalProcessed > 0 {
		finalPercent = float64(c.totalFinal) / float64(c.totalProcessed) * 100
	}

	c.log.Info(prefix+" WebSocket kline consumer stats",
		"total_received", c.totalReceived,
		"total_processed", c.totalProcessed,
		"total_deduplicated", c.totalDeduplicated,
		"total_final", c.totalFinal,
		"total_non_final", c.totalNonFinal,
		"final_percent", int64(finalPercent),
		"total_errors", c.totalErrors,
		"pending_batch", len(c.batch),
		"received_per_sec", int64(receivedRate),
		"processed_per_sec", int64(processedRate),
		"time_since_last_flush", time.Since(c.lastFlushTime).Round(time.Second),
	)

	// Reset interval counters
	c.receivedSinceLastLog = 0
	c.processedSinceLastLog = 0
	c.lastStatsLogTime = time.Now()
}

// incrementStat safely increments a stat counter
func (c *WebSocketKlineConsumer) incrementStat(counter *int64) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	*counter++
}

// convertProtobufToOHLCV converts protobuf WebSocket kline event to domain model
func (c *WebSocketKlineConsumer) convertProtobufToOHLCV(event *eventspb.WebSocketKlineEvent) *market_data.OHLCV {
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

	return &market_data.OHLCV{
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
