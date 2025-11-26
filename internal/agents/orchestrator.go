package agents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"prometheus/internal/adapters/ai"
	"prometheus/internal/adapters/config"
	"prometheus/internal/domain/memory"
	"prometheus/internal/domain/reasoning"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Orchestrator coordinates execution of multiple agents (parallel or sequential)
type Orchestrator struct {
	factory       *Factory
	runtimeConfig config.AgentsConfig
	costTracker   *CostTracker
	memoryService *memory.Service
	reasoningRepo reasoning.Repository
	log           *logger.Logger
}

// NewOrchestrator creates a new agent orchestrator
func NewOrchestrator(
	factory *Factory,
	runtimeConfig config.AgentsConfig,
	costTracker *CostTracker,
	memoryService *memory.Service,
	reasoningRepo reasoning.Repository,
) *Orchestrator {
	return &Orchestrator{
		factory:       factory,
		runtimeConfig: runtimeConfig,
		costTracker:   costTracker,
		memoryService: memoryService,
		reasoningRepo: reasoningRepo,
		log:           logger.Get().With("component", "orchestrator"),
	}
}

// AgentRequest represents a request to run an agent
type AgentRequest struct {
	AgentType AgentType
	Input     ExecutionInput
}

// AgentResult represents the result of an agent execution
type AgentResult struct {
	AgentType AgentType
	Output    *ExecutionOutput
	Error     error
	Duration  time.Duration
}

// RunParallel executes multiple agents in parallel and returns all results
func (o *Orchestrator) RunParallel(
	ctx context.Context,
	requests []AgentRequest,
	provider string,
	model string,
) (map[AgentType]*ExecutionOutput, error) {
	if len(requests) == 0 {
		return nil, errors.ErrInvalidInput
	}

	o.log.Infof("Starting parallel execution of %d agents", len(requests))
	startTime := time.Now()

	var wg sync.WaitGroup
	results := make(chan AgentResult, len(requests))

	// Launch all agents in parallel
	for _, req := range requests {
		wg.Add(1)
		go func(r AgentRequest) {
			defer wg.Done()

			agentStart := time.Now()
			output, err := o.runSingleAgent(ctx, r, provider, model)

			results <- AgentResult{
				AgentType: r.AgentType,
				Output:    output,
				Error:     err,
				Duration:  time.Since(agentStart),
			}
		}(req)
	}

	// Wait for all to complete
	wg.Wait()
	close(results)

	// Collect results
	outputMap := make(map[AgentType]*ExecutionOutput)
	var errs []error

	for result := range results {
		if result.Error != nil {
			o.log.Errorf("Agent %s failed: %v (duration: %v)",
				result.AgentType, result.Error, result.Duration)
			errs = append(errs, errors.Wrapf(result.Error, "agent %s", result.AgentType))
		} else {
			o.log.Infof("Agent %s completed successfully (duration: %v, tokens: %d, cost: $%.4f)",
				result.AgentType, result.Duration, result.Output.TokensUsed, result.Output.CostUSD)
			outputMap[result.AgentType] = result.Output
		}
	}

	totalDuration := time.Since(startTime)
	o.log.Infof("Parallel execution complete: %d/%d agents succeeded (duration: %v)",
		len(outputMap), len(requests), totalDuration)

	// Return error if ANY agent failed (caller can decide how to handle)
	if len(errs) > 0 {
		return outputMap, fmt.Errorf("some agents failed: %v", errs)
	}

	return outputMap, nil
}

// RunSequential executes agents one after another, passing results to next agent
func (o *Orchestrator) RunSequential(
	ctx context.Context,
	requests []AgentRequest,
	provider string,
	model string,
) ([]*ExecutionOutput, error) {
	if len(requests) == 0 {
		return nil, errors.ErrInvalidInput
	}

	o.log.Infof("Starting sequential execution of %d agents", len(requests))
	startTime := time.Now()

	results := make([]*ExecutionOutput, 0, len(requests))

	for i, req := range requests {
		o.log.Debugf("Running agent %d/%d: %s", i+1, len(requests), req.AgentType)

		agentStart := time.Now()
		output, err := o.runSingleAgent(ctx, req, provider, model)

		if err != nil {
			o.log.Errorf("Agent %s failed: %v (duration: %v)",
				req.AgentType, err, time.Since(agentStart))
			return results, errors.Wrapf(err, "sequential execution failed at agent %s", req.AgentType)
		}

		o.log.Infof("Agent %s completed (duration: %v, tokens: %d, cost: $%.4f)",
			req.AgentType, time.Since(agentStart), output.TokensUsed, output.CostUSD)

		results = append(results, output)

		// Pass previous agent results to the next agent via Context
		// This enables chaining where each agent sees previous outputs
		if i+1 < len(requests) {
			nextReq := &requests[i+1]
			if nextReq.Input.Context == nil {
				nextReq.Input.Context = make(map[string]interface{})
			}

			// Add all previous results to context
			previousResults := make([]map[string]interface{}, 0, len(results))
			for _, prevOutput := range results {
				previousResults = append(previousResults, map[string]interface{}{
					"agent":      prevOutput.AgentType,
					"result":     prevOutput.Result,
					"confidence": prevOutput.Confidence,
					"summary":    prevOutput.RawResponse,
				})
			}

			nextReq.Input.Context["previous_agents"] = previousResults

			o.log.Debugf("Passing %d previous agent results to %s", len(previousResults), nextReq.AgentType)
		}
	}

	totalDuration := time.Since(startTime)
	o.log.Infof("Sequential execution complete: %d agents (duration: %v)",
		len(results), totalDuration)

	return results, nil
}

