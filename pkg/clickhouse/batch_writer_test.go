package clickhouse

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchWriter_FlushOnMaxSize(t *testing.T) {
	flushed := make([][]interface{}, 0)
	var mu sync.Mutex

	flushFunc := func(ctx context.Context, batch []interface{}) error {
		mu.Lock()
		defer mu.Unlock()
		flushed = append(flushed, batch)
		return nil
	}

	bw := NewBatchWriter(BatchWriterConfig{
		Conn:         nil, // Not needed for this test
		FlushFunc:    flushFunc,
		TableName:    "test_table",
		MaxBatchSize: 3,
		MaxAge:       10 * time.Second, // Long enough to not trigger
	})

	ctx := context.Background()

	// Add 3 items - should trigger flush
	require.NoError(t, bw.Add(ctx, "item1"))
	require.NoError(t, bw.Add(ctx, "item2"))
	require.NoError(t, bw.Add(ctx, "item3"))

	// Verify flush happened
	mu.Lock()
	assert.Equal(t, 1, len(flushed), "Should have flushed once")
	assert.Equal(t, 3, len(flushed[0]), "Batch should contain 3 items")
	mu.Unlock()

	// Buffer should be empty after flush
	assert.Equal(t, 0, bw.BufferSize())
}

func TestBatchWriter_FlushOnTimer(t *testing.T) {
	flushed := make([][]interface{}, 0)
	var mu sync.Mutex

	flushFunc := func(ctx context.Context, batch []interface{}) error {
		mu.Lock()
		defer mu.Unlock()
		flushed = append(flushed, batch)
		return nil
	}

	bw := NewBatchWriter(BatchWriterConfig{
		Conn:         nil,
		FlushFunc:    flushFunc,
		TableName:    "test_table",
		MaxBatchSize: 100,            // High enough to not trigger by size
		MaxAge:       100 * time.Millisecond, // Short interval for testing
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background flush loop
	bw.Start(ctx)

	// Add items (not enough to trigger size flush)
	require.NoError(t, bw.Add(ctx, "item1"))
	require.NoError(t, bw.Add(ctx, "item2"))

	// Wait for timer flush
	time.Sleep(200 * time.Millisecond)

	// Verify flush happened
	mu.Lock()
	assert.GreaterOrEqual(t, len(flushed), 1, "Should have flushed at least once")
	if len(flushed) > 0 {
		assert.Equal(t, 2, len(flushed[0]), "Batch should contain 2 items")
	}
	mu.Unlock()

	// Stop gracefully
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	require.NoError(t, bw.Stop(stopCtx))
}

func TestBatchWriter_GracefulStop(t *testing.T) {
	flushed := make([][]interface{}, 0)
	var mu sync.Mutex

	flushFunc := func(ctx context.Context, batch []interface{}) error {
		mu.Lock()
		defer mu.Unlock()
		flushed = append(flushed, batch)
		return nil
	}

	bw := NewBatchWriter(BatchWriterConfig{
		Conn:         nil,
		FlushFunc:    flushFunc,
		TableName:    "test_table",
		MaxBatchSize: 100,
		MaxAge:       10 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bw.Start(ctx)

	// Add items
	require.NoError(t, bw.Add(ctx, "item1"))
	require.NoError(t, bw.Add(ctx, "item2"))
	require.NoError(t, bw.Add(ctx, "item3"))

	// Stop should flush remaining items
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	require.NoError(t, bw.Stop(stopCtx))

	// Verify final flush happened
	mu.Lock()
	assert.GreaterOrEqual(t, len(flushed), 1, "Should have flushed on stop")
	totalItems := 0
	for _, batch := range flushed {
		totalItems += len(batch)
	}
	assert.Equal(t, 3, totalItems, "All items should be flushed")
	mu.Unlock()
}

func TestBatchWriter_ConcurrentAdds(t *testing.T) {
	flushed := make([][]interface{}, 0)
	var mu sync.Mutex

	flushFunc := func(ctx context.Context, batch []interface{}) error {
		mu.Lock()
		defer mu.Unlock()
		flushed = append(flushed, batch)
		return nil
	}

	bw := NewBatchWriter(BatchWriterConfig{
		Conn:         nil,
		FlushFunc:    flushFunc,
		TableName:    "test_table",
		MaxBatchSize: 10,
		MaxAge:       1 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bw.Start(ctx)

	// Add items concurrently
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = bw.Add(ctx, idx)
		}(i)
	}

	wg.Wait()

	// Stop and verify all items were flushed
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	require.NoError(t, bw.Stop(stopCtx))

	// Count total flushed items
	mu.Lock()
	totalItems := 0
	for _, batch := range flushed {
		totalItems += len(batch)
	}
	mu.Unlock()

	assert.Equal(t, 50, totalItems, "All 50 items should be flushed")
}



