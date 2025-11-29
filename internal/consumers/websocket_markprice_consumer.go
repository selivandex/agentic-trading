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
	// Batch size for mark price writes
	markPriceBatchSize = 100
	// Flush interval if batch is not full
	markPriceFlushInterval = 5 * time.Second
	// Stats logging interval
	markPriceStatsInterval = 1 * time.Minute
)

// WebSocketMarkPriceConsumer consumes mark price events from Kafka and writes to ClickHouse
type WebSocketMarkPriceConsumer struct {
	consumer   *kafka.Consumer
	repository market_data.Repository
	log        *logger.Logger

	// Batching
	mu    sync.Mutex
	batch []*market_data.MarkPrice

	// Statistics
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

// NewWebSocketMarkPriceConsumer creates a new mark price consumer
func NewWebSocketMarkPriceConsumer(
	consumer *kafka.Consumer,
	repository market_data.Repository,
	log *logger.Logger,
) *WebSocketMarkPriceConsumer {
	now := time.Now()
	return &WebSocketMarkPriceConsumer{
		consumer:         consumer,
		repository:       repository,
		log:              log,
		batch:            make([]*market_data.MarkPrice, 0, markPriceBatchSize),
		lastFlushTime:    now,
		lastStatsLogTime: now,
	}
}

// Start begins consuming mark price events
func (c *WebSocketMarkPriceConsumer) Start(ctx context.Context) error {
	c.log.Info("Starting WebSocket mark price consumer...",
		"batch_size", markPriceBatchSize,
		"flush_interval", markPriceFlushInterval,
	)

	// Start periodic flush and stats logging
	flushTicker := time.NewTicker(markPriceFlushInterval)
	defer flushTicker.Stop()

	statsTicker := time.NewTicker(markPriceStatsInterval)
	defer statsTicker.Stop()

	defer func() {
		c.log.Info("Closing WebSocket mark price consumer...")

		// Final flush
		if err := c.flushBatch(context.Background()); err != nil {
			c.log.Error("Failed to flush final batch", "error", err)
		}

		// Final stats
		c.logStats(true)

		if err := c.consumer.Close(); err != nil {
			c.log.Error("Failed to close consumer", "error", err)
		} else {
			c.log.Info("✓ WebSocket mark price consumer closed")
		}
	}()

	// Start background goroutines
	go c.periodicFlush(ctx, flushTicker.C)
	go c.periodicStatsLog(ctx, statsTicker.C)

	c.log.Info("🔄 Starting to read messages from Kafka...",
		"topic", "websocket.markprice",
	)

	// Consume messages from Kafka
	for {
		msg, err := c.consumer.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				c.log.Info("Mark price consumer stopping (context cancelled)")
				return nil
			}
			c.log.Error("Failed to read mark price event", "error", err)
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
		if err := c.handleMarkPriceEvent(processCtx, msg.Value); err != nil {
			c.log.Error("Failed to handle mark price event", "error", err)
			c.incrementStat(&c.totalErrors)
		}
		cancel()

		// Check if we should stop after processing current message
		if ctx.Err() != nil {
			c.log.Info("Mark price consumer stopping after processing current message")
			return nil
		}
	}
}

// handleMarkPriceEvent processes a single mark price event
func (c *WebSocketMarkPriceConsumer) handleMarkPriceEvent(ctx context.Context, data []byte) error {
	var event eventspb.WebSocketMarkPriceEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal mark price event")
	}

	// Convert to domain model
	markPrice := c.convertProtobufToMarkPrice(&event)

	// Add to batch
	c.addToBatch(markPrice)

	c.log.Debug("Added mark price to batch",
		"exchange", event.Exchange,
		"symbol", event.Symbol,
		"mark_price", event.MarkPrice,
		"batch_size", len(c.batch),
	)

	// Flush if batch is full
	if len(c.batch) >= markPriceBatchSize {
		return c.flushBatch(ctx)
	}

	return nil
}

// addToBatch adds a mark price to the batch (thread-safe)
func (c *WebSocketMarkPriceConsumer) addToBatch(mp *market_data.MarkPrice) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.batch = append(c.batch, mp)
}

// flushBatch writes the batch to ClickHouse with deduplication
func (c *WebSocketMarkPriceConsumer) flushBatch(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.batch) == 0 {
		return nil
	}

	originalSize := len(c.batch)
	start := time.Now()

	// Deduplicate batch
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

	// Write to ClickHouse
	if err := c.repository.InsertMarkPrice(ctx, dedupedBatch); err != nil {
		c.log.Error("Failed to insert mark price batch", "error", err)
		return errors.Wrap(err, "insert batch to ClickHouse")
	}

	duration := time.Since(start)

	c.log.Info("✅ Flushed mark price batch to ClickHouse",
		"original_size", originalSize,
		"written_size", len(dedupedBatch),
		"deduplicated", deduplicatedCount,
		"duration_ms", duration.Milliseconds(),
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

// deduplicateBatch keeps only the latest event for each unique (exchange, symbol, timestamp)
func (c *WebSocketMarkPriceConsumer) deduplicateBatch(batch []*market_data.MarkPrice) []market_data.MarkPrice {
	if len(batch) == 0 {
		return []market_data.MarkPrice{}
	}

	// Map: key -> latest MarkPrice
	seen := make(map[string]*market_data.MarkPrice, len(batch))

	for _, mp := range batch {
		// Create unique key: exchange_symbol_timestamp
		key := mp.Exchange + "_" + mp.Symbol + "_" + strconv.FormatInt(mp.Timestamp.Unix(), 10)

		existing, exists := seen[key]
		if !exists {
			seen[key] = mp
		} else {
			// Keep the one with latest event_time
			if mp.EventTime.After(existing.EventTime) {
				seen[key] = mp
			}
		}
	}

	// Convert map back to slice
	result := make([]market_data.MarkPrice, 0, len(seen))
	for _, mp := range seen {
		result = append(result, *mp)
	}

	return result
}

// periodicFlush flushes batch periodically
func (c *WebSocketMarkPriceConsumer) periodicFlush(ctx context.Context, ticker <-chan time.Time) {
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
func (c *WebSocketMarkPriceConsumer) periodicStatsLog(ctx context.Context, ticker <-chan time.Time) {
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
func (c *WebSocketMarkPriceConsumer) logStats(isFinal bool) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	prefix := "📊"
	if isFinal {
		prefix = "📊 [FINAL]"
	}

	timeSinceLastLog := time.Since(c.lastStatsLogTime)
	receivedRate := float64(c.receivedSinceLastLog) / timeSinceLastLog.Seconds()
	processedRate := float64(c.processedSinceLastLog) / timeSinceLastLog.Seconds()

	c.log.Info(prefix+" Mark price consumer stats",
		"total_received", c.totalReceived,
		"total_processed", c.totalProcessed,
		"total_deduplicated", c.totalDeduplicated,
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
func (c *WebSocketMarkPriceConsumer) incrementStat(counter *int64) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	*counter++
}

// convertProtobufToMarkPrice converts protobuf event to domain model
func (c *WebSocketMarkPriceConsumer) convertProtobufToMarkPrice(event *eventspb.WebSocketMarkPriceEvent) *market_data.MarkPrice {
	// Helper to parse string to float64
	parseFloat := func(s string) float64 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	// Mark price is always futures (it doesn't exist for spot)
	return &market_data.MarkPrice{
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
