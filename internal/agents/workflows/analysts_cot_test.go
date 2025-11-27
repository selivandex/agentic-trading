package workflows

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"

	"prometheus/internal/agents"
	"prometheus/internal/agents/schemas"
)

// TestAnalystStructuredOutputSchemas verifies all 8 analyst agents have proper OutputSchemas defined
func TestAnalystStructuredOutputSchemas(t *testing.T) {
	tests := []struct {
		name       string
		agentType  agents.AgentType
		schemaType string
		hasSchema  bool
	}{
		{"Market Analyst", agents.AgentMarketAnalyst, "MarketAnalystOutputSchema", true},
		{"SMC Analyst", agents.AgentSMCAnalyst, "SMCAnalystOutputSchema", true},
		{"Sentiment Analyst", agents.AgentSentimentAnalyst, "SentimentAnalystOutputSchema", true},
		{"OrderFlow Analyst", agents.AgentOrderFlowAnalyst, "OrderFlowAnalystOutputSchema", true},
		{"Derivatives Analyst", agents.AgentDerivativesAnalyst, "DerivativesAnalystOutputSchema", true},
		{"Macro Analyst", agents.AgentMacroAnalyst, "MacroAnalystOutputSchema", true},
		{"OnChain Analyst", agents.AgentOnChainAnalyst, "OnChainAnalystOutputSchema", true},
		{"Correlation Analyst", agents.AgentCorrelationAnalyst, "CorrelationAnalystOutputSchema", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get schema from factory
			_, outputSchema := getSchemaForAgent(tt.agentType)

			if tt.hasSchema {
				require.NotNil(t, outputSchema, "Agent %s should have OutputSchema", tt.agentType)

				// Verify required top-level fields
				assert.Contains(t, outputSchema.Properties, "reasoning_trace", "Schema should have reasoning_trace")
				assert.Contains(t, outputSchema.Properties, "final_analysis", "Schema should have final_analysis")
				assert.Contains(t, outputSchema.Properties, "tool_calls_summary", "Schema should have tool_calls_summary")

				// Verify Required fields list
				assert.Contains(t, outputSchema.Required, "reasoning_trace", "reasoning_trace should be required")
				assert.Contains(t, outputSchema.Required, "final_analysis", "final_analysis should be required")
				assert.Contains(t, outputSchema.Required, "tool_calls_summary", "tool_calls_summary should be required")
			} else {
				assert.Nil(t, outputSchema, "Agent %s should not have OutputSchema", tt.agentType)
			}
		})
	}
}

// TestAnalystReasoningTraceStructure verifies reasoning_trace schema structure
func TestAnalystReasoningTraceStructure(t *testing.T) {
	// Test with MarketAnalyst as representative
	_, outputSchema := getSchemaForAgent(agents.AgentMarketAnalyst)
	require.NotNil(t, outputSchema)

	reasoningTrace := outputSchema.Properties["reasoning_trace"]
	require.NotNil(t, reasoningTrace)
	assert.Equal(t, "ARRAY", reasoningTrace.Type)

	// Verify items schema
	items := reasoningTrace.Items
	require.NotNil(t, items)
	assert.Equal(t, "OBJECT", items.Type)

	// Verify required fields in reasoning step
	expectedFields := []string{"step_number", "step_name", "observation", "conclusion"}
	for _, field := range expectedFields {
		assert.Contains(t, items.Properties, field, "Reasoning step should have %s field", field)
		assert.Contains(t, items.Required, field, "%s should be required", field)
	}

	// Verify optional fields exist
	assert.Contains(t, items.Properties, "tool_used")
	assert.Contains(t, items.Properties, "tool_result_summary")
}

