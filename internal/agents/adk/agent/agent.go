package agent

import "context"

// Agent defines the contract for all ADK agents.
type Agent interface {
	Name() string
	Description() string
	Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
}

// Config captures metadata shared by composite workflow agents.
type Config struct {
	Name        string
	Description string
	SubAgents   []Agent
}
