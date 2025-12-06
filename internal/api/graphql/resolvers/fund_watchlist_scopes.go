package resolvers

import (
	"prometheus/pkg/relay"
)

// getFundWatchlistScopeDefinitions returns scope definitions for fund watchlist items
// These are used ONLY for metadata (name, id) in GraphQL responses
// Filtering is done at repository level via SQL WHERE clauses
func getFundWatchlistScopeDefinitions() []relay.Scope {
	return []relay.Scope{
		{
			ID:    "all",
			Name:  "All",
			Count: 0, // Will be populated from service
		},
		{
			ID:    "active",
			Name:  "Active",
			Count: 0,
		},
		{
			ID:    "paused",
			Name:  "Paused",
			Count: 0,
		},
		{
			ID:    "inactive",
			Name:  "Inactive",
			Count: 0,
		},
	}
}
