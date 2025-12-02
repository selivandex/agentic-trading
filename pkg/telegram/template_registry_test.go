package telegram

import (
	"strings"
	"testing"
)

func TestNewDefaultTemplateRegistry(t *testing.T) {
	registry, err := NewDefaultTemplateRegistry()
	if err != nil {
		t.Fatalf("Failed to create default template registry: %v", err)
	}

	if registry == nil {
		t.Fatal("Registry is nil")
	}

	// Check that templates are loaded
	templates := registry.List()
	if len(templates) == 0 {
		t.Fatal("No templates loaded")
	}

	t.Logf("Loaded %d templates", len(templates))
}

func TestTemplateRegistry_RenderInvestTemplates(t *testing.T) {
	registry, err := NewDefaultTemplateRegistry()
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	tests := []struct {
		name         string
		templatePath string
		data         map[string]interface{}
		wantContains string
	}{
		{
			name:         "select_exchange template",
			templatePath: "invest/select_exchange",
			data: map[string]interface{}{
				"Exchanges": []map[string]string{
					{"Exchange": "Binance", "Label": "Main Account"},
				},
			},
			wantContains: "exchange",
		},
		{
			name:         "select_market_type template",
			templatePath: "invest/select_market_type",
			data: map[string]interface{}{
				"MarketTypes": []map[string]string{
					{"Label": "Spot Trading"},
					{"Label": "Futures"},
				},
			},
			wantContains: "market",
		},
		{
			name:         "select_risk template",
			templatePath: "invest/select_risk",
			data: map[string]interface{}{
				"RiskProfiles": []map[string]string{
					{"Label": "Conservative"},
					{"Label": "Moderate"},
					{"Label": "Aggressive"},
				},
			},
			wantContains: "risk",
		},
		{
			name:         "enter_amount template",
			templatePath: "invest/enter_amount",
			data:         map[string]interface{}{},
			wantContains: "amount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check template exists
			if !registry.Exists(tt.templatePath) {
				t.Fatalf("Template %s not found. Available: %v", tt.templatePath, registry.List())
			}

			// Render template
			result, err := registry.Render(tt.templatePath, tt.data)
			if err != nil {
				t.Fatalf("Failed to render template %s: %v", tt.templatePath, err)
			}

			if result == "" {
				t.Fatalf("Template %s rendered empty string", tt.templatePath)
			}

			// Check result contains expected text (case insensitive)
			if !containsIgnoreCase(result, tt.wantContains) {
				t.Errorf("Template %s output doesn't contain %q\nGot: %s", tt.templatePath, tt.wantContains, result)
			}

			t.Logf("✓ Template %s rendered successfully", tt.templatePath)
		})
	}
}

func TestTemplateRegistry_TemplateHelpers(t *testing.T) {
	registry, err := NewDefaultTemplateRegistry()
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register test template
	testTemplate := `
Money: {{money 1500.50}}
Bold: {{bold "Hello"}}
Code: {{code "BTC"}}
Checkmark: {{checkmark}}
Percent: {{percent 0.15}}
`
	err = registry.Register("test/helpers", testTemplate)
	if err != nil {
		t.Fatalf("Failed to register test template: %v", err)
	}

	result, err := registry.Render("test/helpers", nil)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	tests := []struct {
		want string
	}{
		{want: "$1500.50"}, // money helper
		{want: "*Hello*"},  // bold helper
		{want: "`BTC`"},    // code helper
		{want: "✅"},        // checkmark helper
		{want: "15.0%"},    // percent helper
	}

	for _, tt := range tests {
		if !strings.Contains(result, tt.want) {
			t.Errorf("Result missing %q\nGot: %s", tt.want, result)
		}
	}
}

func TestTemplateRegistry_MustRender(t *testing.T) {
	registry, err := NewDefaultTemplateRegistry()
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Test valid template
	result := registry.MustRender("common/welcome", map[string]interface{}{
		"FirstName": "John",
	})

	if result == "" {
		t.Error("MustRender returned empty string for valid template")
	}

	// Test invalid template (should return error message, not panic)
	result = registry.MustRender("nonexistent/template", nil)
	if !strings.Contains(result, "Template error") {
		t.Errorf("MustRender should return error message for invalid template, got: %s", result)
	}
}

func TestTemplateRegistry_List(t *testing.T) {
	registry, err := NewDefaultTemplateRegistry()
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	templates := registry.List()

	// Check we have expected templates
	expectedTemplates := []string{
		"invest/select_exchange",
		"invest/select_market_type",
		"invest/select_risk",
		"invest/enter_amount",
		"common/welcome",
		"common/help",
	}

	for _, expected := range expectedTemplates {
		found := false
		for _, tmpl := range templates {
			if tmpl == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected template %q not found in list. Available: %v", expected, templates)
		}
	}

	t.Logf("✓ Found all %d expected templates", len(expectedTemplates))
}

// Helper functions

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
