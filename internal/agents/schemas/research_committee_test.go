package schemas

import (
	"testing"

	"google.golang.org/genai"
)

func TestAnalystReportSchema_RequiredFields(t *testing.T) {
	schema := AnalystReportSchema

	if schema.Type != "OBJECT" {
		t.Errorf("Expected type OBJECT, got %s", schema.Type)
	}

	requiredFields := schema.Required
	expectedFields := []string{"direction", "confidence", "key_signals", "reasoning", "risk_factors", "strength"}

	if len(requiredFields) != len(expectedFields) {
		t.Errorf("Expected %d required fields, got %d", len(expectedFields), len(requiredFields))
	}

	// Verify all expected fields are present
	fieldMap := make(map[string]bool)
	for _, field := range requiredFields {
		fieldMap[field] = true
	}

	for _, expected := range expectedFields {
		if !fieldMap[expected] {
			t.Errorf("Required field %s is missing", expected)
		}
	}
}

func TestAnalystReportSchema_DirectionEnum(t *testing.T) {
	schema := AnalystReportSchema
	directionSchema := schema.Properties["direction"]

	if directionSchema == nil {
		t.Fatal("direction field is missing")
	}

	expectedEnums := []string{"bullish", "bearish", "neutral"}
	if len(directionSchema.Enum) != len(expectedEnums) {
		t.Errorf("Expected %d enum values, got %d", len(expectedEnums), len(directionSchema.Enum))
	}

	enumMap := make(map[string]bool)
	for _, enum := range directionSchema.Enum {
		enumMap[enum] = true
	}

	for _, expected := range expectedEnums {
		if !enumMap[expected] {
			t.Errorf("Expected enum value %s is missing", expected)
		}
	}
}

func TestAnalystReportSchema_ConfidenceRange(t *testing.T) {
	schema := AnalystReportSchema
	confidenceSchema := schema.Properties["confidence"]

	if confidenceSchema == nil {
		t.Fatal("confidence field is missing")
	}

	if confidenceSchema.Type != "NUMBER" {
		t.Errorf("Expected confidence type NUMBER, got %s", confidenceSchema.Type)
	}

	if *confidenceSchema.Minimum != 0.0 {
		t.Errorf("Expected confidence minimum 0.0, got %f", *confidenceSchema.Minimum)
	}

	if *confidenceSchema.Maximum != 1.0 {
		t.Errorf("Expected confidence maximum 1.0, got %f", *confidenceSchema.Maximum)
	}
}

func TestAnalystReportSchema_StrengthEnum(t *testing.T) {
	schema := AnalystReportSchema
	strengthSchema := schema.Properties["strength"]

	if strengthSchema == nil {
		t.Fatal("strength field is missing")
	}

	expectedEnums := []string{"weak", "moderate", "strong"}
	if len(strengthSchema.Enum) != len(expectedEnums) {
		t.Errorf("Expected %d enum values, got %d", len(expectedEnums), len(strengthSchema.Enum))
	}
}

func TestHeadOfResearchOutputSchema_RequiredFields(t *testing.T) {
	schema := HeadOfResearchOutputSchema

	if schema.Type != "OBJECT" {
		t.Errorf("Expected type OBJECT, got %s", schema.Type)
	}

	expectedFields := []string{
		"decision",
		"synthesis",
		"conflicts",
		"consensus_count",
		"weighted_confidence",
		"pre_mortem_analysis",
		"rationale",
		"reasoning_trace",
		"evidence",
	}

	requiredFields := schema.Required
	if len(requiredFields) != len(expectedFields) {
		t.Errorf("Expected %d required fields, got %d", len(expectedFields), len(requiredFields))
	}

	fieldMap := make(map[string]bool)
	for _, field := range requiredFields {
		fieldMap[field] = true
	}

	for _, expected := range expectedFields {
		if !fieldMap[expected] {
			t.Errorf("Required field %s is missing", expected)
		}
	}
}

func TestHeadOfResearchOutputSchema_DecisionEnum(t *testing.T) {
	schema := HeadOfResearchOutputSchema
	decisionSchema := schema.Properties["decision"]

	if decisionSchema == nil {
		t.Fatal("decision field is missing")
	}

	expectedEnums := []string{"publish", "skip"}
	if len(decisionSchema.Enum) != len(expectedEnums) {
		t.Errorf("Expected %d enum values, got %d", len(expectedEnums), len(decisionSchema.Enum))
	}

	enumMap := make(map[string]bool)
	for _, enum := range decisionSchema.Enum {
		enumMap[enum] = true
	}

	for _, expected := range expectedEnums {
		if !enumMap[expected] {
			t.Errorf("Expected enum value %s is missing", expected)
		}
	}
}

func TestHeadOfResearchOutputSchema_ConsensusCountRange(t *testing.T) {
	schema := HeadOfResearchOutputSchema
	consensusSchema := schema.Properties["consensus_count"]

	if consensusSchema == nil {
		t.Fatal("consensus_count field is missing")
	}

	if consensusSchema.Type != "INTEGER" {
		t.Errorf("Expected consensus_count type INTEGER, got %s", consensusSchema.Type)
	}

	if *consensusSchema.Minimum != 0.0 {
		t.Errorf("Expected consensus_count minimum 0, got %f", *consensusSchema.Minimum)
	}

	if *consensusSchema.Maximum != 4.0 {
		t.Errorf("Expected consensus_count maximum 4, got %f", *consensusSchema.Maximum)
	}
}

