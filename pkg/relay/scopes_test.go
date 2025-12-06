package relay

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test item type
type TestItem struct {
	ID     int
	Status string
	Active bool
}

func TestCalculateScopes(t *testing.T) {
	items := []TestItem{
		{ID: 1, Status: "active", Active: true},
		{ID: 2, Status: "active", Active: true},
		{ID: 3, Status: "paused", Active: true},
		{ID: 4, Status: "closed", Active: false},
		{ID: 5, Status: "active", Active: true},
	}

	scopeDefs := []ScopeDefinition[TestItem]{
		{
			ID:   "all",
			Name: "All",
			Filter: func(item TestItem) bool {
				return true
			},
		},
		{
			ID:   "active",
			Name: "Active",
			Filter: func(item TestItem) bool {
				return item.Status == "active"
			},
		},
		{
			ID:   "paused",
			Name: "Paused",
			Filter: func(item TestItem) bool {
				return item.Status == "paused"
			},
		},
		{
			ID:   "closed",
			Name: "Closed",
			Filter: func(item TestItem) bool {
				return item.Status == "closed"
			},
		},
	}

	scopes := CalculateScopes(items, scopeDefs)

	assert.Len(t, scopes, 4)
	assert.Equal(t, "all", scopes[0].ID)
	assert.Equal(t, 5, scopes[0].Count)
	assert.Equal(t, "active", scopes[1].ID)
	assert.Equal(t, 3, scopes[1].Count)
	assert.Equal(t, "paused", scopes[2].ID)
	assert.Equal(t, 1, scopes[2].Count)
	assert.Equal(t, "closed", scopes[3].ID)
	assert.Equal(t, 1, scopes[3].Count)
}

func TestFilterByScope(t *testing.T) {
	items := []TestItem{
		{ID: 1, Status: "active", Active: true},
		{ID: 2, Status: "active", Active: true},
		{ID: 3, Status: "paused", Active: true},
		{ID: 4, Status: "closed", Active: false},
	}

	scopeDefs := []ScopeDefinition[TestItem]{
		{
			ID:   "all",
			Name: "All",
			Filter: func(item TestItem) bool {
				return true
			},
		},
		{
			ID:   "active",
			Name: "Active",
			Filter: func(item TestItem) bool {
				return item.Status == "active"
			},
		},
		{
			ID:   "paused",
			Name: "Paused",
			Filter: func(item TestItem) bool {
				return item.Status == "paused"
			},
		},
	}

	t.Run("no scope returns all items", func(t *testing.T) {
		filtered := FilterByScope(items, nil, scopeDefs)
		assert.Len(t, filtered, 4)
	})

	t.Run("all scope returns all items", func(t *testing.T) {
		scopeID := "all"
		filtered := FilterByScope(items, &scopeID, scopeDefs)
		assert.Len(t, filtered, 4)
	})

	t.Run("active scope filters correctly", func(t *testing.T) {
		scopeID := "active"
		filtered := FilterByScope(items, &scopeID, scopeDefs)
		assert.Len(t, filtered, 2)
		for _, item := range filtered {
			assert.Equal(t, "active", item.Status)
		}
	})

	t.Run("paused scope filters correctly", func(t *testing.T) {
		scopeID := "paused"
		filtered := FilterByScope(items, &scopeID, scopeDefs)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "paused", filtered[0].Status)
	})

	t.Run("unknown scope returns all items", func(t *testing.T) {
		scopeID := "unknown"
		filtered := FilterByScope(items, &scopeID, scopeDefs)
		assert.Len(t, filtered, 4)
	})
}

func TestNewConnectionWithScopes(t *testing.T) {
	allItems := []TestItem{
		{ID: 1, Status: "active", Active: true},
		{ID: 2, Status: "active", Active: true},
		{ID: 3, Status: "paused", Active: true},
		{ID: 4, Status: "closed", Active: false},
		{ID: 5, Status: "active", Active: true},
	}

	// Paginated items (first 2)
	paginatedItems := allItems[:2]

	scopeDefs := []ScopeDefinition[TestItem]{
		{
			ID:   "all",
			Name: "All",
			Filter: func(item TestItem) bool {
				return true
			},
		},
		{
			ID:   "active",
			Name: "Active",
			Filter: func(item TestItem) bool {
				return item.Status == "active"
			},
		},
	}

	first := 2
	params := PaginationParams{
		First: &first,
	}

	conn, err := NewConnectionWithScopes(
		paginatedItems,
		allItems,
		len(allItems),
		params,
		0,
		scopeDefs,
	)

	assert.NoError(t, err)
	assert.NotNil(t, conn)
	assert.Len(t, conn.Edges, 2)
	assert.Equal(t, 5, conn.TotalCount)
	assert.Len(t, conn.Scopes, 2)
	assert.Equal(t, "all", conn.Scopes[0].ID)
	assert.Equal(t, 5, conn.Scopes[0].Count)
	assert.Equal(t, "active", conn.Scopes[1].ID)
	assert.Equal(t, 3, conn.Scopes[1].Count)
}
