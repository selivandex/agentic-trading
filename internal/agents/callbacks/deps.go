package callbacks

import (
	"prometheus/internal/tools"
)

// Deps contains dependencies for creating callbacks
// Note: We use interface{} for some types to avoid import cycles
// Actual types are defined in internal/agents package
type Deps struct {
	Redis       interface{}     // *redis.Client
	CostTracker interface{}     // *agents.CostTracker
	RiskEngine  interface{}     // *risk.RiskEngine
	Registry    *tools.Registry // tools.Registry - used in risk validation callback
}
