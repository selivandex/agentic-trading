package resolvers

import (
	"fmt"
	"prometheus/internal/api/graphql/generated"
	"prometheus/internal/domain/strategy"
	"prometheus/pkg/relay"
)

// buildStrategyConnection is a helper function to build GraphQL connection with scopes and filters
// This avoids code duplication between UserStrategies and Strategies resolvers
func buildStrategyConnection(
	items []*strategy.Strategy,
	totalCount int,
	params relay.PaginationParams,
	offset int,
	scopeCounts map[string]int,
	rangeStats *strategy.RangeStats,
) (*generated.StrategyConnection, error) {
	// Build relay connection
	conn, err := relay.NewConnection(items, totalCount, params, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	// Convert to GraphQL types
	edges := make([]*generated.StrategyEdge, len(conn.Edges))
	for i, edge := range conn.Edges {
		edges[i] = &generated.StrategyEdge{
			Node:   edge.Node,
			Cursor: edge.Cursor,
		}
	}

	// Build scopes with counts
	scopeDefs := getStrategyScopeDefinitions()
	scopes := make([]*generated.Scope, len(scopeDefs))
	for i, scopeDef := range scopeDefs {
		scopes[i] = &generated.Scope{
			ID:    scopeDef.ID,
			Name:  scopeDef.Name,
			Count: scopeCounts[scopeDef.ID],
		}
	}

	// Build filters from definitions
	filterDefs := getStrategyFilterDefinitions()
	filters := make([]*generated.Filter, len(filterDefs))
	for i, filterDef := range filterDefs {
		options := make([]*generated.FilterOption, len(filterDef.Options))
		for j, opt := range filterDef.Options {
			options[j] = &generated.FilterOption{
				Value: opt.Value,
				Label: opt.Label,
			}
		}

		// Apply min/max from range stats for NUMBER_RANGE filters
		var min, max *float64
		if rangeStats != nil && filterDef.Type == relay.FilterTypeNumberRange {
			switch filterDef.ID {
			case "capital_range":
				minVal, _ := rangeStats.MinCapital.Float64()
				maxVal, _ := rangeStats.MaxCapital.Float64()
				min = &minVal
				max = &maxVal
			case "pnl_range":
				minVal, _ := rangeStats.MinPnLPercent.Float64()
				maxVal, _ := rangeStats.MaxPnLPercent.Float64()
				min = &minVal
				max = &maxVal
			}
		}

		filters[i] = &generated.Filter{
			ID:           filterDef.ID,
			Name:         filterDef.Name,
			Type:         generated.FilterType(filterDef.Type),
			Options:      options,
			DefaultValue: filterDef.DefaultValue,
			Placeholder:  filterDef.Placeholder,
			Min:          min,
			Max:          max,
		}
	}

	return &generated.StrategyConnection{
		Edges: edges,
		PageInfo: &generated.PageInfo{
			HasNextPage:     conn.PageInfo.HasNextPage,
			HasPreviousPage: conn.PageInfo.HasPreviousPage,
			StartCursor:     conn.PageInfo.StartCursor,
			EndCursor:       conn.PageInfo.EndCursor,
		},
		TotalCount: totalCount,
		Scopes:     scopes,
		Filters:    filters,
	}, nil
}
