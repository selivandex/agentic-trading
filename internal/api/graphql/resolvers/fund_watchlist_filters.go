package resolvers

import (
	"prometheus/pkg/relay"
)

// getFundWatchlistFilterDefinitions returns filter definitions for fund watchlist items
// These define the available filters that can be applied to watchlist lists
func getFundWatchlistFilterDefinitions() []relay.FilterDefinition {
	return []relay.FilterDefinition{
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
			ID:   "category",
			Name: "Category",
			Type: relay.FilterTypeSelect,
			Options: []relay.FilterOption{
				{Value: "LARGE_CAP", Label: "Large Cap"},
				{Value: "MID_CAP", Label: "Mid Cap"},
				{Value: "SMALL_CAP", Label: "Small Cap"},
				{Value: "DEFI", Label: "DeFi"},
				{Value: "GAMING", Label: "Gaming"},
				{Value: "AI", Label: "AI"},
				{Value: "MEME", Label: "Meme"},
			},
			Placeholder: strPtr("Select category"),
		},
		{
			ID:   "tier",
			Name: "Tier",
			Type: relay.FilterTypeMultiselect,
			Options: []relay.FilterOption{
				{Value: "1", Label: "Tier 1"},
				{Value: "2", Label: "Tier 2"},
				{Value: "3", Label: "Tier 3"},
				{Value: "4", Label: "Tier 4"},
			},
			Placeholder: strPtr("Select tiers"),
		},
	}
}
