package postgres

import (
	"os"
	"testing"

	"prometheus/internal/adapters/config"
)

var cfg *config.Config

// TestMain runs before all tests in this package
func TestMain(m *testing.M) {
	// Load config which handles ENV=test logic internally
	cfg, _ = config.Load()

	// Run tests
	code := m.Run()

	os.Exit(code)
}