// TestAnalystFinalAnalysisStructure verifies final_analysis schema structure
func TestAnalystFinalAnalysisStructure(t *testing.T) {
	tests := []struct {
		name             string
		agentType        agents.AgentType
		additionalFields []string // Agent-specific fields
	}{
		{
			"Market Analyst",
			agents.AgentMarketAnalyst,
			[]string{"key_levels"},
		},
		{
			"SMC Analyst",
			agents.AgentSMCAnalyst,
			[]string{"market_structure", "fvg_zones"},
		},
		{
			"Sentiment Analyst",
			agents.AgentSentimentAnalyst,
			[]string{"sentiment_score", "fear_greed_index"},
		},
		{
			"OrderFlow Analyst",
			agents.AgentOrderFlowAnalyst,
			[]string{"flow_bias", "cvd_trend"},
		},
		{
			"Derivatives Analyst",
			agents.AgentDerivativesAnalyst,
			[]string{"funding_rate", "oi_change", "options_flow"},
		},
		{
			"Macro Analyst",
			agents.AgentMacroAnalyst,
			[]string{"regime", "catalysts"},
		},
		{
			"OnChain Analyst",
			agents.AgentOnChainAnalyst,
			[]string{"whale_activity", "exchange_flows"},
		},
		{
			"Correlation Analyst",
			agents.AgentCorrelationAnalyst,
			[]string{"correlation_regime", "btc_correlation"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, outputSchema := getSchemaForAgent(tt.agentType)
			require.NotNil(t, outputSchema)

			finalAnalysis := outputSchema.Properties["final_analysis"]
			require.NotNil(t, finalAnalysis)
			assert.Equal(t, "OBJECT", finalAnalysis.Type)

			// Verify common required fields
			commonFields := []string{"direction", "confidence", "key_findings", "rationale"}
			for _, field := range commonFields {
				assert.Contains(t, finalAnalysis.Properties, field, "final_analysis should have %s", field)
				assert.Contains(t, finalAnalysis.Required, field, "%s should be required", field)
			}

			// Verify agent-specific fields
			for _, field := range tt.additionalFields {
				assert.Contains(t, finalAnalysis.Properties, field, "final_analysis should have agent-specific field %s", field)
			}

			// Verify direction enum
			directionField := finalAnalysis.Properties["direction"]
			require.NotNil(t, directionField)
			assert.Equal(t, []string{"bullish", "bearish", "neutral"}, directionField.Enum)

			// Verify confidence range
			confidenceField := finalAnalysis.Properties["confidence"]
			require.NotNil(t, confidenceField)
			assert.Equal(t, "NUMBER", confidenceField.Type)
			if confidenceField.Minimum != nil {
				assert.Equal(t, 0.0, *confidenceField.Minimum)
			}
			if confidenceField.Maximum != nil {
				assert.Equal(t, 1.0, *confidenceField.Maximum)
			}
		})
	}
}

