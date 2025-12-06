package resolvers

import (
	"prometheus/pkg/relay"
)

// getAgentScopeDefinitions returns scope definitions for agents
// These are used ONLY for metadata (name, id) in GraphQL responses
// Filtering is done at repository level via SQL WHERE clauses
func getAgentScopeDefinitions() []relay.Scope {
	return []relay.Scope{
		{
			ID:    "all",
			Name:  "All",
			Count: 0, // Will be populated from service
		},
		{
			ID:    "active",
			Name:  "Active",
			Count: 0, // Will be populated from service
		},
		{
			ID:    "not_active",
			Name:  "Not Active",
			Count: 0, // Will be populated from service
		},
	}
}
