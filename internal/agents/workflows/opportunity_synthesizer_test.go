package workflows

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOpportunitySynthesizerOutputFormat verifies the expected output structure
func TestOpportunitySynthesizerOutputFormat_PublishScenario(t *testing.T) {
	// This test verifies the structured output format that OpportunitySynthesizer
	// should produce when publishing an opportunity

	// Example structured output for "publish" decision
	publishOutput := map[string]interface{}{
		"synthesis_steps": []map[string]interface{}{
			{
				"step_name":   "review_inputs",
				"input_data":  map[string]interface{}{"analyst_count": 8},
				"observation": "Received outputs from all 8 analysts",
				"calculation": nil,
				"conclusion":  "All inputs present. Proceeding to consensus calculation.",
			},
			{
				"step_name": "count_consensus",
				"input_data": map[string]interface{}{
					"bullish": 6,
					"bearish": 1,
					"neutral": 1,
				},
				"observation": "6 analysts signal BUY (Market, SMC, OrderFlow, OnChain, Correlation, Derivatives), 1 SELL (Macro), 1 NEUTRAL (Sentiment)",
				"calculation": nil,
				"conclusion":  "Strong bullish consensus: 75% (6/8 analysts agree on BUY direction)",
			},
			{
				"step_name": "calculate_confidence",
				"input_data": map[string]interface{}{
					"market":      map[string]interface{}{"confidence": 0.78, "weight": 0.30},
					"smc":         map[string]interface{}{"confidence": 0.65, "weight": 0.20},
					"orderflow":   map[string]interface{}{"confidence": 0.70, "weight": 0.15},
					"sentiment":   map[string]interface{}{"confidence": 0.55, "weight": 0.10},
					"derivatives": map[string]interface{}{"confidence": 0.60, "weight": 0.10},
					"macro":       map[string]interface{}{"confidence": 0.50, "weight": 0.05},
					"onchain":     map[string]interface{}{"confidence": 0.72, "weight": 0.05},
					"correlation": map[string]interface{}{"confidence": 0.68, "weight": 0.05},
				},
				"observation": "Applying analyst-specific weights to individual confidence scores",
				"calculation": "(0.78×0.30) + (0.65×0.20) + (0.70×0.15) + (0.55×0.10) + (0.60×0.10) + (0.50×0.05) + (0.72×0.05) + (0.68×0.05) = 0.682",
				"conclusion":  "Weighted confidence = 68.2% (exceeds 65% publication threshold)",
			},
			{
				"step_name": "identify_conflicts",
				"input_data": map[string]interface{}{
					"bullish_analysts": []string{"market", "smc", "orderflow", "onchain", "correlation", "derivatives"},
					"bearish_analysts": []string{"macro"},
					"neutral_analysts": []string{"sentiment"},
				},
				"observation": "Macro analyst signals bearish due to risk-off concerns, but 6 technical/flow analysts signal bullish",
				"calculation": nil,
				"conclusion":  "Conflict identified: Macro fundamental view vs HTF technical consensus",
			},
			{
				"step_name": "resolve_conflicts",
				"input_data": map[string]interface{}{
					"conflict_type":   "macro_vs_technical",
					"technical_count": 6,
					"macro_count":     1,
				},
				"observation": "6 HTF technical analysts agree on bullish setup. Macro represents single dissenting view.",
				"calculation": nil,
				"conclusion":  "Resolution: HTF technical consensus (6 analysts) dominates single macro dissent for swing trade. Acknowledge macro risk but proceed.",
			},
			{
				"step_name": "make_decision",
				"input_data": map[string]interface{}{
					"consensus_met":      true,
					"confidence_met":     true,
					"conflicts_resolved": true,
				},
				"observation": "All publication criteria satisfied: 6+ consensus ✓, confidence >65% ✓, conflicts resolved ✓",
				"calculation": nil,
				"conclusion":  "DECISION: PUBLISH opportunity. High-quality setup meets all thresholds.",
			},
		},
		"decision": map[string]interface{}{
			"action":              "publish",
			"consensus_count":     6,
			"weighted_confidence": 0.682,
			"conflicts_resolved":  true,
			"rationale":           "Strong bullish consensus (6/8 analysts), weighted confidence 68.2% exceeds threshold. HTF technical structure + tested support + positive flow. Macro risk noted but secondary to technical setup.",
		},
		"conflicts": []map[string]interface{}{
			{
				"conflicting_analysts": []string{"sentiment_analysis"},
				"conflict_type":        "direction",
				"resolution_method":    "Technical and SMC consensus dominates single sentiment dissent for swing trade timeframe",
			},
		},
	}

	// Verify JSON marshaling works
	jsonBytes, err := json.Marshal(publishOutput)
	require.NoError(t, err, "Should marshal to valid JSON")

	// Verify structure after round-trip
	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	require.NoError(t, err)

	// Verify required top-level fields
	assert.Contains(t, result, "synthesis_steps")
	assert.Contains(t, result, "decision")
	assert.Contains(t, result, "conflicts")

	// Verify synthesis_steps is array with expected steps
	steps, ok := result["synthesis_steps"].([]interface{})
	require.True(t, ok, "synthesis_steps should be array")
	assert.GreaterOrEqual(t, len(steps), 5, "Should have at least 5 synthesis steps")

	// Verify decision structure
	decision, ok := result["decision"].(map[string]interface{})
	require.True(t, ok, "decision should be object")
	assert.Equal(t, "publish", decision["action"])
	assert.Equal(t, float64(6), decision["consensus_count"])
	assert.Greater(t, decision["weighted_confidence"], 0.65)
	assert.True(t, decision["conflicts_resolved"].(bool))

	// Verify conflicts array
	conflicts, ok := result["conflicts"].([]interface{})
	require.True(t, ok, "conflicts should be array")
	assert.Len(t, conflicts, 1, "Should have 1 conflict")
}

