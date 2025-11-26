package agents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/adk/agent"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// PipelineStrategy defines how to handle errors in a pipeline
type PipelineStrategy string

const (
	// StrategyFailFast stops pipeline on first error
	StrategyFailFast PipelineStrategy = "fail_fast"

	// StrategyContinueOnError continues pipeline even if agents fail
	StrategyContinueOnError PipelineStrategy = "continue_on_error"

	// StrategyPartialResults returns partial results on error
	StrategyPartialResults PipelineStrategy = "partial_results"
)

// PipelineConfig configures a pipeline
type PipelineConfig struct {
	Name           string
	Strategy       PipelineStrategy
	Timeout        time.Duration
	MaxConcurrency int // For parallel pipelines
}

// PipelineResult contains results from a pipeline execution
type PipelineResult struct {
	Results map[string]interface{} // Agent name -> output
	Errors  map[string]error       // Agent name -> error
	Partial bool                   // True if some agents failed
	Started time.Time
	Ended   time.Time
}

// Duration returns the total pipeline execution time
func (r *PipelineResult) Duration() time.Duration {
	return r.Ended.Sub(r.Started)
}

// Success returns true if all agents succeeded
func (r *PipelineResult) Success() bool {
	return len(r.Errors) == 0
}

// HasResult checks if a specific agent produced a result
func (r *PipelineResult) HasResult(agentName string) bool {
	_, ok := r.Results[agentName]
	return ok
}

// GetResult returns a result for a specific agent
func (r *PipelineResult) GetResult(agentName string) (interface{}, bool) {
	result, ok := r.Results[agentName]
	return result, ok
}

// GetError returns an error for a specific agent
func (r *PipelineResult) GetError(agentName string) (error, bool) {
	err, ok := r.Errors[agentName]
	return err, ok
}

// Pipeline executes a sequence or parallel group of agents
type Pipeline struct {
	config PipelineConfig
	agents []agent.Agent
	log    *logger.Logger
}

// NewPipeline creates a new pipeline
func NewPipeline(config PipelineConfig, agents []agent.Agent) *Pipeline {
	if config.Strategy == "" {
		config.Strategy = StrategyPartialResults
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}
	if config.MaxConcurrency == 0 {
		config.MaxConcurrency = 5
	}

	return &Pipeline{
		config: config,
		agents: agents,
		log:    logger.Get().With("component", "pipeline", "pipeline", config.Name),
	}
}

// ExecuteSequential runs agents one after another
func (p *Pipeline) ExecuteSequential(ctx context.Context, input interface{}) (*PipelineResult, error) {
	result := &PipelineResult{
		Results: make(map[string]interface{}),
		Errors:  make(map[string]error),
		Started: time.Now(),
	}
	defer func() { result.Ended = time.Now() }()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	defer cancel()

	// Execute agents sequentially
	currentInput := input

	for _, ag := range p.agents {
		agentName := getAgentName(ag)

		p.log.Debugf("Executing agent: %s", agentName)
		startTime := time.Now()

		// Execute agent
		output, err := p.executeAgent(ctx, ag, currentInput)
		duration := time.Since(startTime)

		if err != nil {
			p.log.Errorf("Agent %s failed: %v (duration: %s)", agentName, err, duration)
			result.Errors[agentName] = err
			result.Partial = true

			// Handle error based on strategy
			switch p.config.Strategy {
			case StrategyFailFast:
				return result, errors.Wrapf(err, "pipeline failed at %s", agentName)

			case StrategyContinueOnError:
				// Continue to next agent even on error
				continue

			case StrategyPartialResults:
				// Continue but mark as partial
				continue
			}
		}

		// Store result
		result.Results[agentName] = output
		p.log.Debugf("Agent %s completed successfully (duration: %s)", agentName, duration)

		// Use output as input for next agent
		currentInput = output
	}

	// Check if we have any results
	if len(result.Results) == 0 && len(result.Errors) > 0 {
		return result, errors.Wrapf(errors.ErrInternal, "all agents failed")
	}

	return result, nil
}

