package schemas

import (
	"time"

	"google.golang.org/genai"
)

// Helper functions for creating float64 pointers
func float64Ptr(v float64) *float64 {
	return &v
}

// ============================================================================
// Common Reasoning Structures for Explainable AI
// ============================================================================
// These types support the Chain-of-Thought framework and provide transparent
// decision-making traces for audit, learning, and debugging.

// ReasoningStep represents one step in the agent's reasoning chain.
// Each step documents what was done, the input/output, and the reasoning.
type ReasoningStep struct {
	Step      string      `json:"step"`      // e.g., "evidence_gathering", "synthesis", "decision"
	Input     interface{} `json:"input"`     // Input to this step
	Output    interface{} `json:"output"`    // Output from this step
	Reasoning string      `json:"reasoning"` // Explanation of this step
}

// Evidence tracks the sources and quality of data used in decision.
// This ensures decisions are traceable and quality-assessed.
type Evidence struct {
	Sources     []string  `json:"sources"`      // Tool names or data sources used
	Timestamp   time.Time `json:"timestamp"`    // When evidence was gathered
	DataQuality float64   `json:"data_quality"` // 0-1 quality score
}

// Alternative represents an option that was considered but rejected.
// This documents the decision space and shows deliberation.
type Alternative struct {
	Option      string `json:"option"`       // Description of alternative
	WhyRejected string `json:"why_rejected"` // Reason for rejecting this option
}

// ============================================================================
// ADK Schemas for Reasoning Structures
// ============================================================================

// ReasoningStepSchema is the ADK schema for a reasoning step
var ReasoningStepSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"step": {
			Type:        "STRING",
			Description: "Name of this reasoning step (e.g., evidence_gathering, synthesis, decision)",
		},
		"input": {
			Type:        "OBJECT",
			Description: "Input to this step (structured data)",
		},
		"output": {
			Type:        "OBJECT",
			Description: "Output from this step (structured data)",
		},
		"reasoning": {
			Type:        "STRING",
			Description: "Explanation of the reasoning in this step",
		},
	},
	Required: []string{"step", "reasoning"},
}

// EvidenceSchema is the ADK schema for evidence tracking
var EvidenceSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"sources": {
			Type:        "ARRAY",
			Description: "List of data sources or tools used (e.g., ['get_technical_analysis', 'get_smc_analysis'])",
			Items: &genai.Schema{
				Type: "STRING",
			},
		},
		"timestamp": {
			Type:        "STRING",
			Description: "ISO 8601 timestamp when evidence was gathered",
		},
		"data_quality": {
			Type:        "NUMBER",
			Description: "Quality score of data (0-1), assessing freshness and completeness",
			Minimum:     float64Ptr(0),
			Maximum:     float64Ptr(1),
		},
	},
	Required: []string{"sources", "timestamp"},
}

// AlternativeSchema is the ADK schema for alternatives considered
var AlternativeSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"option": {
			Type:        "STRING",
			Description: "Description of the alternative considered",
		},
		"why_rejected": {
			Type:        "STRING",
			Description: "Reason why this alternative was not chosen",
		},
	},
	Required: []string{"option", "why_rejected"},
}

// ============================================================================
// Agent Output Schemas
// ============================================================================

