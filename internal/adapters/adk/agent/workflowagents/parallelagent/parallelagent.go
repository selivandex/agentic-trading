package parallelagent

import (
	"context"
	"sync"

	"google.golang.org/adk/agent"
)

// Config holds the metadata for the parallel workflow agent.
type Config struct {
	agent.Config
}

// Agent executes subagents concurrently and merges their outputs.
type Agent struct {
	cfg agent.Config
}

// New constructs a new parallel workflow agent.
func New(cfg Config) (*Agent, error) {
	return &Agent{cfg: cfg.Config}, nil
}

// Name returns the agent name.
func (a *Agent) Name() string { return a.cfg.Name }

// Description returns a description.
func (a *Agent) Description() string { return a.cfg.Description }

// Run executes subagents concurrently and merges their outputs.
func (a *Agent) Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	var wg sync.WaitGroup
	mu := sync.Mutex{}
	results := map[string]interface{}{}
	var firstErr error

	wg.Add(len(a.cfg.SubAgents))
	for _, sub := range a.cfg.SubAgents {
		sub := sub
		go func() {
			defer wg.Done()
			if sub == nil {
				return
			}
			out, err := sub.Run(ctx, input)
			mu.Lock()
			defer mu.Unlock()
			if err != nil && firstErr == nil {
				firstErr = err
			}
			for k, v := range out {
				results[k] = v
			}
		}()
	}

	wg.Wait()
	return results, firstErr
}

var _ agent.Agent = (*Agent)(nil)
