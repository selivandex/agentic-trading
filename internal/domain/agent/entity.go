package agent

import (
	"encoding/json"
	"time"

	"prometheus/internal/adapters/ai"
)

// Agent represents an AI agent definition with configuration
type Agent struct {
	ID          int    `db:"id"`
	Identifier  string `db:"identifier"` // portfolio_manager, technical_analyzer
	Name        string `db:"name"`
	Description string `db:"description"`
	Category    string `db:"category"` // expert, coordinator, specialist

	// Prompts (editable in production)
	SystemPrompt string `db:"system_prompt"`
	Instructions string `db:"instructions"`

	// Model configuration
	ModelProvider ai.ProviderName      `db:"model_provider"` // Type-safe provider enum
	ModelName     ai.ProviderModelName `db:"model_name"`
	Temperature   float64              `db:"temperature"`
	MaxTokens     int                  `db:"max_tokens"`

	// Tools (JSONB array)
	AvailableTools json.RawMessage `db:"available_tools"`

	// Limits
	MaxCostPerRun  float64 `db:"max_cost_per_run"`
	TimeoutSeconds int     `db:"timeout_seconds"`

	// Status
	IsActive bool `db:"is_active"`
	Version  int  `db:"version"`

	// Timestamps
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Category constants
const (
	CategoryCoordinator = "coordinator" // High-level decision makers (PortfolioManager)
	CategoryExpert      = "expert"      // Specialists (TechnicalAnalyzer, SMCAnalyzer)
	CategorySpecialist  = "specialist"  // Domain specialists
)

// Well-known agent identifiers
const (
	IdentifierPortfolioArchitect     = "portfolio_architect"
	IdentifierPortfolioManager       = "portfolio_manager"
	IdentifierTechnicalAnalyzer      = "technical_analyzer"
	IdentifierSMCAnalyzer            = "smc_analyzer"
	IdentifierCorrelationAnalyzer    = "correlation_analyzer"
	IdentifierOpportunitySynthesizer = "opportunity_synthesizer"
)

// GetAvailableTools parses the JSONB tools array
func (a *Agent) GetAvailableTools() ([]string, error) {
	var tools []string
	if len(a.AvailableTools) == 0 {
		return tools, nil
	}
	if err := json.Unmarshal(a.AvailableTools, &tools); err != nil {
		return nil, err
	}
	return tools, nil
}

// SetAvailableTools encodes tools array to JSONB
func (a *Agent) SetAvailableTools(tools []string) error {
	data, err := json.Marshal(tools)
	if err != nil {
		return err
	}
	a.AvailableTools = data
	return nil
}
