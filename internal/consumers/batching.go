package consumers

import (
	"context"
	"time"

	"prometheus/internal/adapters/kafka"
	"prometheus/pkg/logger"
)

// BatchConsumer defines the interface for batch-based consumers
// that accumulate messages and flush them periodically
type BatchConsumer interface {
	// FlushBatch flushes the current batch to storage
	FlushBatch(ctx context.Context) error

	// LogStats logs consumer statistics (final should be true on shutdown)
	LogStats(final bool)
}

// BatchConsumerConfig holds configuration for batch consumer lifecycle
type BatchConsumerConfig struct {
	ConsumerName  string
	FlushInterval time.Duration
	StatsInterval time.Duration
	Logger        *logger.Logger
}

// BatchConsumerLifecycle manages the lifecycle of a batch consumer
// This includes:
// - Managing flush and stats tickers
// - Starting periodic background goroutines
// - Graceful shutdown with final flush and stats
type BatchConsumerLifecycle struct {
	config        BatchConsumerConfig
	flushTicker   *time.Ticker
	statsTicker   *time.Ticker
	kafkaConsumer *kafka.Consumer
	batchConsumer BatchConsumer
}

// NewBatchConsumerLifecycle creates a new batch consumer lifecycle manager
func NewBatchConsumerLifecycle(
	config BatchConsumerConfig,
	kafkaConsumer *kafka.Consumer,
	batchConsumer BatchConsumer,
) *BatchConsumerLifecycle {
	return &BatchConsumerLifecycle{
		config:        config,
		kafkaConsumer: kafkaConsumer,
		batchConsumer: batchConsumer,
	}
}

// Start initializes tickers and returns cleanup function
// Usage:
//
//	lifecycle := NewBatchConsumerLifecycle(config, consumer, batchConsumer)
//	cleanup := lifecycle.Start(ctx)
//	defer cleanup()
//
//	// Start background goroutines
//	lifecycle.StartBackgroundWorkers(ctx)
//
//	// ... process messages ...
func (l *BatchConsumerLifecycle) Start(ctx context.Context) func() {
	l.config.Logger.Infow("Starting batch consumer lifecycle",
		"consumer", l.config.ConsumerName,
		"flush_interval", l.config.FlushInterval,
		"stats_interval", l.config.StatsInterval,
	)

	// Create tickers
	l.flushTicker = time.NewTicker(l.config.FlushInterval)
	l.statsTicker = time.NewTicker(l.config.StatsInterval)

	// Return cleanup function that will be called on defer
	return func() {
		l.config.Logger.Infow("Closing batch consumer", "consumer", l.config.ConsumerName)

		// Stop tickers first
		if l.flushTicker != nil {
			l.flushTicker.Stop()
		}
		if l.statsTicker != nil {
			l.statsTicker.Stop()
		}

		// Final flush with background context (main ctx is cancelled)
		if err := l.batchConsumer.FlushBatch(context.Background()); err != nil {
			l.config.Logger.Error("Failed to flush final batch",
				"consumer", l.config.ConsumerName,
				"error", err,
			)
		}

		// Log final stats
		l.batchConsumer.LogStats(true)

		// Close Kafka consumer
		if err := l.kafkaConsumer.Close(); err != nil {
			l.config.Logger.Error("Failed to close Kafka consumer",
				"consumer", l.config.ConsumerName,
				"error", err,
			)
		} else {
			l.config.Logger.Infow("✓ Batch consumer closed",
				"consumer", l.config.ConsumerName,
			)
		}
	}
}

// StartBackgroundWorkers starts periodic flush and stats logging goroutines
func (l *BatchConsumerLifecycle) StartBackgroundWorkers(ctx context.Context) {
	go l.periodicFlush(ctx)
	go l.periodicStatsLog(ctx)
}

// periodicFlush flushes batches at regular intervals
func (l *BatchConsumerLifecycle) periodicFlush(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-l.flushTicker.C:
			if err := l.batchConsumer.FlushBatch(ctx); err != nil {
				l.config.Logger.Error("Periodic flush failed",
					"consumer", l.config.ConsumerName,
					"error", err,
				)
			}
		}
	}
}

// periodicStatsLog logs statistics at regular intervals
func (l *BatchConsumerLifecycle) periodicStatsLog(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-l.statsTicker.C:
			l.batchConsumer.LogStats(false)
		}
	}
}
