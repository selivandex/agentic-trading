package resolvers

import (
	"prometheus/pkg/relay"
)

// getUserFilterDefinitions returns filter definitions for users
// These define the available filters that can be applied to user lists
func getUserFilterDefinitions() []relay.FilterDefinition {
	return []relay.FilterDefinition{
		{
			ID:   "is_active",
			Name: "Status",
			Type: relay.FilterTypeSelect,
			Options: []relay.FilterOption{
				{Value: "true", Label: "Active"},
				{Value: "false", Label: "Inactive"},
			},
			Placeholder: strPtr("Select status"),
		},
		{
			ID:   "is_premium",
			Name: "Account Type",
			Type: relay.FilterTypeSelect,
			Options: []relay.FilterOption{
				{Value: "true", Label: "Premium"},
				{Value: "false", Label: "Free"},
			},
			Placeholder: strPtr("Select account type"),
		},
		{
			ID:   "risk_level",
			Name: "Risk Level",
			Type: relay.FilterTypeMultiselect,
			Options: []relay.FilterOption{
				{Value: "CONSERVATIVE", Label: "Conservative"},
				{Value: "MODERATE", Label: "Moderate"},
				{Value: "AGGRESSIVE", Label: "Aggressive"},
			},
			Placeholder: strPtr("Select risk levels"),
		},
		{
			ID:          "created_at_range",
			Name:        "Registration Date",
			Type:        relay.FilterTypeDateRange,
			Placeholder: strPtr("Select date range"),
		},
		{
			ID:   "language_code",
			Name: "Language",
			Type: relay.FilterTypeMultiselect,
			Options: []relay.FilterOption{
				{Value: "en", Label: "English"},
				{Value: "ru", Label: "Russian"},
				{Value: "es", Label: "Spanish"},
				{Value: "zh", Label: "Chinese"},
				{Value: "de", Label: "German"},
				{Value: "fr", Label: "French"},
			},
			Placeholder: strPtr("Select languages"),
		},
		{
			ID:          "max_positions_range",
			Name:        "Max Positions",
			Type:        relay.FilterTypeNumberRange,
			Placeholder: strPtr("Enter min-max positions"),
		},
	}
}
