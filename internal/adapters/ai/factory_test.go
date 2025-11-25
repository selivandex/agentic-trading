package ai

import (
	"testing"

	"prometheus/internal/adapters/config"
)

func TestBuildRegistryReturnsErrorWhenNoKeys(t *testing.T) {
	cfg := config.AIConfig{}
	if _, err := BuildRegistry(cfg); err == nil {
		t.Fatal("expected error when no providers configured")
	}
}

func TestBuildRegistryRegistersProvidedKeys(t *testing.T) {
	cfg := config.AIConfig{
		ClaudeKey:   "c",
		OpenAIKey:   "o",
		DeepSeekKey: "d",
		GeminiKey:   "g",
	}

	registry, err := BuildRegistry(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(registry.List()); got != 4 {
		t.Fatalf("expected 4 providers, got %d", got)
	}
}

func TestNormalizeProviderName(t *testing.T) {
	if got := NormalizeProviderName("  OpenAI "); got != "openai" {
		t.Fatalf("unexpected normalized name %s", got)
	}
}