// ExecuteParallel runs agents concurrently
func (p *Pipeline) ExecuteParallel(ctx context.Context, input interface{}) (*PipelineResult, error) {
	result := &PipelineResult{
		Results: make(map[string]interface{}),
		Errors:  make(map[string]error),
		Started: time.Now(),
	}
	defer func() { result.Ended = time.Now() }()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	defer cancel()

	// Execute agents in parallel
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, p.config.MaxConcurrency)
	resultMu := sync.Mutex{}

	for _, ag := range p.agents {
		wg.Add(1)
		go func(agent agent.Agent) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			agentName := getAgentName(agent)
			p.log.Debugf("Executing agent: %s", agentName)
			startTime := time.Now()

			// Execute agent
			output, err := p.executeAgent(ctx, agent, input)
			duration := time.Since(startTime)

			// Store result
			resultMu.Lock()
			defer resultMu.Unlock()

			if err != nil {
				p.log.Errorf("Agent %s failed: %v (duration: %s)", agentName, err, duration)
				result.Errors[agentName] = err
				result.Partial = true
			} else {
				result.Results[agentName] = output
				p.log.Debugf("Agent %s completed successfully (duration: %s)", agentName, duration)
			}
		}(ag)
	}

	// Wait for all agents to complete
	wg.Wait()

	// Check results based on strategy
	switch p.config.Strategy {
	case StrategyFailFast:
		if len(result.Errors) > 0 {
			// Return first error
			for agentName, err := range result.Errors {
				return result, errors.Wrapf(err, "pipeline failed at %s", agentName)
			}
		}

	case StrategyContinueOnError, StrategyPartialResults:
		// Return partial results
		if len(result.Results) == 0 && len(result.Errors) > 0 {
			return result, errors.Wrapf(errors.ErrInternal, "all agents failed")
		}
	}

	return result, nil
}

// executeAgent executes a single agent with error recovery
func (p *Pipeline) executeAgent(ctx context.Context, ag agent.Agent, input interface{}) (output interface{}, err error) {
	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrapf(errors.ErrInternal, "agent panicked: %v", r)
			p.log.Errorf("Agent panic recovered: %v", r)
		}
	}()

	// Check context before execution
	if ctx.Err() != nil {
		return nil, errors.Wrapf(ctx.Err(), "context cancelled before execution")
	}

	// Execute agent
	// Note: This assumes a generic execution interface.
	// In reality, you'd call the specific ADK agent methods
	output, err = executeAgentInterface(ctx, ag, input)
	if err != nil {
		return nil, err
	}

	return output, nil
}

// executeAgentInterface is a placeholder for actual agent execution
// In production, this would call the actual ADK agent Run method
func executeAgentInterface(ctx context.Context, ag agent.Agent, input interface{}) (interface{}, error) {
	// This is a placeholder - actual implementation depends on ADK's agent interface
	// You would call something like: ag.Run(ctx, input)
	return input, nil
}

// getAgentName extracts the name from an agent
func getAgentName(ag agent.Agent) string {
	// Try to get name from agent
	if named, ok := ag.(interface{ Name() string }); ok {
		return named.Name()
	}
	return fmt.Sprintf("%T", ag)
}

// SequentialPipeline creates a sequential pipeline
func SequentialPipeline(name string, agents []agent.Agent) *Pipeline {
	return NewPipeline(PipelineConfig{
		Name:     name,
		Strategy: StrategyPartialResults,
		Timeout:  5 * time.Minute,
	}, agents)
}

// ParallelPipeline creates a parallel pipeline
func ParallelPipeline(name string, agents []agent.Agent, maxConcurrency int) *Pipeline {
	return NewPipeline(PipelineConfig{
		Name:           name,
		Strategy:       StrategyPartialResults,
		Timeout:        5 * time.Minute,
		MaxConcurrency: maxConcurrency,
	}, agents)
}

// FailFastPipeline creates a pipeline that stops on first error
func FailFastPipeline(name string, agents []agent.Agent) *Pipeline {
	return NewPipeline(PipelineConfig{
		Name:     name,
		Strategy: StrategyFailFast,
		Timeout:  5 * time.Minute,
	}, agents)
}
