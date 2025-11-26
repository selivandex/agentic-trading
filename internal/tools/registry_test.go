package tools
import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
func TestRegistry_RiskMetadata(t *testing.T) {
	registry := NewRegistry()
	t.Run("GetMetadata", func(t *testing.T) {
		// Test getting metadata for a known tool
		meta, ok := registry.GetMetadata("place_order")
		require.True(t, ok, "place_order should be in registry")
		assert.Equal(t, "place_order", meta.Name)
		assert.Equal(t, RiskLevelMedium, meta.RiskLevel)
		assert.True(t, meta.RequiresAuth)
		// Test getting metadata for emergency tool
		meta, ok = registry.GetMetadata("emergency_close_all")
		require.True(t, ok, "emergency_close_all should be in registry")
		assert.Equal(t, RiskLevelCritical, meta.RiskLevel)
		// Test unknown tool
		_, ok = registry.GetMetadata("unknown_tool")
		assert.False(t, ok, "unknown tool should not be found")
	})
	t.Run("GetRiskLevel", func(t *testing.T) {
		// Test various risk levels
		assert.Equal(t, RiskLevelNone, registry.GetRiskLevel("get_price"))
		assert.Equal(t, RiskLevelLow, registry.GetRiskLevel("get_balance"))
		assert.Equal(t, RiskLevelMedium, registry.GetRiskLevel("place_order"))
		assert.Equal(t, RiskLevelCritical, registry.GetRiskLevel("emergency_close_all"))
		// Unknown tool returns RiskLevelNone
		assert.Equal(t, RiskLevelNone, registry.GetRiskLevel("unknown_tool"))
	})
	t.Run("IsHighRisk", func(t *testing.T) {
		// Low/None risk tools
		assert.False(t, registry.IsHighRisk("get_price"))
		assert.False(t, registry.IsHighRisk("get_balance"))
		assert.False(t, registry.IsHighRisk("place_order"))
		// Critical risk tools
		assert.True(t, registry.IsHighRisk("emergency_close_all"))
		// Unknown tool is not high risk
		assert.False(t, registry.IsHighRisk("unknown_tool"))
	})
	t.Run("ListByRiskLevel", func(t *testing.T) {
		criticalTools := registry.ListByRiskLevel(RiskLevelCritical)
		assert.Contains(t, criticalTools, "emergency_close_all")
		mediumTools := registry.ListByRiskLevel(RiskLevelMedium)
		assert.Contains(t, mediumTools, "place_order")
		noneTools := registry.ListByRiskLevel(RiskLevelNone)
		assert.Contains(t, noneTools, "get_price")
		assert.Contains(t, noneTools, "rsi")
		assert.NotEmpty(t, noneTools, "should have many read-only tools")
	})
	t.Run("Register and Get", func(t *testing.T) {
		// Create a mock tool
		mockTool := &mockToolImpl{name: "test_tool"}
		// Register it
		registry.Register("test_tool", mockTool)
		// Retrieve it
		retrieved, ok := registry.Get("test_tool")
		require.True(t, ok)
		assert.Equal(t, mockTool, retrieved)
	})
}
// mockToolImpl is a minimal implementation of tool.Tool for testing
type mockToolImpl struct {
	name string
}
func (m *mockToolImpl) Name() string        { return m.name }
func (m *mockToolImpl) Description() string { return "Test tool" }
func (m *mockToolImpl) IsLongRunning() bool { return false }
