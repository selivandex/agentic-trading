package agents

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"prometheus/internal/domain/reasoning"
)

// CoTWrapper logs reasoning metadata for agent runs.
type CoTWrapper struct {
	agent         Agent
	agentID       string
	reasoningRepo reasoning.Repository
}

// NewCoTWrapper constructs a wrapper around an agent.
func NewCoTWrapper(agent Agent, agentID string, repo reasoning.Repository) *CoTWrapper {
	return &CoTWrapper{
		agent:         agent,
		agentID:       agentID,
		reasoningRepo: repo,
	}
}

// Run executes the underlying agent and records a simplified reasoning log.
func (w *CoTWrapper) Run(ctx context.Context, req AgentRequest) (*AgentResponse, error) {
	start := time.Now()
	resp, err := w.agent.Run(ctx, req)
	duration := time.Since(start)

	if w.reasoningRepo != nil {
		steps, _ := json.Marshal([]reasoning.Step{{
			Step:      1,
			Action:    "decision",
			Content:   resp.Output,
			Timestamp: time.Now(),
		}})

		decision, _ := json.Marshal(resp)

		entry := &reasoning.LogEntry{
			ID:             uuid.New(),
			UserID:         req.UserID,
			AgentID:        w.agentID,
			SessionID:      req.SessionID,
			Symbol:         req.Symbol,
			TradingPairID:  req.TradingPairID,
			ReasoningSteps: steps,
			Decision:       decision,
			Confidence:     0,
			TokensUsed:     resp.TokensUsed,
			CostUSD:        resp.CostUSD,
			DurationMs:     int(duration.Milliseconds()),
			ToolCallsCount: resp.ToolCallsMade,
			CreatedAt:      time.Now(),
		}

		go w.reasoningRepo.Create(context.Background(), entry)
	}

	return resp, err
}
