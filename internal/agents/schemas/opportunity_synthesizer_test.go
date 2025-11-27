package schemas

import (
	"encoding/json"
	"testing"
)

func TestOpportunitySynthesizerOutputSchema_ValidStructure(t *testing.T) {
	// Valid structured output matching schema
	validOutput := map[string]interface{}{
		"synthesis_steps": []map[string]interface{}{
			{
				"step_name":   "review_inputs",
				"input_data":  map[string]interface{}{"analyst_count": 8},
				"observation": "Received 8 analyst outputs",
				"calculation": nil,
				"conclusion":  "Proceeding to consensus",
			},
			{
				"step_name":   "count_consensus",
				"input_data":  map[string]interface{}{"bullish": 5, "bearish": 1, "neutral": 2},
				"observation": "5/8 analysts signal BUY",
				"calculation": nil,
				"conclusion":  "Strong bullish consensus",
			},
		},
		"decision": map[string]interface{}{
			"action":              "publish",
			"consensus_count":     5,
			"weighted_confidence": 0.654,
			"conflicts_resolved":  true,
			"rationale":           "Strong consensus, weighted confidence 65.4%",
		},
		"conflicts": []map[string]interface{}{
			{
				"conflicting_analysts": []string{"macro_analyst"},
				"conflict_type":        "direction",
				"resolution_method":    "HTF technical consensus dominates",
			},
		},
	}

	// Marshal to JSON to verify it's valid JSON
	jsonBytes, err := json.Marshal(validOutput)
	if err != nil {
		t.Fatalf("Failed to marshal valid output: %v", err)
	}

	// Unmarshal back to verify structure
	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify required top-level fields exist
	if _, ok := result["synthesis_steps"]; !ok {
		t.Error("Missing required field: synthesis_steps")
	}
	if _, ok := result["decision"]; !ok {
		t.Error("Missing required field: decision")
	}
	if _, ok := result["conflicts"]; !ok {
		t.Error("Missing required field: conflicts")
	}

	// Verify decision object has required fields
	decision, ok := result["decision"].(map[string]interface{})
	if !ok {
		t.Fatal("decision is not an object")
	}

	requiredDecisionFields := []string{"action", "consensus_count", "weighted_confidence", "conflicts_resolved", "rationale"}
	for _, field := range requiredDecisionFields {
		if _, ok := decision[field]; !ok {
			t.Errorf("Missing required decision field: %s", field)
		}
	}
}

func TestOpportunitySynthesizerOutputSchema_SkipDecision(t *testing.T) {
	// Valid "skip" decision
	skipOutput := map[string]interface{}{
		"synthesis_steps": []map[string]interface{}{
			{
				"step_name":   "review_inputs",
				"input_data":  map[string]interface{}{},
				"observation": "Weak consensus detected",
				"calculation": nil,
				"conclusion":  "Insufficient quality",
			},
		},
		"decision": map[string]interface{}{
			"action":              "skip",
			"consensus_count":     3,
			"weighted_confidence": 0.52,
			"conflicts_resolved":  false,
			"rationale":           "Only 3/8 consensus, below threshold",
		},
		"conflicts": []map[string]interface{}{},
	}

	jsonBytes, err := json.Marshal(skipOutput)
	if err != nil {
		t.Fatalf("Failed to marshal skip output: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	decision := result["decision"].(map[string]interface{})
	if decision["action"] != "skip" {
		t.Errorf("Expected action=skip, got %v", decision["action"])
	}
}

func TestOpportunitySynthesizerOutputSchema_SchemaStructure(t *testing.T) {
	// Verify schema is properly defined
	if OpportunitySynthesizerOutputSchema == nil {
		t.Fatal("OpportunitySynthesizerOutputSchema is nil")
	}

	if OpportunitySynthesizerOutputSchema.Type != "OBJECT" {
		t.Errorf("Expected schema type OBJECT, got %s", OpportunitySynthesizerOutputSchema.Type)
	}

	// Check required properties exist
	requiredProps := []string{"synthesis_steps", "decision", "conflicts"}
	for _, prop := range requiredProps {
		if _, ok := OpportunitySynthesizerOutputSchema.Properties[prop]; !ok {
			t.Errorf("Missing required property in schema: %s", prop)
		}
	}

	// Check synthesis_steps is an array
	synthSteps := OpportunitySynthesizerOutputSchema.Properties["synthesis_steps"]
	if synthSteps.Type != "ARRAY" {
		t.Errorf("Expected synthesis_steps to be ARRAY, got %s", synthSteps.Type)
	}

	// Check decision is an object
	decision := OpportunitySynthesizerOutputSchema.Properties["decision"]
	if decision.Type != "OBJECT" {
		t.Errorf("Expected decision to be OBJECT, got %s", decision.Type)
	}

	// Verify Required field lists the required properties
	if len(OpportunitySynthesizerOutputSchema.Required) != 3 {
		t.Errorf("Expected 3 required fields, got %d", len(OpportunitySynthesizerOutputSchema.Required))
	}
}
