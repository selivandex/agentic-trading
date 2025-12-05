package ai

import (
	"context"
	"strings"
	"testing"
)

func TestProvidersExposeModels(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		provider Provider
	}{
		{name: "claude", provider: NewClaudeProvider("key", defaultTimeout(), nil)},
		{name: "openai", provider: NewOpenAIProvider("key", defaultTimeout(), nil)},
		{name: "deepseek", provider: NewDeepSeekProvider("key", defaultTimeout(), nil)},
		{name: "gemini", provider: NewGeminiProvider("key", defaultTimeout(), nil)},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			models, err := tt.provider.ListModels(ctx)
			if err != nil {
				t.Fatalf("list models failed: %v", err)
			}

			if len(models) == 0 {
				t.Fatalf("expected models for %s", tt.name)
			}

			// Case-insensitive lookup
			info, err := tt.provider.GetModel(ctx, strings.ToUpper(models[0].Name))
			if err != nil {
				t.Fatalf("get model failed: %v", err)
			}

			if info.Name != models[0].Name {
				t.Fatalf("expected %s, got %s", models[0].Name, info.Name)
			}

			if !tt.provider.SupportsStreaming() || !tt.provider.SupportsTools() {
				t.Fatalf("expected streaming and tools support for %s", tt.name)
			}

			if _, err := tt.provider.GetModel(ctx, "missing-model"); err == nil {
				t.Fatalf("expected error for missing model on %s", tt.name)
			}
		})
	}
}
