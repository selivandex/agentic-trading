package clickhouse

import (
	"context"
	"sync"
	"time"

	"prometheus/pkg/logger"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// FlushFunc is the function called to flush a batch of items to ClickHouse
// It receives a batch of items and should perform the actual INSERT
type FlushFunc func(ctx context.Context, batch []interface{}) error

// BatchWriter accumulates items in memory and flushes them to ClickHouse in batches
// This is essential for ClickHouse performance - single row inserts are inefficient
type BatchWriter struct {
	conn      driver.Conn
	flushFunc FlushFunc
	buffer    []interface{}
	mu        sync.Mutex
	log       *logger.Logger

	// Configuration
	maxBatchSize int           // Flush when buffer reaches this size
	maxAge       time.Duration // Flush when oldest item exceeds this age
	tableName    string        // Table name for logging

	// State
	lastFlush time.Time
	ticker    *time.Ticker
	stopCh    chan struct{}
	wg        sync.WaitGroup
	running   bool
}

// BatchWriterConfig contains configuration for BatchWriter
type BatchWriterConfig struct {
	Conn         driver.Conn
	FlushFunc    FlushFunc
	TableName    string
	MaxBatchSize int           // Default: 500
	MaxAge       time.Duration // Default: 5s
}

// NewBatchWriter creates a new batch writer
func NewBatchWriter(cfg BatchWriterConfig) *BatchWriter {
	if cfg.MaxBatchSize <= 0 {
		cfg.MaxBatchSize = 500
	}
	if cfg.MaxAge <= 0 {
		cfg.MaxAge = 5 * time.Second
	}

	bw := &BatchWriter{
		conn:         cfg.Conn,
		flushFunc:    cfg.FlushFunc,
		buffer:       make([]interface{}, 0, cfg.MaxBatchSize),
		maxBatchSize: cfg.MaxBatchSize,
		maxAge:       cfg.MaxAge,
		tableName:    cfg.TableName,
		lastFlush:    time.Now(),
		stopCh:       make(chan struct{}),
		log:          logger.Get().With("component", "batch_writer", "table", cfg.TableName),
	}

	return bw
}

// Start begins the background flush ticker
func (bw *BatchWriter) Start(ctx context.Context) {
	bw.mu.Lock()
	if bw.running {
		bw.mu.Unlock()
		return
	}
	bw.running = true
	bw.ticker = time.NewTicker(bw.maxAge)
	bw.mu.Unlock()

	bw.wg.Add(1)
	go bw.flushLoop(ctx)

	bw.log.Infof("BatchWriter started (maxBatchSize=%d, maxAge=%v)", bw.maxBatchSize, bw.maxAge)
}

// Add adds an item to the buffer
// If buffer reaches maxBatchSize, it will be flushed immediately
func (bw *BatchWriter) Add(ctx context.Context, item interface{}) error {
	bw.mu.Lock()

	bw.buffer = append(bw.buffer, item)
	bufferSize := len(bw.buffer)

	// Check if we need to flush due to size
	shouldFlush := bufferSize >= bw.maxBatchSize

	bw.mu.Unlock()

	// Flush if buffer is full
	if shouldFlush {
		return bw.Flush(ctx)
	}

	return nil
}

// Flush writes all buffered items to ClickHouse
func (bw *BatchWriter) Flush(ctx context.Context) error {
	bw.mu.Lock()

	if len(bw.buffer) == 0 {
		bw.mu.Unlock()
		return nil // Nothing to flush
	}

	// Take ownership of current buffer and create new one
	batch := bw.buffer
	bw.buffer = make([]interface{}, 0, bw.maxBatchSize)
	bw.lastFlush = time.Now()

	bw.mu.Unlock()

	// Flush outside of lock to avoid blocking Add() calls
	start := time.Now()
	err := bw.flushFunc(ctx, batch)
	duration := time.Since(start)

	if err != nil {
		bw.log.Errorf("Failed to flush %d items to %s: %v (took %v)",
			len(batch), bw.tableName, err, duration)
		return err
	}

	bw.log.Debugf("Flushed %d items to %s (took %v)", len(batch), bw.tableName, duration)
	return nil
}

// flushLoop runs in background and flushes periodically
func (bw *BatchWriter) flushLoop(ctx context.Context) {
	defer bw.wg.Done()

	for {
		select {
		case <-ctx.Done():
			// Context cancelled - perform final flush and exit
			bw.log.Info("BatchWriter stopping, performing final flush...")
			if err := bw.Flush(context.Background()); err != nil {
				bw.log.Errorf("Final flush failed: %v", err)
			}
			return

		case <-bw.stopCh:
			// Explicit stop signal
			bw.log.Info("BatchWriter received stop signal, performing final flush...")
			if err := bw.Flush(context.Background()); err != nil {
				bw.log.Errorf("Final flush failed: %v", err)
			}
			return

		case <-bw.ticker.C:
			// Periodic flush
			bw.mu.Lock()
			bufferSize := len(bw.buffer)
			age := time.Since(bw.lastFlush)
			bw.mu.Unlock()

			if bufferSize > 0 {
				bw.log.Debugf("Periodic flush triggered: %d items, age=%v", bufferSize, age)
				if err := bw.Flush(ctx); err != nil {
					bw.log.Errorf("Periodic flush failed: %v", err)
				}
			}
		}
	}
}

// Stop gracefully shuts down the batch writer
// It flushes any remaining items and waits for completion
func (bw *BatchWriter) Stop(ctx context.Context) error {
	bw.mu.Lock()
	if !bw.running {
		bw.mu.Unlock()
		return nil
	}
	bw.running = false
	bw.mu.Unlock()

	// Stop ticker
	if bw.ticker != nil {
		bw.ticker.Stop()
	}

	// Signal stop
	close(bw.stopCh)

	// Wait for flush loop to complete (with timeout)
	done := make(chan struct{})
	go func() {
		bw.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		bw.log.Info("BatchWriter stopped gracefully")
		return nil
	case <-ctx.Done():
		bw.log.Warn("BatchWriter stop timed out")
		return ctx.Err()
	}
}

// BufferSize returns the current buffer size (for monitoring)
func (bw *BatchWriter) BufferSize() int {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return len(bw.buffer)
}

// Stats returns statistics about the batch writer
type BatchWriterStats struct {
	BufferSize   int
	LastFlushAge time.Duration
	MaxBatchSize int
	MaxAge       time.Duration
	Running      bool
}

// GetStats returns current statistics
func (bw *BatchWriter) GetStats() BatchWriterStats {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	return BatchWriterStats{
		BufferSize:   len(bw.buffer),
		LastFlushAge: time.Since(bw.lastFlush),
		MaxBatchSize: bw.maxBatchSize,
		MaxAge:       bw.maxAge,
		Running:      bw.running,
	}
}
