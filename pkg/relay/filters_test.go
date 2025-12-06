package relay

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test item type for filters
type FilterTestItem struct {
	ID             int
	Name           string
	Status         string
	RiskTolerance  string
	AllocatedCapital float64
	Active         bool
}

func TestApplyFilters(t *testing.T) {
	items := []FilterTestItem{
		{ID: 1, Name: "Strategy A", Status: "active", RiskTolerance: "conservative", AllocatedCapital: 1000.0, Active: true},
		{ID: 2, Name: "Strategy B", Status: "active", RiskTolerance: "aggressive", AllocatedCapital: 5000.0, Active: true},
		{ID: 3, Name: "Strategy C", Status: "paused", RiskTolerance: "moderate", AllocatedCapital: 3000.0, Active: false},
		{ID: 4, Name: "Test D", Status: "closed", RiskTolerance: "conservative", AllocatedCapital: 2000.0, Active: false},
	}

	filterDefs := []FilterDefinition{
		{
			ID:   "risk_tolerance",
			Name: "Risk Tolerance",
			Type: FilterTypeSelect,
			Options: []FilterOption{
				{Value: "conservative", Label: "Conservative"},
				{Value: "moderate", Label: "Moderate"},
				{Value: "aggressive", Label: "Aggressive"},
			},
		},
		{
			ID:   "min_capital",
			Name: "Minimum Capital",
			Type: FilterTypeNumber,
		},
	}

	applyFunc := func(item FilterTestItem, filterID string, filterValue interface{}) bool {
		switch filterID {
		case "risk_tolerance":
			if val, ok := filterValue.(string); ok {
				return item.RiskTolerance == val
			}
		case "min_capital":
			if val, ok := filterValue.(float64); ok {
				return item.AllocatedCapital >= val
			}
		}
		return true
	}

	t.Run("no filters returns all items", func(t *testing.T) {
		filters := map[string]interface{}{}
		filtered := ApplyFilters(items, filters, filterDefs, applyFunc)
		assert.Len(t, filtered, 4)
	})

	t.Run("single filter - risk tolerance", func(t *testing.T) {
		filters := map[string]interface{}{
			"risk_tolerance": "conservative",
		}
		filtered := ApplyFilters(items, filters, filterDefs, applyFunc)
		assert.Len(t, filtered, 2)
		assert.Equal(t, "conservative", filtered[0].RiskTolerance)
		assert.Equal(t, "conservative", filtered[1].RiskTolerance)
	})

	t.Run("single filter - min capital", func(t *testing.T) {
		filters := map[string]interface{}{
			"min_capital": 3000.0,
		}
		filtered := ApplyFilters(items, filters, filterDefs, applyFunc)
		assert.Len(t, filtered, 2)
		for _, item := range filtered {
			assert.GreaterOrEqual(t, item.AllocatedCapital, 3000.0)
		}
	})

	t.Run("multiple filters combined", func(t *testing.T) {
		filters := map[string]interface{}{
			"risk_tolerance": "conservative",
			"min_capital":    2000.0,
		}
		filtered := ApplyFilters(items, filters, filterDefs, applyFunc)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "conservative", filtered[0].RiskTolerance)
		assert.GreaterOrEqual(t, filtered[0].AllocatedCapital, 2000.0)
	})

	t.Run("nil filter values are ignored", func(t *testing.T) {
		filters := map[string]interface{}{
			"risk_tolerance": "conservative",
			"min_capital":    nil,
		}
		filtered := ApplyFilters(items, filters, filterDefs, applyFunc)
		assert.Len(t, filtered, 2)
	})
}

func TestGetFilterDefinitions(t *testing.T) {
	placeholder := "Enter value"
	defaultVal := "conservative"
	optionsQuery := "users"
	optionsQueryArgs := map[string]interface{}{
		"role": "ADMIN",
	}

	defs := []FilterDefinition{
		{
			ID:   "risk_tolerance",
			Name: "Risk Tolerance",
			Type: FilterTypeSelect,
			Options: []FilterOption{
				{Value: "conservative", Label: "Conservative"},
				{Value: "moderate", Label: "Moderate"},
			},
			DefaultValue: &defaultVal,
			Placeholder:  &placeholder,
		},
		{
			ID:   "min_capital",
			Name: "Minimum Capital",
			Type: FilterTypeNumber,
		},
		{
			ID:               "user_id",
			Name:             "User",
			Type:             FilterTypeSelect,
			OptionsQuery:     &optionsQuery,
			OptionsQueryArgs: optionsQueryArgs,
			Placeholder:      &placeholder,
		},
	}

	filters := GetFilterDefinitions(defs)

	assert.Len(t, filters, 3)

	// Test static options filter
	assert.Equal(t, "risk_tolerance", filters[0].ID)
	assert.Equal(t, "Risk Tolerance", filters[0].Name)
	assert.Equal(t, FilterTypeSelect, filters[0].Type)
	assert.Len(t, filters[0].Options, 2)
	assert.Equal(t, "conservative", *filters[0].DefaultValue)
	assert.Equal(t, "Enter value", *filters[0].Placeholder)
	assert.Nil(t, filters[0].OptionsQuery)

	// Test number filter
	assert.Equal(t, "min_capital", filters[1].ID)
	assert.Equal(t, FilterTypeNumber, filters[1].Type)
	assert.Nil(t, filters[1].DefaultValue)
	assert.Nil(t, filters[1].Placeholder)
	assert.Nil(t, filters[1].OptionsQuery)

	// Test dynamic options filter
	assert.Equal(t, "user_id", filters[2].ID)
	assert.Equal(t, "User", filters[2].Name)
	assert.Equal(t, FilterTypeSelect, filters[2].Type)
	assert.NotNil(t, filters[2].OptionsQuery)
	assert.Equal(t, "users", *filters[2].OptionsQuery)
	assert.NotNil(t, filters[2].OptionsQueryArgs)
	assert.Equal(t, "ADMIN", filters[2].OptionsQueryArgs["role"])
	assert.Nil(t, filters[2].Options) // No static options for dynamic filter
}
