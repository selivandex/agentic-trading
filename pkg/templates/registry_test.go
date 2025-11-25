package templates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryLoadAndRender(t *testing.T) {
	base := t.TempDir()
	agentDir := filepath.Join(base, "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	tplPath := filepath.Join(agentDir, "market_analyst.tmpl")
	initial := "Hello {{.Name}}"
	if err := os.WriteFile(tplPath, []byte(initial), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	reg, err := NewRegistry(base)
	if err != nil {
		t.Fatalf("init registry: %v", err)
	}

	tmpl, err := reg.GetTemplate("agents/market_analyst")
	if err != nil {
		t.Fatalf("get template: %v", err)
	}

	rendered, err := tmpl.Render(map[string]string{"Name": "Alice"})
	if err != nil {
		t.Fatalf("render template: %v", err)
	}
	if rendered != "Hello Alice" {
		t.Fatalf("unexpected render result: %s", rendered)
	}

	updated := "Hi {{.Name}}"
	if err := os.WriteFile(tplPath, []byte(updated), 0o644); err != nil {
		t.Fatalf("rewrite template: %v", err)
	}

	rendered, err = tmpl.Render(map[string]string{"Name": "Bob"})
	if err != nil {
		t.Fatalf("render template after update: %v", err)
	}
	if rendered != "Hello Bob" {
		t.Fatalf("expected embedded registry to keep initial content, got: %s", rendered)
	}
}

func TestRegistryLazyLoad(t *testing.T) {
	base := t.TempDir()
	reg, err := NewRegistry(base)
	if err != nil {
		t.Fatalf("init registry: %v", err)
	}

	path := filepath.Join(base, "notifications", "trade_opened.tmpl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create dirs: %v", err)
	}

	content := "Trade {{.Symbol}}"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	rendered, err := reg.Render("notifications/trade_opened", map[string]string{"Symbol": "BTCUSDT"})
	if err != nil {
		t.Fatalf("render lazily loaded template: %v", err)
	}

	if rendered != "Trade BTCUSDT" {
		t.Fatalf("unexpected render output: %s", rendered)
	}
}
