package workers

import (
	"context"
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

// BaseWorker provides common functionality for workers
type BaseWorker struct {
	name     string
	interval time.Duration
	enabled  bool
	log      *logger.Logger
}

// NewBaseWorker creates a new base worker
func NewBaseWorker(name string, interval time.Duration, enabled bool) *BaseWorker {
	return &BaseWorker{
		name:     name,
		interval: interval,
		enabled:  enabled,
		log:      logger.Get(),
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
	return w.enabled
}

// SetEnabled updates the enabled status
func (w *BaseWorker) SetEnabled(enabled bool) {
	w.enabled = enabled
}

// Log returns the logger
func (w *BaseWorker) Log() *logger.Logger {
	return w.log
}
