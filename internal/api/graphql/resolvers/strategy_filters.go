package resolvers

import (
	"prometheus/pkg/relay"
)

// getStrategyFilterDefinitions returns filter definitions for strategies
// These define the available filters that can be applied to strategy lists
func getStrategyFilterDefinitions() []relay.FilterDefinition {
	return []relay.FilterDefinition{
		{
			ID:   "risk_tolerance",
			Name: "Risk Tolerance",
			Type: relay.FilterTypeSelect,
			Options: []relay.FilterOption{
				{Value: "CONSERVATIVE", Label: "Conservative"},
				{Value: "MODERATE", Label: "Moderate"},
				{Value: "AGGRESSIVE", Label: "Aggressive"},
			},
			Placeholder: strPtr("Select risk tolerance"),
		},
		{
			ID:   "market_type",
			Name: "Market Type",
			Type: relay.FilterTypeSelect,
			Options: []relay.FilterOption{
				{Value: "SPOT", Label: "Spot"},
				{Value: "FUTURES", Label: "Futures"},
			},
			Placeholder: strPtr("Select market type"),
		},
		{
			ID:          "capital_range",
			Name:        "Capital Range",
			Type:        relay.FilterTypeNumberRange,
			Placeholder: strPtr("Select capital range"),
		},
		{
			ID:   "rebalance_frequency",
			Name: "Rebalance Frequency",
			Type: relay.FilterTypeMultiselect,
			Options: []relay.FilterOption{
				{Value: "DAILY", Label: "Daily"},
				{Value: "WEEKLY", Label: "Weekly"},
				{Value: "MONTHLY", Label: "Monthly"},
				{Value: "NEVER", Label: "Never"},
			},
			Placeholder: strPtr("Select rebalance frequencies"),
		},
		{
			ID:          "pnl_range",
			Name:        "PnL % Range",
			Type:        relay.FilterTypeNumberRange,
			Placeholder: strPtr("Select PnL range"),
		},
		{
			ID:          "created_at_range",
			Name:        "Created Date",
			Type:        relay.FilterTypeDateRange,
			Placeholder: strPtr("Select date range"),
		},
		{
			ID:           "user_id",
			Name:         "User",
			Type:         relay.FilterTypeSelect,
			OptionsQuery: strPtr("users"), // Frontend will call users query
			OptionsQueryArgs: map[string]interface{}{
				"first": 100, // Limit users list
			},
			Placeholder: strPtr("Select user"),
		},
	}
}

// strPtr is a helper to create string pointers
func strPtr(s string) *string {
	return &s
}