// PortfolioManagerOutputSchema defines the output schema for the Portfolio Manager agent.
// This agent personalizes global trading opportunities for individual clients based on their
// portfolio, risk profile, and capital constraints.
var PortfolioManagerOutputSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"recommendation": {
			Type:        "STRING",
			Description: "Overall trade recommendation",
			Enum:        []string{"strong_buy", "buy", "hold", "sell", "strong_sell", "no_trade"},
		},
		"entry_price": {
			Type:        "NUMBER",
			Description: "Recommended entry price",
			Minimum:     float64Ptr(0),
		},
		"stop_loss": {
			Type:        "NUMBER",
			Description: "Stop loss price",
			Minimum:     float64Ptr(0),
		},
		"take_profit": {
			Type:        "ARRAY",
			Description: "Multiple take profit levels",
			Items: &genai.Schema{
				Type:    "NUMBER",
				Minimum: float64Ptr(0),
			},
		},
		"position_size": {
			Type:        "NUMBER",
			Description: "Recommended position size as percentage of portfolio",
			Minimum:     float64Ptr(0),
			Maximum:     float64Ptr(100),
		},
		"confidence": {
			Type:        "NUMBER",
			Description: "Confidence in this strategy (0-1)",
			Minimum:     float64Ptr(0),
			Maximum:     float64Ptr(1),
		},
		"time_horizon": {
			Type:        "STRING",
			Description: "Expected trade duration",
			Enum:        []string{"scalp", "intraday", "swing", "position"},
		},
		"key_factors": {
			Type:        "ARRAY",
			Description: "Key factors supporting this strategy",
			Items: &genai.Schema{
				Type: "STRING",
			},
		},
		// Explainable reasoning fields (Phase 1 refactoring)
		"reasoning_trace": {
			Type:        "ARRAY",
			Description: "Step-by-step reasoning chain showing how the decision was made",
			Items:       ReasoningStepSchema,
		},
		"evidence": {
			Type:        "OBJECT",
			Description: "Evidence sources and data quality used in decision",
			Properties:  EvidenceSchema.Properties,
			Required:    EvidenceSchema.Required,
		},
		"alternatives_considered": {
			Type:        "ARRAY",
			Description: "Alternative options that were evaluated and rejected",
			Items:       AlternativeSchema,
		},
	},
	Required: []string{"recommendation", "confidence", "reasoning_trace", "evidence"},
}

// PreTradeReviewerOutputSchema defines the output schema for the Pre-Trade Reviewer agent.
// This agent acts as a quality gate, red-teaming trade plans before execution.
var PreTradeReviewerOutputSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"decision": {
			Type:        "STRING",
			Description: "Final review decision",
			Enum:        []string{"GO", "HOLD", "NO-GO"},
		},
		"adjusted_confidence": {
			Type:        "NUMBER",
			Description: "Confidence level after review (0-1)",
			Minimum:     float64Ptr(0),
			Maximum:     float64Ptr(1),
		},
		"concerns": {
			Type:        "ARRAY",
			Description: "List of concerns or issues identified",
			Items: &genai.Schema{
				Type: "STRING",
			},
		},
		"pre_mortem_scenarios": {
			Type:        "ARRAY",
			Description: "Worst-case scenarios and their probabilities",
			Items: &genai.Schema{
				Type: "OBJECT",
				Properties: map[string]*genai.Schema{
					"scenario": {
						Type:        "STRING",
						Description: "Description of the failure scenario",
					},
					"probability": {
						Type:        "NUMBER",
						Description: "Estimated probability (0-1)",
						Minimum:     float64Ptr(0),
						Maximum:     float64Ptr(1),
					},
					"impact": {
						Type:        "STRING",
						Description: "Expected impact if scenario occurs",
					},
				},
				Required: []string{"scenario", "probability", "impact"},
			},
		},
		"recommendation": {
			Type:        "STRING",
			Description: "Detailed recommendation with rationale",
		},
		"reasoning": {
			Type:        "STRING",
			Description: "Summary of review reasoning",
		},
		// Explainability fields
		"reasoning_trace": {
			Type:        "ARRAY",
			Description: "Step-by-step reasoning chain from evidence to decision",
			Items:       ReasoningStepSchema,
		},
		"evidence": {
			Type:        "OBJECT",
			Description: "Evidence sources and data quality used in review",
			Properties:  EvidenceSchema.Properties,
			Required:    EvidenceSchema.Required,
		},
		"alternatives_considered": {
			Type:        "ARRAY",
			Description: "Alternative decisions that were considered and rejected",
			Items:       AlternativeSchema,
		},
	},
	Required: []string{"decision", "adjusted_confidence", "concerns", "recommendation", "reasoning", "reasoning_trace", "evidence"},
}

