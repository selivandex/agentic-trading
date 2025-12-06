// Package relay provides utilities for implementing Relay-style cursor pagination, scopes and filters
package relay

// FilterType represents the type of filter input
type FilterType string

const (
	FilterTypeText        FilterType = "TEXT"
	FilterTypeNumber      FilterType = "NUMBER"
	FilterTypeDate        FilterType = "DATE"
	FilterTypeSelect      FilterType = "SELECT"
	FilterTypeMultiselect FilterType = "MULTISELECT"
	FilterTypeBoolean     FilterType = "BOOLEAN"
	FilterTypeDateRange   FilterType = "DATE_RANGE"
	FilterTypeNumberRange FilterType = "NUMBER_RANGE"
)

// FilterOption represents an option for select/multiselect filters
type FilterOption struct {
	Value string
	Label string
}

// Filter represents a dynamic filter definition
type Filter struct {
	ID               string
	Name             string
	Type             FilterType
	Options          []FilterOption         // Static options
	OptionsQuery     *string                // GraphQL query name for dynamic options
	OptionsQueryArgs map[string]interface{} // Args for options query
	DefaultValue     *string
	Placeholder      *string
}

// FilterDefinition defines a filter with its metadata
type FilterDefinition struct {
	ID               string
	Name             string
	Type             FilterType
	Options          []FilterOption         // Static options for SELECT/MULTISELECT
	OptionsQuery     *string                // GraphQL query name for dynamic options
	OptionsQueryArgs map[string]interface{} // Args for options query
	DefaultValue     *string
	Placeholder      *string
}

// ApplyFilters is a generic function to apply filters to items
// filters: map of filter_id -> filter_value (from GraphQL input)
// filterDefs: definitions of available filters
// applyFunc: function that applies a single filter to an item
func ApplyFilters[T any](
	items []T,
	filters map[string]interface{},
	filterDefs []FilterDefinition,
	applyFunc func(item T, filterID string, filterValue interface{}) bool,
) []T {
	// If no filters provided, return all items
	if len(filters) == 0 {
		return items
	}

	// Filter items
	filtered := make([]T, 0, len(items))
	for _, item := range items {
		matchesAll := true

		// Check each active filter
		for filterID, filterValue := range filters {
			// Skip nil values
			if filterValue == nil {
				continue
			}

			// Apply filter using provided function
			if !applyFunc(item, filterID, filterValue) {
				matchesAll = false
				break
			}
		}

		if matchesAll {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

// GetFilterDefinitions returns filter definitions as relay.Filter structs
func GetFilterDefinitions(defs []FilterDefinition) []Filter {
	filters := make([]Filter, len(defs))
	for i, def := range defs {
		filters[i] = Filter{
			ID:               def.ID,
			Name:             def.Name,
			Type:             def.Type,
			Options:          def.Options,
			OptionsQuery:     def.OptionsQuery,
			OptionsQueryArgs: def.OptionsQueryArgs,
			DefaultValue:     def.DefaultValue,
			Placeholder:      def.Placeholder,
		}
	}
	return filters
}
