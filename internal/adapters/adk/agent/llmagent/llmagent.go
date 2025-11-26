package llmagent

import (
	"context"
	"fmt"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
)

// Config defines the construction parameters for an LLM-backed agent.
type Config struct {
	Name        string
	Description string
	Model       model.Model
	Tools       []tool.Tool
	Instruction string
	OutputKey   string
}

// Agent is a lightweight LLM-style agent placeholder.
type Agent struct {
	cfg Config
}

// New constructs a new llm-style agent.
func New(cfg Config) (*Agent, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if cfg.Model == nil {
		return nil, fmt.Errorf("model is required")
	}
	return &Agent{cfg: cfg}, nil
}

// Name returns the agent's name.
func (a *Agent) Name() string { return a.cfg.Name }

// Description returns the agent description.
func (a *Agent) Description() string { return a.cfg.Description }

// Run synthesizes a deterministic placeholder response capturing instruction and tools.
func (a *Agent) Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	_ = ctx
	toolNames := make([]string, 0, len(a.cfg.Tools))
	for _, t := range a.cfg.Tools {
		toolNames = append(toolNames, fmt.Sprintf("%s - %s", t.Name(), t.Description()))
	}

	output := map[string]interface{}{
		"instruction": a.cfg.Instruction,
		"tools":       toolNames,
		"model":       a.cfg.Model.Name(),
		"input":       input,
	}

	if a.cfg.OutputKey != "" {
		return map[string]interface{}{a.cfg.OutputKey: output}, nil
	}

	return output, nil
}

var _ agent.Agent = (*Agent)(nil)
