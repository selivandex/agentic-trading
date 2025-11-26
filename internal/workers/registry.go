package workers

import (
	"prometheus/pkg/errors"
	"sync"
	"time"
)

// Registry manages all workers in the system
type Registry struct {
	workers map[string]WorkerWithHealth
	health  map[string]*WorkerHealth
	mu      sync.RWMutex
}

// NewRegistry creates a new worker registry
func NewRegistry() *Registry {
	return &Registry{
		workers: make(map[string]WorkerWithHealth),
		health:  make(map[string]*WorkerHealth),
	}
}

// Register adds a worker to the registry
func (r *Registry) Register(w WorkerWithHealth) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := w.Name()
	if _, exists := r.workers[name]; exists {
		return errors.Wrapf(errors.ErrAlreadyExists, "worker %s already registered", name)
	}

	r.workers[name] = w
	r.health[name] = &WorkerHealth{
		Enabled: w.Enabled(),
	}

	return nil
}

// Unregister removes a worker from the registry
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.workers[name]; !exists {
		return errors.Wrapf(errors.ErrNotFound, "worker %s not found", name)
	}

	delete(r.workers, name)
	delete(r.health, name)

	return nil
}

// Get returns a worker by name
func (r *Registry) Get(name string) (WorkerWithHealth, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	w, ok := r.workers[name]
	return w, ok
}

// List returns all registered workers
func (r *Registry) List() []WorkerWithHealth {
	r.mu.RLock()
	defer r.mu.RUnlock()

	workers := make([]WorkerWithHealth, 0, len(r.workers))
	for _, w := range r.workers {
		workers = append(workers, w)
	}

	return workers
}

// ListNames returns names of all registered workers
func (r *Registry) ListNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.workers))
	for name := range r.workers {
		names = append(names, name)
	}

	return names
}

// EnableWorker enables or disables a worker by name
func (r *Registry) EnableWorker(name string, enabled bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	w, ok := r.workers[name]
	if !ok {
		return errors.Wrapf(errors.ErrNotFound, "worker %s not found", name)
	}

	w.SetEnabled(enabled)

	if h, ok := r.health[name]; ok {
		h.Enabled = enabled
	}

	return nil
}

// GetHealth returns health information for a worker
func (r *Registry) GetHealth(name string) (*WorkerHealth, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	h, ok := r.health[name]
	return h, ok
}

// UpdateHealth updates health information for a worker
func (r *Registry) UpdateHealth(name string, update func(*WorkerHealth)) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	h, ok := r.health[name]
	if !ok {
		return errors.Wrapf(errors.ErrNotFound, "worker %s not found", name)
	}

	update(h)
	return nil
}

// GetAllHealth returns health information for all workers
func (r *Registry) GetAllHealth() map[string]WorkerHealth {
	r.mu.RLock()
	defer r.mu.RUnlock()

	health := make(map[string]WorkerHealth, len(r.health))
	for name, h := range r.health {
		health[name] = *h
	}

	return health
}

// RecordRun records a successful worker run
func (r *Registry) RecordRun(name string, duration time.Duration) error {
	return r.UpdateHealth(name, func(h *WorkerHealth) {
		h.LastRun = time.Now()
		h.RunCount++
		h.IsRunning = false

		// Update average duration (simple moving average)
		if h.AvgDuration == 0 {
			h.AvgDuration = duration
		} else {
			// Weighted average: 80% old, 20% new
			h.AvgDuration = time.Duration(
				int64(float64(h.AvgDuration)*0.8 + float64(duration)*0.2),
			)
		}
	})
}

// RecordError records a worker error
func (r *Registry) RecordError(name string, err error) error {
	return r.UpdateHealth(name, func(h *WorkerHealth) {
		h.LastRun = time.Now()
		h.LastError = err
		h.ErrorCount++
		h.IsRunning = false
	})
}

// MarkRunning marks a worker as currently running
func (r *Registry) MarkRunning(name string) error {
	return r.UpdateHealth(name, func(h *WorkerHealth) {
		h.IsRunning = true
	})
}

// GetUnhealthyWorkers returns workers that appear unhealthy
func (r *Registry) GetUnhealthyWorkers(maxAge time.Duration) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var unhealthy []string
	now := time.Now()

	for name, h := range r.health {
		if !h.Enabled {
			continue
		}

		// Worker hasn't run recently
		if now.Sub(h.LastRun) > maxAge {
			unhealthy = append(unhealthy, name)
			continue
		}

		// Worker has high error rate
		if h.RunCount > 10 {
			errorRate := float64(h.ErrorCount) / float64(h.RunCount)
			if errorRate > 0.5 {
				unhealthy = append(unhealthy, name)
			}
		}
	}

	return unhealthy
}

// Clear removes all workers from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.workers = make(map[string]WorkerWithHealth)
	r.health = make(map[string]*WorkerHealth)
}

// Count returns the number of registered workers
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.workers)
}

// CountEnabled returns the number of enabled workers
func (r *Registry) CountEnabled() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, h := range r.health {
		if h.Enabled {
			count++
		}
	}

	return count
}
