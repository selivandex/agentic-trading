package workflows

import (
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/workflowagents/loopagent"

	"prometheus/internal/agents"
	"prometheus/pkg/errors"
)

// CreatePositionMonitoringLoop creates a looping workflow for continuous position monitoring
func (f *Factory) CreatePositionMonitoringLoop(maxIterations uint) (agent.Agent, error) {
	f.log.Info("Creating position monitoring loop workflow")

	// Position manager agent
	positionManagerAgent, err := f.createAgent(agents.AgentPositionManager)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create position manager")
	}

	// Create looping workflow
	monitoringLoop, err := loopagent.New(loopagent.Config{
		AgentConfig: agent.Config{
			Name:        "PositionMonitoringLoop",
			Description: "Continuously monitors open positions and adjusts stops/targets",
			SubAgents: []agent.Agent{
				positionManagerAgent,
			},
		},
		MaxIterations: maxIterations, // 0 = infinite loop
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to create loop agent")
	}

	f.log.Infof("Position monitoring loop created with max iterations: %d", maxIterations)
	return monitoringLoop, nil
}