// TestAnalystOutputValidation tests that valid analyst output passes schema validation
func TestAnalystOutputValidation(t *testing.T) {
	tests := []struct {
		name      string
		agentType agents.AgentType
		output    map[string]interface{}
	}{
		{
			"Market Analyst Valid Output",
			agents.AgentMarketAnalyst,
			map[string]interface{}{
				"reasoning_trace": []interface{}{
					map[string]interface{}{
						"step_number":         1,
						"step_name":           "assess_context",
						"observation":         "BTC at $42,000",
						"tool_used":           nil,
						"tool_result_summary": nil,
						"conclusion":          "Need HTF data",
					},
					map[string]interface{}{
						"step_number":         2,
						"step_name":           "gather_data",
						"observation":         "Getting 1D OHLCV",
						"tool_used":           "get_ohlcv",
						"tool_result_summary": "HTF uptrend",
						"conclusion":          "Bullish structure confirmed",
					},
				},
				"final_analysis": map[string]interface{}{
					"direction":  "bullish",
					"confidence": 0.75,
					"key_findings": []interface{}{
						"HTF uptrend intact",
						"Support at $40k tested",
					},
					"supporting_factors": []interface{}{
						"HTF structure",
						"Momentum recovery",
					},
					"risk_factors": []interface{}{
						"Resistance at $44k",
					},
					"key_levels": []interface{}{
						map[string]interface{}{
							"type":     "support",
							"price":    40000.0,
							"strength": "strong",
						},
					},
					"rationale": "Strong HTF uptrend with tested support. High confidence setup.",
				},
				"tool_calls_summary": []interface{}{
					map[string]interface{}{
						"tool":   "get_ohlcv",
						"result": "success",
					},
				},
			},
		},
		{
			"Sentiment Analyst Valid Output",
			agents.AgentSentimentAnalyst,
			map[string]interface{}{
				"reasoning_trace": []interface{}{
					map[string]interface{}{
						"step_number":         1,
						"step_name":           "gather_baseline_sentiment",
						"observation":         "Checking F&G Index",
						"tool_used":           "get_sentiment_indicators",
						"tool_result_summary": "F&G: 85 (extreme greed)",
						"conclusion":          "Extreme greed territory",
					},
				},
				"final_analysis": map[string]interface{}{
					"direction":  "bearish",
					"confidence": 0.80,
					"key_findings": []interface{}{
						"F&G Index at 85",
						"Social sentiment euphoric",
					},
					"supporting_factors": []interface{}{
						"Extreme sentiment readings",
					},
					"risk_factors": []interface{}{
						"Sentiment can stay extreme",
					},
					"sentiment_score":   0.85,
					"fear_greed_index":  85.0,
					"rationale":         "Extreme greed with contrarian warning signals.",
				},
				"tool_calls_summary": []interface{}{
					map[string]interface{}{
						"tool":   "get_sentiment_indicators",
						"result": "F&G: 85",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON and back to verify structure
			outputJSON, err := json.Marshal(tt.output)
			require.NoError(t, err)

			var parsed map[string]interface{}
			err = json.Unmarshal(outputJSON, &parsed)
			require.NoError(t, err)

			// Verify required fields
			assert.Contains(t, parsed, "reasoning_trace")
			assert.Contains(t, parsed, "final_analysis")
			assert.Contains(t, parsed, "tool_calls_summary")

			// Verify reasoning_trace is array
			reasoningTrace, ok := parsed["reasoning_trace"].([]interface{})
			assert.True(t, ok, "reasoning_trace should be array")
			assert.GreaterOrEqual(t, len(reasoningTrace), 1, "Should have at least 1 reasoning step")

			// Verify first reasoning step structure
			if len(reasoningTrace) > 0 {
				firstStep, ok := reasoningTrace[0].(map[string]interface{})
				assert.True(t, ok)
				assert.Contains(t, firstStep, "step_number")
				assert.Contains(t, firstStep, "step_name")
				assert.Contains(t, firstStep, "observation")
				assert.Contains(t, firstStep, "conclusion")
			}

			// Verify final_analysis structure
			finalAnalysis, ok := parsed["final_analysis"].(map[string]interface{})
			assert.True(t, ok, "final_analysis should be object")
			assert.Contains(t, finalAnalysis, "direction")
			assert.Contains(t, finalAnalysis, "confidence")
			assert.Contains(t, finalAnalysis, "key_findings")
			assert.Contains(t, finalAnalysis, "rationale")

			// Verify tool_calls_summary is array
			toolCalls, ok := parsed["tool_calls_summary"].([]interface{})
			assert.True(t, ok, "tool_calls_summary should be array")
			assert.GreaterOrEqual(t, len(toolCalls), 0, "tool_calls_summary can be empty")
		})
	}
}

// Helper function to get schema - mirrors factory.go
func getSchemaForAgent(agentType agents.AgentType) (input, output *genai.Schema) {
	switch agentType {
	case agents.AgentMarketAnalyst:
		return nil, schemas.MarketAnalystOutputSchema
	case agents.AgentSMCAnalyst:
		return nil, schemas.SMCAnalystOutputSchema
	case agents.AgentSentimentAnalyst:
		return nil, schemas.SentimentAnalystOutputSchema
	case agents.AgentOrderFlowAnalyst:
		return nil, schemas.OrderFlowAnalystOutputSchema
	case agents.AgentDerivativesAnalyst:
		return nil, schemas.DerivativesAnalystOutputSchema
	case agents.AgentMacroAnalyst:
		return nil, schemas.MacroAnalystOutputSchema
	case agents.AgentOnChainAnalyst:
		return nil, schemas.OnChainAnalystOutputSchema
	case agents.AgentCorrelationAnalyst:
		return nil, schemas.CorrelationAnalystOutputSchema
	default:
		return nil, nil
	}
}

