package workers

import (
	"context"
	"sync"
	"time"

	"prometheus/pkg/logger"
)

// Worker defines the interface for background workers
type Worker interface {
	// Name returns the unique identifier for this worker
	Name() string

	// Run executes the worker's task
	// It should complete one iteration of work and return
	// The scheduler will call this repeatedly based on Interval()
	Run(ctx context.Context) error

	// Interval returns how often this worker should run
	Interval() time.Duration

	// Enabled returns whether this worker is active
	Enabled() bool
}

// WorkerWithHealth extends Worker with health monitoring capabilities
type WorkerWithHealth interface {
	Worker
	Health() WorkerHealth
	SetEnabled(enabled bool)
}

// WorkerHealth contains health information for a worker
type WorkerHealth struct {
	LastRun     time.Time
	LastError   error
	RunCount    int64
	ErrorCount  int64
	AvgDuration time.Duration
	IsRunning   bool
	Enabled     bool
}

// BaseWorker provides common functionality for workers
type BaseWorker struct {
	name     string
	interval time.Duration
	enabled  bool
	log      *logger.Logger

	// Health monitoring
	healthMu      sync.RWMutex
	lastRun       time.Time
	lastError     error
	runCount      int64
	errorCount    int64
	totalDuration time.Duration
}

// NewBaseWorker creates a new base worker
func NewBaseWorker(name string, interval time.Duration, enabled bool) *BaseWorker {
	return &BaseWorker{
		name:     name,
		interval: interval,
		enabled:  enabled,
		log:      logger.Get().With("worker", name),
	}
}

// Name returns the worker name
func (w *BaseWorker) Name() string {
	return w.name
}

// Interval returns the run interval
func (w *BaseWorker) Interval() time.Duration {
	return w.interval
}

// Enabled returns whether the worker is enabled
func (w *BaseWorker) Enabled() bool {
	w.healthMu.RLock()
	defer w.healthMu.RUnlock()
	return w.enabled
}

// SetEnabled updates the enabled status
func (w *BaseWorker) SetEnabled(enabled bool) {
	w.healthMu.Lock()
	defer w.healthMu.Unlock()
	w.enabled = enabled
	w.log.Infof("Worker enabled state changed to: %v", enabled)
}

// Log returns the logger
func (w *BaseWorker) Log() *logger.Logger {
	return w.log
}

// Health returns health information for the worker
func (w *BaseWorker) Health() WorkerHealth {
	w.healthMu.RLock()
	defer w.healthMu.RUnlock()

	avgDuration := time.Duration(0)
	if w.runCount > 0 {
		avgDuration = time.Duration(int64(w.totalDuration) / w.runCount)
	}

	return WorkerHealth{
		LastRun:     w.lastRun,
		LastError:   w.lastError,
		RunCount:    w.runCount,
		ErrorCount:  w.errorCount,
		AvgDuration: avgDuration,
		IsRunning:   false, // Updated by scheduler
		Enabled:     w.enabled,
	}
}

// RecordRun records a successful run
func (w *BaseWorker) RecordRun(duration time.Duration) {
	w.healthMu.Lock()
	defer w.healthMu.Unlock()

	w.lastRun = time.Now()
	w.runCount++
	w.totalDuration += duration
	w.lastError = nil
}

// RecordError records a failed run
func (w *BaseWorker) RecordError(err error, duration time.Duration) {
	w.healthMu.Lock()
	defer w.healthMu.Unlock()

	w.lastRun = time.Now()
	w.runCount++
	w.errorCount++
	w.totalDuration += duration
	w.lastError = err
}
