// Package relay provides utilities for implementing Relay-style cursor pagination and scopes
package relay

// Scope represents a filter scope with count
type Scope struct {
	ID    string
	Name  string
	Count int
}

// ScopeFilter is a function that filters items based on scope
type ScopeFilter[T any] func(item T) bool

// ScopeDefinition defines a scope with its ID, name and filter function
type ScopeDefinition[T any] struct {
	ID     string
	Name   string
	Filter ScopeFilter[T]
}

// CalculateScopes calculates counts for all defined scopes
// items: all items to count
// scopes: definitions of scopes to calculate
// Returns: array of Scope with calculated counts
func CalculateScopes[T any](items []T, scopeDefs []ScopeDefinition[T]) []Scope {
	result := make([]Scope, len(scopeDefs))

	for i, scopeDef := range scopeDefs {
		count := 0
		for _, item := range items {
			if scopeDef.Filter(item) {
				count++
			}
		}

		result[i] = Scope{
			ID:    scopeDef.ID,
			Name:  scopeDef.Name,
			Count: count,
		}
	}

	return result
}

// FilterByScope filters items by scope ID
// items: all items to filter
// scopeID: ID of the scope to apply
// scopeDefs: definitions of scopes
// Returns: filtered items
func FilterByScope[T any](items []T, scopeID *string, scopeDefs []ScopeDefinition[T]) []T {
	// If no scope specified, return all items
	if scopeID == nil || *scopeID == "" || *scopeID == "all" {
		return items
	}

	// Find the scope definition
	var scopeDef *ScopeDefinition[T]
	for i := range scopeDefs {
		if scopeDefs[i].ID == *scopeID {
			scopeDef = &scopeDefs[i]
			break
		}
	}

	// If scope not found, return all items (fallback)
	if scopeDef == nil {
		return items
	}

	// Apply filter
	filtered := make([]T, 0, len(items))
	for _, item := range items {
		if scopeDef.Filter(item) {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

// ConnectionWithScopes extends Connection with scopes support
type ConnectionWithScopes[T any] struct {
	Edges      []Edge[T]
	PageInfo   PageInfo
	TotalCount int
	Scopes     []Scope
}

// NewConnectionWithScopes creates a new connection with scopes
func NewConnectionWithScopes[T any](
	items []T,
	allItems []T, // all items before pagination, for scope counts
	totalCount int,
	params PaginationParams,
	startOffset int,
	scopeDefs []ScopeDefinition[T],
) (*ConnectionWithScopes[T], error) {
	// Create base connection
	conn, err := NewConnection(items, totalCount, params, startOffset)
	if err != nil {
		return nil, err
	}

	// Calculate scopes from all items (before pagination)
	scopes := CalculateScopes(allItems, scopeDefs)

	return &ConnectionWithScopes[T]{
		Edges:      conn.Edges,
		PageInfo:   conn.PageInfo,
		TotalCount: conn.TotalCount,
		Scopes:     scopes,
	}, nil
}
