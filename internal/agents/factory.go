package agents

import (
	"fmt"

	"prometheus/internal/tools"
	"prometheus/pkg/templates"
)

// FactoryDeps gathers external dependencies needed to instantiate agents.
type FactoryDeps struct {
	ToolRegistry *tools.Registry
	Templates    *templates.Registry
}

// Factory creates configured agents and registries.
type Factory struct {
	deps FactoryDeps
}

// NewFactory builds an agent factory with required dependencies.
func NewFactory(deps FactoryDeps) (*Factory, error) {
	if deps.ToolRegistry == nil {
		return nil, fmt.Errorf("tool registry is required")
	}

	if deps.Templates == nil {
		deps.Templates = templates.Get()
	}

	return &Factory{deps: deps}, nil
}

// CreateAgent constructs a single agent instance from a config.
func (f *Factory) CreateAgent(cfg AgentConfig) (Agent, error) {
	agent, err := newPromptAgent(cfg, f.deps.ToolRegistry, f.deps.Templates)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

// CreateDefaultRegistry builds and registers agents using DefaultAgentConfigs.
func (f *Factory) CreateDefaultRegistry() (*Registry, error) {
	reg := NewRegistry()

	for _, cfg := range DefaultAgentConfigs {
		ag, err := f.CreateAgent(cfg)
		if err != nil {
			return nil, err
		}
		reg.Register(cfg.Type, ag)
	}

	return reg, nil
}