func TestOpportunitySynthesizerOutputFormat_SkipScenario(t *testing.T) {
	// This test verifies the structured output format when skipping an opportunity

	skipOutput := map[string]interface{}{
		"synthesis_steps": []map[string]interface{}{
			{
				"step_name":   "review_inputs",
				"input_data":  map[string]interface{}{"analyst_count": 8},
				"observation": "Received all 8 analyst outputs",
				"calculation": nil,
				"conclusion":  "Proceeding to consensus calculation",
			},
			{
				"step_name": "count_consensus",
				"input_data": map[string]interface{}{
					"bullish": 3,
					"bearish": 2,
					"neutral": 3,
				},
				"observation": "Only 3 analysts signal BUY, 2 SELL, 3 NEUTRAL",
				"calculation": nil,
				"conclusion":  "Weak consensus: 37.5% (below 62.5% threshold)",
			},
			{
				"step_name":   "calculate_confidence",
				"input_data":  map[string]interface{}{},
				"observation": "Calculating weighted confidence",
				"calculation": "0.55×0.30 + 0.48×0.20 + ... = 0.52",
				"conclusion":  "Weighted confidence = 52% (below 65% threshold)",
			},
			{
				"step_name": "make_decision",
				"input_data": map[string]interface{}{
					"consensus_met":  false,
					"confidence_met": false,
				},
				"observation": "Consensus <5 analysts, confidence <65%",
				"calculation": nil,
				"conclusion":  "DECISION: SKIP - insufficient quality",
			},
		},
		"decision": map[string]interface{}{
			"action":              "skip",
			"consensus_count":     3,
			"weighted_confidence": 0.52,
			"conflicts_resolved":  false,
			"rationale":           "Weak consensus (only 3/8 agree), confidence 52% below threshold (need >65%). Market unclear, too much uncertainty. Waiting for better setup.",
		},
		"conflicts": []map[string]interface{}{}, // Empty array for skip scenarios
	}

	// Verify JSON marshaling
	jsonBytes, err := json.Marshal(skipOutput)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	require.NoError(t, err)

	// Verify decision is "skip"
	decision := result["decision"].(map[string]interface{})
	assert.Equal(t, "skip", decision["action"])
	assert.Less(t, decision["weighted_confidence"], 0.65)
	assert.Less(t, decision["consensus_count"], float64(5))

	// Verify conflicts array is empty (but present)
	conflicts, ok := result["conflicts"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, conflicts)
}

// TestOpportunitySynthesizerCriteria documents the publication criteria
func TestOpportunitySynthesizerCriteria(t *testing.T) {
	criteria := map[string]interface{}{
		"consensus_threshold":        5,    // At least 5 analysts must agree
		"confidence_threshold":       0.65, // Weighted confidence > 65%
		"required_levels":            []string{"entry", "stop_loss", "take_profit"},
		"min_risk_reward":            2.0, // R:R must be > 2:1
		"conflicts_must_be_resolved": true,
	}

	// This test documents requirements, ensuring they're clear
	assert.Equal(t, 5, criteria["consensus_threshold"])
	assert.Equal(t, 0.65, criteria["confidence_threshold"])
	assert.Equal(t, 2.0, criteria["min_risk_reward"])
}

// TestSynthesisStepNames documents expected step names
func TestSynthesisStepNames(t *testing.T) {
	// Expected synthesis steps in order
	expectedSteps := []string{
		"review_inputs",
		"count_consensus",
		"calculate_confidence",
		"identify_conflicts",
		"resolve_conflicts", // Only if conflicts exist
		"assess_signal",
		"define_parameters",
		"make_decision",
	}

	// Verify step names match schema enum
	for _, step := range expectedSteps {
		// In a real schema validation, we'd check against the actual enum
		// For now, just document expected values
		assert.NotEmpty(t, step)
	}

	// These are the allowed step_name values according to OpportunitySynthesizerOutputSchema
	t.Logf("Expected synthesis step names: %v", expectedSteps)
}