// RunAnalysisPipeline runs the full analysis pipeline:
// 1. Parallel analysis agents (market, SMC, sentiment, etc.)
// 2. Strategy planner synthesizes results
// 3. Risk manager validates
func (o *Orchestrator) RunAnalysisPipeline(
	ctx context.Context,
	userID uuid.UUID,
	symbol string,
	marketType string,
	provider string,
	model string,
) (*TradePlan, error) {
	o.log.Infof("Starting analysis pipeline: user=%s symbol=%s", userID, symbol)

	// Phase 1: Run analysis agents in parallel
	analysisRequests := []AgentRequest{
		{
			AgentType: AgentMarketAnalyst,
			Input: ExecutionInput{
				UserID:     userID,
				Symbol:     symbol,
				MarketType: marketType,
			},
		},
		{
			AgentType: AgentSMCAnalyst,
			Input: ExecutionInput{
				UserID:     userID,
				Symbol:     symbol,
				MarketType: marketType,
			},
		},
		{
			AgentType: AgentOrderFlowAnalyst,
			Input: ExecutionInput{
				UserID:     userID,
				Symbol:     symbol,
				MarketType: marketType,
			},
		},
		// TODO: Add more analysts based on configuration
		// AgentSentimentAnalyst, AgentDerivativesAnalyst, etc.
	}

	analysisResults, err := o.RunParallel(ctx, analysisRequests, provider, model)
	if err != nil {
		// Log error but continue with partial results
		o.log.Warnf("Some analysis agents failed: %v", err)
	}

	if len(analysisResults) == 0 {
		return nil, errors.New("all analysis agents failed")
	}

	// Phase 2: Run strategy planner with analysis results
	// Prepare comprehensive context for strategy planner
	strategyContext := make(map[string]interface{})
	analystSummaries := make([]map[string]interface{}, 0, len(analysisResults))

	for agentType, output := range analysisResults {
		analystSummaries = append(analystSummaries, map[string]interface{}{
			"agent":      agentType,
			"result":     output.Result,
			"confidence": output.Confidence,
			"raw":        output.RawResponse,
		})
	}

	strategyContext["analyst_results"] = analystSummaries
	strategyContext["num_analysts"] = len(analysisResults)

	strategyInput := ExecutionInput{
		UserID:     userID,
		Symbol:     symbol,
		MarketType: marketType,
		Context:    strategyContext,
	}

	strategyOutput, err := o.runSingleAgent(
		ctx,
		AgentRequest{AgentType: AgentStrategyPlanner, Input: strategyInput},
		provider,
		model,
	)
	if err != nil {
		return nil, errors.Wrap(err, "strategy planner failed")
	}

	// Phase 3: Run risk manager to validate the strategy plan
	riskContext := map[string]interface{}{
		"proposed_plan":      strategyOutput.Result,
		"analysis_context":   analystSummaries,
		"planner_confidence": strategyOutput.Confidence,
	}

	riskInput := ExecutionInput{
		UserID:     userID,
		Symbol:     symbol,
		MarketType: marketType,
		Context:    riskContext,
	}

	riskOutput, err := o.runSingleAgent(
		ctx,
		AgentRequest{AgentType: AgentRiskManager, Input: riskInput},
		provider,
		model,
	)
	if err != nil {
		return nil, errors.Wrap(err, "risk manager failed")
	}

	// Extract risk approval decision from risk manager output
	riskApproved := false
	if decision, ok := riskOutput.Result["decision"].(string); ok {
		riskApproved = (decision == "approved" || decision == "approved_with_conditions")
	}

	// Build complete trading plan
	plan := &TradePlan{
		UserID:         userID,
		Symbol:         symbol,
		MarketType:     marketType,
		StrategyOutput: strategyOutput,
		AnalysisInputs: analysisResults,
		RiskApproved:   riskApproved,
		CreatedAt:      time.Now(),
	}

	o.log.Infof("Analysis pipeline complete: %d analysts → strategy → risk validation",
		len(analysisResults))

	return plan, nil
}

// runSingleAgent creates and runs a single agent
func (o *Orchestrator) runSingleAgent(
	ctx context.Context,
	req AgentRequest,
	provider string,
	model string,
) (*ExecutionOutput, error) {
	// Create agent instance
	ag, err := o.factory.CreateAgentForUser(req.AgentType, provider, model)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create agent %s", req.AgentType)
	}

	// Get agent config and model info
	cfg, ok := DefaultAgentConfigs[req.AgentType]
	if !ok {
		return nil, fmt.Errorf("config not found for agent %s", req.AgentType)
	}

	// Get model info from AI registry (factory has aiRegistry)
	modelInfoValue, err := o.factory.aiRegistry.ResolveModel(ctx, provider, model)
	if err != nil {
		o.log.Warnf("Failed to resolve model %s/%s from registry: %v, using defaults", provider, model, err)
		// Fallback to placeholder
		modelInfoValue = ai.ModelInfo{
			Name:              model,
			InputCostPer1K:    0.003,
			OutputCostPer1K:   0.015,
			SupportsTools:     true,
			SupportsStreaming: false,
		}
	}

	// Create agent runner
	runner, err := NewAgentRunner(
		ag,
		req.AgentType,
		cfg,
		&modelInfoValue,
		o.runtimeConfig,
		o.memoryService,
		o.reasoningRepo,
		o.costTracker,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create agent runner")
	}

	// Execute agent
	return runner.Execute(ctx, req.Input)
}

// TradePlan represents a complete trading plan from the pipeline
type TradePlan struct {
	UserID         uuid.UUID
	Symbol         string
	MarketType     string
	StrategyOutput *ExecutionOutput
	AnalysisInputs map[AgentType]*ExecutionOutput
	RiskApproved   bool
	CreatedAt      time.Time
}
