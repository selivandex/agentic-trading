package workers

import (
	"context"
	"sync"
	"time"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Scheduler manages and coordinates multiple workers
type Scheduler struct {
	workers []Worker
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.RWMutex
	log     *logger.Logger
	started bool
}

// NewScheduler creates a new worker scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		workers: make([]Worker, 0),
		log:     logger.Get(),
		started: false,
	}
}

// RegisterWorker adds a worker to the scheduler
func (s *Scheduler) RegisterWorker(w Worker) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		s.log.Warn("Cannot register worker after scheduler has started", "worker", w.Name())
		return
	}

	s.workers = append(s.workers, w)
	s.log.Info("Worker registered", "worker", w.Name(), "interval", w.Interval())
}

// Start begins running all registered workers
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return errors.Wrapf(errors.ErrInternal, "scheduler already started")
	}

	s.started = true
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.mu.Unlock()

	s.log.Info("Starting worker scheduler", "workers", len(s.workers))

	// Start each enabled worker in its own goroutine
	for _, worker := range s.workers {
		if !worker.Enabled() {
			s.log.Info("Skipping disabled worker", "worker", worker.Name())
			continue
		}

		s.wg.Add(1)
		go s.runWorker(worker)
	}

	s.log.Info("All workers started")
	return nil
}

// Stop gracefully shuts down all workers
// Uses 2-minute timeout to allow long-running operations (e.g., ADK workflows) to complete
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return errors.Wrapf(errors.ErrInternal, "scheduler not started")
	}

	// Cancel context to signal all workers to stop
	s.cancel()

	// Release lock before waiting to allow workers to check s.started if needed
	s.mu.Unlock()

	s.log.Info("Stopping worker scheduler...")

	// Wait for all workers to finish with timeout
	// 2-minute timeout accommodates long-running operations like:
	// - OpportunityFinder ADK workflows (can take 2-3 minutes per symbol)
	// - Large batch processing in other workers
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	var shutdownErr error
	select {
	case <-done:
		s.log.Info("All workers stopped gracefully")
	case <-time.After(2 * time.Minute):
		s.log.Warn("Worker shutdown timed out after 2 minutes")
		shutdownErr = errors.Wrapf(errors.ErrInternal, "shutdown timeout after 2 minutes")
	}

	// Re-acquire lock to update state
	s.mu.Lock()
	s.started = false
	s.mu.Unlock()

	return shutdownErr
}

// runWorker executes a single worker in a loop
func (s *Scheduler) runWorker(worker Worker) {
	defer s.wg.Done()

	s.log.Info("Worker started", "worker", worker.Name())

	ticker := time.NewTicker(worker.Interval())
	defer ticker.Stop()

	// Run immediately on start
	s.executeWorker(worker)

	for {
		select {
		case <-s.ctx.Done():
			s.log.Info("Worker stopping due to context cancellation", "worker", worker.Name())
			return

		case <-ticker.C:
			s.executeWorker(worker)
		}
	}
}

// executeWorker runs a single iteration of the worker with error handling
func (s *Scheduler) executeWorker(worker Worker) {
	start := time.Now()

	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("Worker panicked",
				"worker", worker.Name(),
				"panic", r,
			)
		}
	}()

	if err := worker.Run(s.ctx); err != nil {
		s.log.Error("Worker execution failed",
			"worker", worker.Name(),
			"error", err,
			"duration", time.Since(start),
		)
	} else {
		s.log.Debug("Worker execution completed",
			"worker", worker.Name(),
			"duration", time.Since(start),
		)
	}
}

// GetWorkers returns a list of all registered workers (for debugging/monitoring)
func (s *Scheduler) GetWorkers() []Worker {
	s.mu.RLock()
	defer s.mu.RUnlock()

	workers := make([]Worker, len(s.workers))
	copy(workers, s.workers)
	return workers
}

// IsRunning returns whether the scheduler is currently running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.started
}
