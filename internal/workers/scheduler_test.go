package workers

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock worker for testing
type mockWorker struct {
	*BaseWorker
	runCount int32
	runFunc  func(ctx context.Context) error
}

func newMockWorker(name string, interval time.Duration, enabled bool) *mockWorker {
	return &mockWorker{
		BaseWorker: NewBaseWorker(name, interval, enabled),
		runFunc:    func(ctx context.Context) error { return nil },
	}
}

func (m *mockWorker) Run(ctx context.Context) error {
	atomic.AddInt32(&m.runCount, 1)
	if m.runFunc != nil {
		return m.runFunc(ctx)
	}
	return nil
}

func (m *mockWorker) GetRunCount() int {
	return int(atomic.LoadInt32(&m.runCount))
}

func TestScheduler_StartStop(t *testing.T) {
	scheduler := NewScheduler()

	worker1 := newMockWorker("test-worker-1", 100*time.Millisecond, true)
	scheduler.RegisterWorker(worker1)

	ctx := context.Background()
	err := scheduler.Start(ctx)
	require.NoError(t, err)
	assert.True(t, scheduler.IsRunning())

	// Let it run for a bit
	time.Sleep(250 * time.Millisecond)

	err = scheduler.Stop()
	require.NoError(t, err)
	assert.False(t, scheduler.IsRunning())

	// Worker should have run at least 2 times (immediate + 2 ticks)
	runCount := worker1.GetRunCount()
	assert.GreaterOrEqual(t, runCount, 2, "Worker should have run at least 2 times")
}

func TestScheduler_GracefulShutdown(t *testing.T) {
	scheduler := NewScheduler()

	// Worker that takes some time to complete
	worker := newMockWorker("slow-worker", 100*time.Millisecond, true)
	worker.runFunc = func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	scheduler.RegisterWorker(worker)

	ctx := context.Background()
	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Let it run once
	time.Sleep(150 * time.Millisecond)

	// Should stop gracefully
	err = scheduler.Stop()
	require.NoError(t, err)
}

func TestScheduler_ContextCancellation(t *testing.T) {
	scheduler := NewScheduler()

	worker := newMockWorker("test-worker", 100*time.Millisecond, true)
	scheduler.RegisterWorker(worker)

	ctx, cancel := context.WithCancel(context.Background())

	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Cancel context
	cancel()

	// Wait a bit for workers to stop
	time.Sleep(200 * time.Millisecond)

	// Stop should work even after context cancellation
	err = scheduler.Stop()
	require.NoError(t, err)
}

func TestScheduler_DisabledWorker(t *testing.T) {
	scheduler := NewScheduler()

	enabledWorker := newMockWorker("enabled-worker", 100*time.Millisecond, true)
	disabledWorker := newMockWorker("disabled-worker", 100*time.Millisecond, false)

	scheduler.RegisterWorker(enabledWorker)
	scheduler.RegisterWorker(disabledWorker)

	ctx := context.Background()
	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Let them run
	time.Sleep(250 * time.Millisecond)

	err = scheduler.Stop()
	require.NoError(t, err)

	// Enabled worker should have run
	assert.Greater(t, enabledWorker.GetRunCount(), 0)

	// Disabled worker should not have run
	assert.Equal(t, 0, disabledWorker.GetRunCount())
}

func TestScheduler_MultipleWorkers(t *testing.T) {
	scheduler := NewScheduler()

	worker1 := newMockWorker("worker-1", 100*time.Millisecond, true)
	worker2 := newMockWorker("worker-2", 100*time.Millisecond, true)
	worker3 := newMockWorker("worker-3", 100*time.Millisecond, true)

	scheduler.RegisterWorker(worker1)
	scheduler.RegisterWorker(worker2)
	scheduler.RegisterWorker(worker3)

	ctx := context.Background()
	err := scheduler.Start(ctx)
	require.NoError(t, err)

	time.Sleep(250 * time.Millisecond)

	err = scheduler.Stop()
	require.NoError(t, err)

	// All workers should have run
	assert.Greater(t, worker1.GetRunCount(), 0)
	assert.Greater(t, worker2.GetRunCount(), 0)
	assert.Greater(t, worker3.GetRunCount(), 0)
}

func TestScheduler_CannotStartTwice(t *testing.T) {
	scheduler := NewScheduler()

	worker := newMockWorker("test-worker", 100*time.Millisecond, true)
	scheduler.RegisterWorker(worker)

	ctx := context.Background()

	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Try to start again
	err = scheduler.Start(ctx)
	assert.Error(t, err)

	scheduler.Stop()
}

func TestScheduler_GetWorkers(t *testing.T) {
	scheduler := NewScheduler()

	worker1 := newMockWorker("worker-1", 100*time.Millisecond, true)
	worker2 := newMockWorker("worker-2", 200*time.Millisecond, false)

	scheduler.RegisterWorker(worker1)
	scheduler.RegisterWorker(worker2)

	workers := scheduler.GetWorkers()
	assert.Len(t, workers, 2)
	assert.Equal(t, "worker-1", workers[0].Name())
	assert.Equal(t, "worker-2", workers[1].Name())
}
