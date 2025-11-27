package agents

import (
	"google.golang.org/adk/tool"

	"prometheus/pkg/logger"
)

// CreateExpertTools creates agent-as-tool wrappers for specialized expert agents
// These can be used by higher-level agents (like Strategy Planner) to consult experts
//
// NOTE: After Phase 2 refactoring, most analyst agents were removed and replaced with algorithmic tools.
// This function is kept for future extensibility but currently returns an empty map.
// Expert consultation can be re-enabled if needed by creating specialized agents for specific use cases.
func (f *Factory) CreateExpertTools(provider, model string) (map[string]tool.Tool, error) {
	log := logger.Get().With("component", "expert_tools")

	expertTools := make(map[string]tool.Tool)

	// Phase 2 refactoring: All analyst agents removed
	// Expert tools disabled until specific use cases identified
	// Previous expert tools (macro, onchain, correlation, derivatives) replaced by algorithmic tools:
	// - get_technical_analysis (comprehensive indicators)
	// - get_smc_analysis (Smart Money Concepts)
	// - get_market_analysis (order flow, whale detection)

	log.Info("Expert tools disabled (Phase 2 refactoring - use algorithmic tools instead)")
	return expertTools, nil
}
