package resolvers

import (
	"prometheus/pkg/relay"
)

// getAgentFilterDefinitions returns filter definitions for agents
// These define the available filters that can be applied to agent lists
func getAgentFilterDefinitions() []relay.FilterDefinition {
	return []relay.FilterDefinition{
		{
			ID:          "temperature_range",
			Name:        "Temperature Range",
			Type:        relay.FilterTypeNumberRange,
			Placeholder: strPtr("Enter min-max"),
		},
		{
			ID:          "max_tokens_range",
			Name:        "Max Tokens Range",
			Type:        relay.FilterTypeNumberRange,
			Placeholder: strPtr("Enter min-max"),
		},
		{
			ID:          "max_cost_per_run_range",
			Name:        "Max Cost Per Run Range",
			Type:        relay.FilterTypeNumberRange,
			Placeholder: strPtr("Enter min-max"),
		},
		{
			ID:          "timeout_seconds_range",
			Name:        "Timeout Seconds Range",
			Type:        relay.FilterTypeNumberRange,
			Placeholder: strPtr("Enter min-max"),
		},
		{
			ID:   "is_active",
			Name: "Is Active",
			Type: relay.FilterTypeSelect,
			Options: []relay.FilterOption{
				{Value: "true", Label: "Yes"},
				{Value: "false", Label: "No"},
			},
		},
		{
			ID:          "version_range",
			Name:        "Version Range",
			Type:        relay.FilterTypeNumberRange,
			Placeholder: strPtr("Enter min-max"),
		},
		{
			ID:          "created_at_range",
			Name:        "Created At",
			Type:        relay.FilterTypeDateRange,
			Placeholder: strPtr("Select date range"),
		},
		{
			ID:          "updated_at_range",
			Name:        "Updated At",
			Type:        relay.FilterTypeDateRange,
			Placeholder: strPtr("Select date range"),
		},
	}
}
