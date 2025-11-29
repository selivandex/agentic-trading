package workflows

import (
	"context"
	"os"
	"strconv"

	"google.golang.org/adk/agent"

	"prometheus/pkg/logger"
)

// Priority levels for analysis requests
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// AnalysisRequest represents a request for market analysis
type AnalysisRequest struct {
	Symbol             string
	Exchange           string
	Priority           Priority
	PositionSizePct    float64 // Position size as % of portfolio (for high-stakes detection)
	Volatility         float64 // Current volatility (for high-stakes detection)
	ConflictingSignals bool    // Fast-path detected conflicts (escalate to committee)
}

// PathSelector intelligently routes analysis requests between fast-path and committee workflows
type PathSelector struct {
	// Two available paths
	fastPath     agent.Agent // OpportunitySynthesizer (fast, cheap, 15-30s)
	committePath agent.Agent // ResearchCommittee (thorough, expensive, 60-90s)

	// Configuration thresholds
	highStakesThreshold   float64 // Position size % threshold for committee (default 10%)
	volatilityThreshold   float64 // Volatility threshold for committee (default 5.0%)
	forceCommitteeOnHigh  bool    // Always use committee for high/critical priority
	committeeUsagePercent float64 // Target % of requests to route to committee (for cost control)

	// Metrics tracking
	totalRequests     int
	committeeRequests int

	log *logger.Logger
}

// PathSelectorConfig configures the PathSelector
type PathSelectorConfig struct {
	FastPath              agent.Agent
	CommitteePath         agent.Agent
	HighStakesThreshold   float64 // Default: 10%
	VolatilityThreshold   float64 // Default: 5.0%
	ForceCommitteeOnHigh  bool    // Default: true
	CommitteeUsagePercent float64 // Default: 20% (cost control)
}

// NewPathSelector creates a new intelligent path selector
func NewPathSelector(cfg PathSelectorConfig) *PathSelector {
	// Apply defaults
	if cfg.HighStakesThreshold == 0 {
		// Try to read from env
		if envVal := os.Getenv("RESEARCH_HIGH_STAKES_THRESHOLD"); envVal != "" {
			if val, err := strconv.ParseFloat(envVal, 64); err == nil {
				cfg.HighStakesThreshold = val
			}
		}
		if cfg.HighStakesThreshold == 0 {
			cfg.HighStakesThreshold = 10.0 // Default 10%
		}
	}

	if cfg.VolatilityThreshold == 0 {
		// Try to read from env
		if envVal := os.Getenv("RESEARCH_VOLATILITY_THRESHOLD"); envVal != "" {
			if val, err := strconv.ParseFloat(envVal, 64); err == nil {
				cfg.VolatilityThreshold = val
			}
		}
		if cfg.VolatilityThreshold == 0 {
			cfg.VolatilityThreshold = 5.0 // Default 5.0% volatility
		}
	}

	if cfg.CommitteeUsagePercent == 0 {
		cfg.CommitteeUsagePercent = 20.0 // Default 20%
	}

	return &PathSelector{
		fastPath:              cfg.FastPath,
		committePath:          cfg.CommitteePath,
		highStakesThreshold:   cfg.HighStakesThreshold,
		volatilityThreshold:   cfg.VolatilityThreshold,
		forceCommitteeOnHigh:  cfg.ForceCommitteeOnHigh,
		committeeUsagePercent: cfg.CommitteeUsagePercent,
		log:                   logger.Get().With("component", "path_selector"),
	}
}

// SelectPath chooses between fast-path and committee based on request characteristics
func (ps *PathSelector) SelectPath(ctx context.Context, req AnalysisRequest) agent.Agent {
	ps.totalRequests++

	// Criteria for using Research Committee (expensive, thorough path):
	//
	// 1. High/Critical Priority (user explicitly marked as important)
	// 2. Large position size (high stakes - affects significant % of portfolio)
	// 3. High market volatility (increased risk requires more analysis)
	// 4. Conflicting signals from fast-path (needs deeper analysis)
	// 5. Cost control: Don't exceed target % of requests to committee

	useCommittee := false
	reason := ""

	// Check 1: Priority-based routing
	if req.Priority == PriorityHigh || req.Priority == PriorityCritical {
		if ps.forceCommitteeOnHigh {
			useCommittee = true
			reason = "high/critical priority"
		}
	}

	// Check 2: High-stakes position (large % of portfolio)
	if req.PositionSizePct >= ps.highStakesThreshold {
		useCommittee = true
		if reason != "" {
			reason += ", "
		}
		reason += "high-stakes position"
	}

	// Check 3: High volatility environment
	if req.Volatility >= ps.volatilityThreshold {
		useCommittee = true
		if reason != "" {
			reason += ", "
		}
		reason += "high volatility"
	}

	// Check 4: Conflicting signals (escalation from fast-path)
	if req.ConflictingSignals {
		useCommittee = true
		if reason != "" {
			reason += ", "
		}
		reason += "conflicting signals"
	}

	// Check 5: Cost control - don't exceed target committee usage %
	if useCommittee {
		currentUsagePercent := float64(ps.committeeRequests) / float64(ps.totalRequests) * 100.0
		if currentUsagePercent >= ps.committeeUsagePercent {
			// Already at or above target, downgrade to fast-path unless critical
			if req.Priority != PriorityCritical {
				useCommittee = false
				reason = "cost control limit reached, downgraded to fast-path"
			}
		}
	}

	// Route decision
	if useCommittee {
		ps.committeeRequests++
		ps.log.Info("Routing to ResearchCommittee",
			"symbol", req.Symbol,
			"exchange", req.Exchange,
			"reason", reason,
			"priority", req.Priority,
			"position_size_pct", req.PositionSizePct,
			"volatility", req.Volatility,
			"committee_usage", float64(ps.committeeRequests)/float64(ps.totalRequests)*100.0,
		)
		return ps.committePath
	}

	ps.log.Debug("Routing to fast-path OpportunitySynthesizer",
		"symbol", req.Symbol,
		"exchange", req.Exchange,
		"priority", req.Priority,
	)
	return ps.fastPath
}

// GetMetrics returns current routing metrics
func (ps *PathSelector) GetMetrics() map[string]interface{} {
	committeeUsage := 0.0
	if ps.totalRequests > 0 {
		committeeUsage = float64(ps.committeeRequests) / float64(ps.totalRequests) * 100.0
	}

	return map[string]interface{}{
		"total_requests":        ps.totalRequests,
		"committee_requests":    ps.committeeRequests,
		"fast_path_requests":    ps.totalRequests - ps.committeeRequests,
		"committee_usage_pct":   committeeUsage,
		"target_committee_pct":  ps.committeeUsagePercent,
		"high_stakes_threshold": ps.highStakesThreshold,
		"volatility_threshold":  ps.volatilityThreshold,
	}
}

// ResetMetrics resets the routing counters (useful for testing or periodic reset)
func (ps *PathSelector) ResetMetrics() {
	ps.totalRequests = 0
	ps.committeeRequests = 0
}
