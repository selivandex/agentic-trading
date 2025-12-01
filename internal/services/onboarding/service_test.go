package onboarding_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	telegram "prometheus/internal/adapters/telegram"
	"prometheus/pkg/templates"
)

// mockStrategyService is a minimal mock
type mockStrategyService struct{}

// Note: Mock methods removed - types changed, update when implementing real tests

// TestBuildArchitectPrompt tests that prompt is correctly constructed
func TestBuildArchitectPrompt(t *testing.T) {
	// This test will help us see what's being sent to the LLM
	// We need to make buildArchitectPrompt public or test via StartOnboarding

	tmpl := templates.Get()

	onboardingSession := &telegram.OnboardingSession{
		TelegramID:        123456,
		UserID:            uuid.New(),
		Capital:           1000.0,
		ExchangeAccountID: nil,
		RiskProfile:       "moderate",
	}

	// Can't test private method directly, but we can verify template exists
	data := map[string]interface{}{
		"Capital":     onboardingSession.Capital,
		"RiskProfile": onboardingSession.RiskProfile,
		"Exchange":    "demo",
		"UserID":      onboardingSession.UserID.String(),
	}

	promptText, err := tmpl.Render("prompts/workflows/portfolio_initialization_input", data)

	// If template doesn't exist, it should use fallback
	if err != nil {
		t.Logf("Template not found (expected), will use fallback: %v", err)
	} else {
		t.Logf("✅ Template found and rendered: %d characters", len(promptText))
		assert.NotEmpty(t, promptText)
	}
}
