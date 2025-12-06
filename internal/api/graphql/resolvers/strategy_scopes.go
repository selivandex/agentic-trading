package resolvers

import (
	"prometheus/pkg/relay"
)

// getStrategyScopeDefinitions returns scope definitions for strategies
// These are used ONLY for metadata (name, id) in GraphQL responses
// Filtering is done at repository level via SQL WHERE clauses
func getStrategyScopeDefinitions() []relay.Scope {
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
			ID:    "closed",
			Name:  "Closed",
			Count: 0,
		},
	}
}
