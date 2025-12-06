package resolvers

import (
	"prometheus/pkg/relay"
)

// getUserScopeDefinitions returns scope definitions for users
// These are used ONLY for metadata (name, id) in GraphQL responses
// Filtering is done at repository level via SQL WHERE clauses
func getUserScopeDefinitions() []relay.Scope {
	return []relay.Scope{
		{
			ID:    "all",
			Name:  "All Users",
			Count: 0, // Will be populated from service
		},
		{
			ID:    "active",
			Name:  "Active",
			Count: 0,
		},
		{
			ID:    "inactive",
			Name:  "Inactive",
			Count: 0,
		},
		{
			ID:    "premium",
			Name:  "Premium",
			Count: 0,
		},
		{
			ID:    "free",
			Name:  "Free",
			Count: 0,
		},
	}
}
