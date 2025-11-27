package schemas

import "google.golang.org/genai"

// OpportunitySynthesizerOutputSchema defines the structured Chain-of-Thought output
// for OpportunitySynthesizer agent. This schema enforces transparent reasoning with
// explicit synthesis steps, decision rationale, and conflict resolution.
var OpportunitySynthesizerOutputSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"synthesis_steps": {
			Type:        "ARRAY",
			Description: "Step-by-step synthesis process with observations and conclusions",
			Items: &genai.Schema{
				Type: "OBJECT",
				Properties: map[string]*genai.Schema{
					"step_name": {
						Type:        "STRING",
						Description: "Name of this synthesis step",
						Enum: []string{
							"review_inputs",
							"count_consensus",
							"calculate_confidence",
							"identify_conflicts",
							"resolve_conflicts",
							"assess_signal",
							"define_parameters",
							"make_decision",
						},
					},
					"input_data": {
						Type:        "OBJECT",
						Description: "Structured data relevant to this step (e.g., analyst counts, confidence values)",
					},
					"observation": {
						Type:        "STRING",
						Description: "What was observed/analyzed in this step",
					},
					"calculation": {
						Type:        "STRING",
						Description: "Mathematical calculation performed (e.g., weighted confidence formula). Null if not applicable.",
					},
					"conclusion": {
						Type:        "STRING",
						Description: "What was concluded from this step",
					},
				},
				Required: []string{"step_name", "observation", "conclusion"},
			},
		},
		"decision": {
			Type:        "OBJECT",
			Description: "Final synthesis decision with supporting data",
			Properties: map[string]*genai.Schema{
				"action": {
					Type:        "STRING",
					Description: "Whether to publish the opportunity or skip it",
					Enum:        []string{"publish", "skip"},
				},
				"consensus_count": {
					Type:        "INTEGER",
					Description: "Number of analysts who agreed on the primary direction",
					Minimum:     float64Ptr(0),
					Maximum:     float64Ptr(8),
				},
				"weighted_confidence": {
					Type:        "NUMBER",
					Description: "Weighted confidence score (0-1) based on analyst weights",
					Minimum:     float64Ptr(0),
					Maximum:     float64Ptr(1),
				},
				"conflicts_resolved": {
					Type:        "BOOLEAN",
					Description: "Whether all conflicts between analysts were resolved",
				},
				"rationale": {
					Type:        "STRING",
					Description: "2-3 sentence summary explaining the decision, referencing specific synthesis steps",
				},
			},
			Required: []string{"action", "consensus_count", "weighted_confidence", "conflicts_resolved", "rationale"},
		},
		"conflicts": {
			Type:        "ARRAY",
			Description: "Array of conflicts identified between analysts (empty if no conflicts)",
			Items: &genai.Schema{
				Type: "OBJECT",
				Properties: map[string]*genai.Schema{
					"conflicting_analysts": {
						Type:        "ARRAY",
						Description: "Names of analysts with conflicting views",
						Items: &genai.Schema{
							Type: "STRING",
						},
					},
					"conflict_type": {
						Type:        "STRING",
						Description: "Type of conflict",
						Enum:        []string{"direction", "confidence", "timing", "risk_assessment"},
					},
					"resolution_method": {
						Type:        "STRING",
						Description: "How this conflict was resolved (e.g., 'HTF consensus dominates', 'Higher confidence analyst prioritized')",
					},
				},
				Required: []string{"conflicting_analysts", "conflict_type", "resolution_method"},
			},
		},
	},
	Required: []string{"synthesis_steps", "decision", "conflicts"},
}
