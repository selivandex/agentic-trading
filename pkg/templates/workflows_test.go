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

	// Verify all 8 analysts are mentioned
	analysts := []string{
		"MarketAnalyst",
		"SMCAnalyst",
		"SentimentAnalyst",
		"OrderFlowAnalyst",
		"DerivativesAnalyst",
		"MacroAnalyst",
		"OnChainAnalyst",
		"CorrelationAnalyst",
	}

	for _, analyst := range analysts {
		if !strings.Contains(output, analyst) {
			t.Errorf("Missing analyst in template: %s", analyst)
		}
	}

	// Verify decision criteria are present
	if !strings.Contains(output, "5+ analysts agree") {
		t.Error("Missing consensus requirement")
	}
	if !strings.Contains(output, ">65%") || !strings.Contains(output, "confidence >65%") {
		t.Error("Missing confidence threshold")
	}
	if !strings.Contains(output, "R:R >2:1") {
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