func TestHeadOfResearchOutputSchema_WeightedConfidenceRange(t *testing.T) {
	schema := HeadOfResearchOutputSchema
	confidenceSchema := schema.Properties["weighted_confidence"]

	if confidenceSchema == nil {
		t.Fatal("weighted_confidence field is missing")
	}

	if confidenceSchema.Type != "NUMBER" {
		t.Errorf("Expected weighted_confidence type NUMBER, got %s", confidenceSchema.Type)
	}

	if *confidenceSchema.Minimum != 0.0 {
		t.Errorf("Expected weighted_confidence minimum 0.0, got %f", *confidenceSchema.Minimum)
	}

	if *confidenceSchema.Maximum != 1.0 {
		t.Errorf("Expected weighted_confidence maximum 1.0, got %f", *confidenceSchema.Maximum)
	}
}

func TestHeadOfResearchOutputSchema_ConflictsStructure(t *testing.T) {
	schema := HeadOfResearchOutputSchema
	conflictsSchema := schema.Properties["conflicts"]

	if conflictsSchema == nil {
		t.Fatal("conflicts field is missing")
	}

	if conflictsSchema.Type != "ARRAY" {
		t.Errorf("Expected conflicts type ARRAY, got %s", conflictsSchema.Type)
	}

	if conflictsSchema.Items == nil {
		t.Fatal("conflicts items schema is missing")
	}

	itemSchema := conflictsSchema.Items
	if itemSchema.Type != "OBJECT" {
		t.Errorf("Expected conflicts items type OBJECT, got %s", itemSchema.Type)
	}

	// Verify conflict object has required fields
	requiredConflictFields := itemSchema.Required
	expectedConflictFields := []string{"conflicting_analysts", "conflict_type", "resolution"}

	if len(requiredConflictFields) != len(expectedConflictFields) {
		t.Errorf("Expected %d required conflict fields, got %d", len(expectedConflictFields), len(requiredConflictFields))
	}
}

func TestHeadOfResearchOutputSchema_ConflictTypeEnum(t *testing.T) {
	schema := HeadOfResearchOutputSchema
	conflictsSchema := schema.Properties["conflicts"]

	if conflictsSchema == nil || conflictsSchema.Items == nil {
		t.Fatal("conflicts schema structure is missing")
	}

	conflictTypeSchema := conflictsSchema.Items.Properties["conflict_type"]
	if conflictTypeSchema == nil {
		t.Fatal("conflict_type field is missing")
	}

	expectedEnums := []string{"direction", "confidence", "timing", "risk_assessment"}
	if len(conflictTypeSchema.Enum) != len(expectedEnums) {
		t.Errorf("Expected %d conflict_type enum values, got %d", len(expectedEnums), len(conflictTypeSchema.Enum))
	}
}

func TestHeadOfResearchOutputSchema_PricingFields(t *testing.T) {
	schema := HeadOfResearchOutputSchema

	// Verify pricing fields exist
	pricingFields := []string{"entry_price", "stop_loss", "take_profit", "risk_reward_ratio"}

	for _, field := range pricingFields {
		fieldSchema := schema.Properties[field]
		if fieldSchema == nil {
			t.Errorf("Pricing field %s is missing", field)
			continue
		}

		if fieldSchema.Type != "NUMBER" {
			t.Errorf("Expected %s type NUMBER, got %s", field, fieldSchema.Type)
		}

		if fieldSchema.Minimum == nil || *fieldSchema.Minimum != 0.0 {
			t.Errorf("Expected %s minimum 0.0, got %v", field, fieldSchema.Minimum)
		}
	}
}

func TestHeadOfResearchOutputSchema_ExplainabilityFields(t *testing.T) {
	schema := HeadOfResearchOutputSchema

	// Verify explainability fields exist
	explainabilityFields := []string{"reasoning_trace", "evidence", "alternatives_considered"}

	for _, field := range explainabilityFields {
		fieldSchema := schema.Properties[field]
		if fieldSchema == nil {
			t.Errorf("Explainability field %s is missing", field)
		}
	}

	// Verify reasoning_trace is array of ReasoningSteps
	reasoningTraceSchema := schema.Properties["reasoning_trace"]
	if reasoningTraceSchema.Type != "ARRAY" {
		t.Errorf("Expected reasoning_trace type ARRAY, got %s", reasoningTraceSchema.Type)
	}

	// Verify evidence is object with required structure
	evidenceSchema := schema.Properties["evidence"]
	if evidenceSchema.Type != "OBJECT" {
		t.Errorf("Expected evidence type OBJECT, got %s", evidenceSchema.Type)
	}

	// Verify alternatives_considered is array
	alternativesSchema := schema.Properties["alternatives_considered"]
	if alternativesSchema.Type != "ARRAY" {
		t.Errorf("Expected alternatives_considered type ARRAY, got %s", alternativesSchema.Type)
	}
}

// Integration test: Verify all Phase 3 schemas are properly structured
func TestPhase3Schemas_Integration(t *testing.T) {
	schemas := []struct {
		name   string
		schema *genai.Schema
	}{
		{"AnalystReportSchema", AnalystReportSchema},
		{"HeadOfResearchOutputSchema", HeadOfResearchOutputSchema},
	}

	for _, test := range schemas {
		t.Run(test.name, func(t *testing.T) {
			if test.schema == nil {
				t.Fatalf("%s is nil", test.name)
			}

			if test.schema.Type != "OBJECT" {
				t.Errorf("%s: Expected type OBJECT, got %s", test.name, test.schema.Type)
			}

			if test.schema.Properties == nil || len(test.schema.Properties) == 0 {
				t.Errorf("%s: Properties are missing or empty", test.name)
			}

			if len(test.schema.Required) == 0 {
				t.Errorf("%s: No required fields defined", test.name)
			}

			// Verify all required fields exist in properties
			for _, requiredField := range test.schema.Required {
				if test.schema.Properties[requiredField] == nil {
					t.Errorf("%s: Required field %s is missing from properties", test.name, requiredField)
				}
			}
		})
	}
}

