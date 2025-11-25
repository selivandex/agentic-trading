package agents

import (
	"context"
	"fmt"
)

// ParallelAgent runs multiple agents concurrently and returns their responses.
type ParallelAgent struct {
	Agents []Agent
}

// Run executes all agents in parallel.
func (p ParallelAgent) Run(ctx context.Context, req AgentRequest) (map[AgentType]*AgentResponse, error) {
	results := make(map[AgentType]*AgentResponse)
	errs := make(chan error, len(p.Agents))
	respChan := make(chan struct {
		t AgentType
		r *AgentResponse
	}, len(p.Agents))

	for _, ag := range p.Agents {
		agent := ag
		go func() {
			res, err := agent.Run(ctx, req)
			if err != nil {
				errs <- err
				return
			}
			respChan <- struct {
				t AgentType
				r *AgentResponse
			}{t: agent.Type(), r: res}
		}()
	}

	for i := 0; i < len(p.Agents); i++ {
		select {
		case err := <-errs:
			return nil, err
		case resp := <-respChan:
			results[resp.t] = resp.r
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return results, nil
}

// SequentialAgent runs agents one after another, feeding responses forward.
type SequentialAgent struct {
	Agents []Agent
}

// Run executes agents sequentially, aggregating responses.
func (s SequentialAgent) Run(ctx context.Context, req AgentRequest) ([]*AgentResponse, error) {
	responses := make([]*AgentResponse, 0, len(s.Agents))

	for _, ag := range s.Agents {
		res, err := ag.Run(ctx, req)
		if err != nil {
			return nil, err
		}
		responses = append(responses, res)
	}

	return responses, nil
}

// CreateTradingPipeline wires a basic analysis → strategy → risk → execution flow.
func CreateTradingPipeline(reg *Registry) (*SequentialAgent, error) {
	required := []AgentType{AgentMarketAnalyst, AgentStrategyPlanner, AgentRiskManager, AgentExecutor}
	agents := make([]Agent, 0, len(required))

	for _, t := range required {
		ag, ok := reg.Get(t)
		if !ok {
			return nil, fmt.Errorf("missing agent %s in registry", t)
		}
		agents = append(agents, ag)
	}

	return &SequentialAgent{Agents: agents}, nil
}
