package agents

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"prometheus/internal/tools"
	"prometheus/pkg/templates"
)

// Agent defines minimal execution contract used by orchestrators.
type Agent interface {
	ID() string
	Type() AgentType
	Run(ctx context.Context, req AgentRequest) (*AgentResponse, error)
}

// AgentRequest is passed to agents for a single execution.
type AgentRequest struct {
	UserID        uuid.UUID
	SessionID     string
	Symbol        string
	RiskLevel     string
	TradingPairID *uuid.UUID
	Input         map[string]interface{}
}

// AgentResponse represents a result placeholder with telemetry.
type AgentResponse struct {
	AgentID       string
	Output        string
	ToolsAllowed  []string
	Prompt        string
	DurationMs    int
	TokensUsed    int
	CostUSD       float64
	ToolCallsMade int
}

// promptAgent is a lightweight implementation that renders system prompts and records metadata.
type promptAgent struct {
	id           string
	agentType    AgentType
	config       AgentConfig
	tools        *tools.Registry
	templates    *templates.Registry
	toolMetadata map[string]tools.Definition
}

// newPromptAgent constructs a prompt-driven agent used for orchestration scaffolding.
func newPromptAgent(cfg AgentConfig, reg *tools.Registry, tmpl *templates.Registry) (*promptAgent, error) {
	if reg == nil {
		return nil, fmt.Errorf("tool registry is required")
	}

	if tmpl == nil {
		tmpl = templates.Get()
	}

	toolMeta := map[string]tools.Definition{}
	for _, def := range tools.Definitions() {
		toolMeta[def.Name] = def
	}

	return &promptAgent{
		id:           cfg.Name,
		agentType:    cfg.Type,
		config:       cfg,
		tools:        reg,
		templates:    tmpl,
		toolMetadata: toolMeta,
	}, nil
}

func (a *promptAgent) ID() string      { return a.id }
func (a *promptAgent) Type() AgentType { return a.agentType }

func (a *promptAgent) Run(ctx context.Context, req AgentRequest) (*AgentResponse, error) {
	start := time.Now()

	allowed := make([]tools.Definition, 0, len(a.config.Tools))
	for _, name := range a.config.Tools {
		if def, ok := a.toolMetadata[name]; ok {
			allowed = append(allowed, def)
			continue
		}
		allowed = append(allowed, tools.Definition{Name: name, Description: ""})
	}

	data := map[string]interface{}{
		"Symbol":       req.Symbol,
		"RiskLevel":    req.RiskLevel,
		"Tools":        allowed,
		"MaxToolCalls": a.config.MaxToolCalls,
		"AgentName":    a.config.Name,
		"AgentType":    a.config.Type,
		"Input":        req.Input,
	}

	for key, value := range req.Input {
		data[key] = value
	}

	prompt, err := a.templates.Render(a.config.SystemPromptTemplate, data)
	if err != nil {
		return nil, fmt.Errorf("render prompt for %s: %w", a.config.Name, err)
	}

	duration := time.Since(start)

	return &AgentResponse{
		AgentID:      a.id,
		Output:       "prompt generated",
		ToolsAllowed: a.config.Tools,
		Prompt:       prompt,
		DurationMs:   int(duration.Milliseconds()),
		TokensUsed:   0,
		CostUSD:      0,
	}, nil
}
