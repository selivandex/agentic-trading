package workflows

import (
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/workflowagents/sequentialagent"

	"prometheus/internal/agents"
	"prometheus/pkg/errors"
)

// CreateTradingPipeline creates a sequential pipeline: analysis → strategy → risk validation
func (f *Factory) CreateTradingPipeline() (agent.Agent, error) {
	f.log.Info("Creating trading pipeline workflow")

	// Step 1: Parallel analysts
	analystsStep, err := f.CreateParallelAnalysts()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create parallel analysts")
	}

	// Step 2: Strategy planner
	strategyAgent, err := f.createAgent(agents.AgentStrategyPlanner)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create strategy planner")
	}

	// Step 3: Risk manager
	riskAgent, err := f.createAgent(agents.AgentRiskManager)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create risk manager")
	}

	// Create sequential workflow
	pipeline, err := sequentialagent.New(sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        "TradingPipeline",
			Description: "Complete trading pipeline: parallel analysis → strategy synthesis → risk validation",
			SubAgents: []agent.Agent{
				analystsStep,
				strategyAgent,
				riskAgent,
			},
		},
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to create sequential pipeline")
	}

	f.log.Info("Trading pipeline workflow created successfully")
	return pipeline, nil
}

// CreateExecutionPipeline creates a full pipeline including executor
func (f *Factory) CreateExecutionPipeline() (agent.Agent, error) {
	f.log.Info("Creating full execution pipeline workflow")

	// Get trading pipeline (analysis → strategy → risk)
	tradingPipeline, err := f.CreateTradingPipeline()
	if err != nil {
		return nil, err
	}

	// Add executor as final step
	executorAgent, err := f.createAgent(agents.AgentExecutor)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create executor")
	}

	// Wrap in sequential agent
	fullPipeline, err := sequentialagent.New(sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        "FullExecutionPipeline",
			Description: "Complete pipeline including execution: analysis → strategy → risk → execution",
			SubAgents: []agent.Agent{
				tradingPipeline,
				executorAgent,
			},
		},
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to create full pipeline")
	}

	f.log.Info("Full execution pipeline workflow created successfully")
	return fullPipeline, nil
}
