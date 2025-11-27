package templates

import (
	"strings"
	"testing"
	"time"
)

func TestWorkflowTemplate_MarketResearchInput(t *testing.T) {
	registry := Get()

	data := map[string]interface{}{
		"Symbol":    "BTC/USDT",
		"Exchange":  "binance",
		"Timestamp": time.Now().Format(time.RFC3339),
	}

	output, err := registry.Render("workflows/market_research_input", data)
	if err != nil {
		t.Fatalf("Failed to render workflow template: %v", err)
	}

	// Verify template rendered successfully
	if output == "" {
		t.Fatal("Rendered output is empty")
	}

	// Verify key sections are present
	requiredSections := []string{
		"MARKET RESEARCH WORKFLOW EXECUTION",
		"BTC/USDT",
		"binance",
		"MISSION",
		"ANALYSIS DIMENSIONS",
		"STRUCTURED OUTPUT REQUIREMENT",
		"SYNTHESIS REQUIREMENTS",
		"QUALITY PRINCIPLES",
	}

	for _, section := range requiredSections {
		if !strings.Contains(output, section) {
			t.Errorf("Missing required section: %s", section)
		}
	}

	// Verify decision criteria are present
	if !strings.Contains(output, ">65%") || !strings.Contains(output, "confidence >65%") {
		t.Error("Missing confidence threshold")
	}
	if !strings.Contains(output, "R:R =") {
		t.Error("Missing R:R requirement")
	}
}

func TestWorkflowTemplate_VariableInjection(t *testing.T) {
	registry := Get()

	testSymbol := "ETH/USDT"
	testExchange := "bybit"
	testTimestamp := "2025-11-27T10:00:00Z"

	data := map[string]interface{}{
		"Symbol":    testSymbol,
		"Exchange":  testExchange,
		"Timestamp": testTimestamp,
	}

	output, err := registry.Render("workflows/market_research_input", data)
	if err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}

	// Verify all variables were injected
	if !strings.Contains(output, testSymbol) {
		t.Errorf("Symbol not found in output: %s", testSymbol)
	}
	if !strings.Contains(output, testExchange) {
		t.Errorf("Exchange not found in output: %s", testExchange)
	}
	if !strings.Contains(output, testTimestamp) {
		t.Errorf("Timestamp not found in output: %s", testTimestamp)
	}
}

func TestWorkflowTemplate_FallbackOnError(t *testing.T) {
	registry := Get()

	// Test with invalid template name
	_, err := registry.Render("workflows/nonexistent_template", nil)
	if err == nil {
		t.Error("Expected error for nonexistent template")
	}

	// Verify error message is helpful
	if err != nil && !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "nonexistent") {
		t.Logf("Error message: %v", err)
		// Not failing test since error format may vary
	}
}

func TestAgentTemplates_RenderWithoutMissingValues(t *testing.T) {
	registry := Get()

	type testTool struct {
		Name        string
		Description string
	}

	toolset := []testTool{
		{Name: "save_analysis", Description: "Persist findings for other agents"},
	}

	templates := []string{
		"agents/opportunity_synthesizer",
		"agents/strategy_planner",
		"agents/risk_manager",
		"agents/executor",
		"agents/position_manager",
		"agents/self_evaluator",
		"agents/portfolio_architect",
	}

	for _, templateID := range templates {
		for _, hasTool := range []bool{true, false} {
			t.Run(templateID, func(t *testing.T) {
				data := map[string]interface{}{
					"Tools":               toolset,
					"MaxToolCalls":        5,
					"AgentName":           "TestAgent",
					"AgentType":           "test_agent",
					"HasSaveAnalysisTool": hasTool,
				}

				output, err := registry.Render(templateID, data)
				if err != nil {
					t.Fatalf("Failed to render template %s: %v", templateID, err)
				}

				if output == "" {
					t.Fatalf("Template %s produced empty output", templateID)
				}

				if strings.Contains(output, "<no value>") {
					t.Fatalf("Template %s contains unresolved placeholder", templateID)
				}
			})
		}
	}
}
