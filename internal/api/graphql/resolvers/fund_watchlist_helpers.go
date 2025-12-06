package resolvers

import (
	"fmt"
	"prometheus/internal/api/graphql/generated"
	"prometheus/internal/domain/fundwatchlist"
	"prometheus/pkg/relay"
)

// buildFundWatchlistConnection is a helper function to build GraphQL connection with scopes and filters
// This avoids code duplication between FundWatchlistsConnection and MonitoredSymbolsConnection resolvers
func buildFundWatchlistConnection(
	items []*fundwatchlist.Watchlist,
	totalCount int,
	params relay.PaginationParams,
	offset int,
	scopeCounts map[string]int,
) (*generated.FundWatchlistConnection, error) {
	// Build relay connection
	conn, err := relay.NewConnection(items, totalCount, params, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	// Convert to GraphQL types
	edges := make([]*generated.FundWatchlistEdge, len(conn.Edges))
	for i, edge := range conn.Edges {
		edges[i] = &generated.FundWatchlistEdge{
			Node:   edge.Node,
			Cursor: edge.Cursor,
		}
	}

	// Build scopes with counts
	scopeDefs := getFundWatchlistScopeDefinitions()
	scopes := make([]*generated.Scope, len(scopeDefs))
	for i, scopeDef := range scopeDefs {
		scopes[i] = &generated.Scope{
			ID:    scopeDef.ID,
			Name:  scopeDef.Name,
			Count: scopeCounts[scopeDef.ID],
		}
	}

	// Build filters from definitions
	filterDefs := getFundWatchlistFilterDefinitions()
	filters := make([]*generated.Filter, 0, len(filterDefs))

	for _, filterDef := range filterDefs {
		options := make([]*generated.FilterOption, len(filterDef.Options))
		for j, opt := range filterDef.Options {
			options[j] = &generated.FilterOption{
				Value: opt.Value,
				Label: opt.Label,
			}
		}

		filters = append(filters, &generated.Filter{
			ID:          filterDef.ID,
			Name:        filterDef.Name,
			Type:        generated.FilterType(filterDef.Type),
			Options:     options,
			Placeholder: filterDef.Placeholder,
		})
	}

	return &generated.FundWatchlistConnection{
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