// PerformanceCommitteeOutputSchema defines the output schema for the Performance Committee agent.
// This agent conducts weekly performance reviews, extracting patterns and lessons for system improvement.
var PerformanceCommitteeOutputSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"summary": {
			Type:        "OBJECT",
			Description: "High-level summary of trading performance",
			Properties: map[string]*genai.Schema{
				"total_trades": {
					Type:        "INTEGER",
					Description: "Total number of closed trades analyzed",
				},
				"win_rate": {
					Type:        "NUMBER",
					Description: "Overall win rate (0-1)",
					Minimum:     float64Ptr(0),
					Maximum:     float64Ptr(1),
				},
				"avg_rr": {
					Type:        "NUMBER",
					Description: "Average risk:reward ratio",
					Minimum:     float64Ptr(0),
				},
				"best_strategy": {
					Type:        "STRING",
					Description: "Best performing strategy/setup",
				},
				"worst_strategy": {
					Type:        "STRING",
					Description: "Worst performing strategy/setup",
				},
				"sample_period": {
					Type:        "STRING",
					Description: "Time period analyzed (e.g., 'Nov 21-28, 2025')",
				},
			},
			Required: []string{"total_trades", "win_rate", "avg_rr"},
		},
		"validated_patterns": {
			Type:        "ARRAY",
			Description: "Patterns with n≥10 trades showing statistical significance",
			Items: &genai.Schema{
				Type: "OBJECT",
				Properties: map[string]*genai.Schema{
					"pattern": {
						Type:        "STRING",
						Description: "Name/description of the pattern",
					},
					"sample_size": {
						Type:        "INTEGER",
						Description: "Number of trades in this pattern",
						Minimum:     float64Ptr(10),
					},
					"win_rate": {
						Type:        "NUMBER",
						Description: "Win rate for this pattern (0-1)",
						Minimum:     float64Ptr(0),
						Maximum:     float64Ptr(1),
					},
					"avg_rr": {
						Type:        "NUMBER",
						Description: "Average R:R for this pattern",
						Minimum:     float64Ptr(0),
					},
					"conditions": {
						Type:        "STRING",
						Description: "Conditions when this pattern applies",
					},
					"actionable_rule": {
						Type:        "STRING",
						Description: "Specific rule to implement based on this pattern",
					},
					"confidence": {
						Type:        "STRING",
						Description: "Confidence in pattern validity",
						Enum:        []string{"low", "medium", "high"},
					},
				},
				Required: []string{"pattern", "sample_size", "win_rate", "avg_rr", "actionable_rule"},
			},
		},
		"failure_modes": {
			Type:        "ARRAY",
			Description: "Common failure patterns and their lessons",
			Items: &genai.Schema{
				Type: "OBJECT",
				Properties: map[string]*genai.Schema{
					"issue": {
						Type:        "STRING",
						Description: "Description of the failure mode",
					},
					"sample_size": {
						Type:        "INTEGER",
						Description: "Number of trades exhibiting this issue",
					},
					"win_rate": {
						Type:        "NUMBER",
						Description: "Win rate for trades with this issue",
						Minimum:     float64Ptr(0),
						Maximum:     float64Ptr(1),
					},
					"avg_rr": {
						Type:        "NUMBER",
						Description: "Average R:R for trades with this issue",
					},
					"lesson": {
						Type:        "STRING",
						Description: "Lesson learned from this failure mode",
					},
					"action": {
						Type:        "STRING",
						Description: "Specific action to prevent this issue",
					},
				},
				Required: []string{"issue", "sample_size", "lesson", "action"},
			},
		},
		"agent_calibration": {
			Type:        "OBJECT",
			Description: "Analysis of agent confidence calibration",
		},
		"recommendations": {
			Type:        "ARRAY",
			Description: "Actionable recommendations for system improvement",
			Items: &genai.Schema{
				Type: "STRING",
			},
		},
		// Explainability fields
		"reasoning_trace": {
			Type:        "ARRAY",
			Description: "Step-by-step analysis process from data to recommendations",
			Items:       ReasoningStepSchema,
		},
		"evidence": {
			Type:        "OBJECT",
			Description: "Data sources and quality used in analysis",
			Properties:  EvidenceSchema.Properties,
			Required:    EvidenceSchema.Required,
		},
		"alternatives_considered": {
			Type:        "ARRAY",
			Description: "Alternative interpretations or recommendations that were considered",
			Items:       AlternativeSchema,
		},
	},
	Required: []string{"summary", "validated_patterns", "failure_modes", "recommendations", "reasoning_trace", "evidence"},
}
