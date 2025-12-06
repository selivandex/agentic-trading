package main

const backendGraphQLSchemaTemplate = `# @format

# {{.ResourceName}} types and operations
# Auto-generated from table: {{.TableName}}

type {{.ResourceName}} {
{{range .Columns}}  {{toCamel .Name}}: {{.GraphQLType}}
{{end}}}

"""
Edge type for {{.ResourceName}}
"""
type {{.ResourceName}}Edge {
  """
  The item at the end of the edge
  """
  node: {{.ResourceName}}!

  """
  A cursor for use in pagination
  """
  cursor: String!
}

"""
Connection type for {{.ResourceName}} collection
"""
type {{.ResourceName}}Connection {
  """
  A list of edges
  """
  edges: [{{.ResourceName}}Edge!]!

  """
  Information to aid in pagination
  """
  pageInfo: PageInfo!

  """
  Total count of items (if available)
  """
  totalCount: Int!

  """
  Available filter scopes with counts
  """
  scopes: [Scope!]!

  """
  Available dynamic filters
  """
  filters: [Filter!]!
}

input Create{{.ResourceName}}Input {
{{range .Columns -}}
{{if not .IsPrimaryKey -}}
{{if not (or (eq .Name "created_at") (eq .Name "updated_at")) -}}
  {{toCamel .Name}}: {{replace .GraphQLType "!" ""}}
{{end -}}
{{end -}}
{{end}}}

input Update{{.ResourceName}}Input {
{{range .Columns -}}
{{if not .IsPrimaryKey -}}
{{if not (or (eq .Name "created_at") (eq .Name "updated_at")) -}}
  {{toCamel .Name}}: {{replace .GraphQLType "!" ""}}
{{end -}}
{{end -}}
{{end}}}

# Queries
extend type Query {
  # Get {{.ResourceName}} by ID
  {{toCamel .ResourceName}}(id: {{if .IsUUID}}UUID{{else}}Int{{end}}!): {{.ResourceName}}

  # Get all {{pluralize .ResourceName}} with scopes and filters
  {{toCamel (pluralize .ResourceName)}}(
    scope: String
    search: String
    filters: JSONObject
    first: Int
    after: String
    last: Int
    before: String
  ): {{.ResourceName}}Connection!
}

# Mutations
extend type Mutation {
  # Create new {{.ResourceName}}
  create{{.ResourceName}}(input: Create{{.ResourceName}}Input!): {{.ResourceName}}!

  # Update {{.ResourceName}}
  update{{.ResourceName}}(id: {{if .IsUUID}}UUID{{else}}Int{{end}}!, input: Update{{.ResourceName}}Input!): {{.ResourceName}}!

  # Delete {{.ResourceName}}
  delete{{.ResourceName}}(id: {{if .IsUUID}}UUID{{else}}Int{{end}}!): Boolean!
}
`

const backendScopesTemplate = `package resolvers

import (
	"prometheus/pkg/relay"
)

// get{{.ResourceName}}ScopeDefinitions returns scope definitions for {{toLower (pluralize .ResourceName)}}
// These are used ONLY for metadata (name, id) in GraphQL responses
// Filtering is done at repository level via SQL WHERE clauses
func get{{.ResourceName}}ScopeDefinitions() []relay.Scope {
	return []relay.Scope{
{{range .Scopes}}		{
			ID:    "{{.ID}}",
			Name:  "{{.Name}}",
			Count: 0, // Will be populated from service
		},
{{end}}	}
}
`

const backendFiltersTemplate = `package resolvers

import (
	"prometheus/pkg/relay"
)

// get{{.ResourceName}}FilterDefinitions returns filter definitions for {{toLower (pluralize .ResourceName)}}
// These define the available filters that can be applied to {{toLower .ResourceName}} lists
func get{{.ResourceName}}FilterDefinitions() []relay.FilterDefinition {
	return []relay.FilterDefinition{
{{range .Filters}}		{
			ID:   "{{.ID}}",
			Name: "{{.Name}}",
			Type: relay.FilterType{{toPascal .Type}},
{{if .Options}}			Options: []relay.FilterOption{
{{range .Options}}				{Value: "{{.Value}}", Label: "{{.Label}}"},
{{end}}			},
{{end}}{{if .Placeholder}}			Placeholder: strPtr("{{.Placeholder}}"),
{{end}}		},
{{end}}	}
}

// strPtr is a helper to create string pointers
func strPtr(s string) *string {
	return &s
}
`

const backendHelpersTemplate = `package resolvers

import (
	"fmt"

	"prometheus/internal/api/graphql/generated"
	"prometheus/internal/domain/{{toSnake .ResourceName}}"
	"prometheus/pkg/relay"
)

// build{{.ResourceName}}Connection is a helper function to build GraphQL connection with scopes and filters
func build{{.ResourceName}}Connection(
	items []*{{toSnake .ResourceName}}.{{.ResourceName}},
	totalCount int,
	params relay.PaginationParams,
	offset int,
	scopeCounts map[string]int,
) (*generated.{{.ResourceName}}Connection, error) {
	// Build relay connection
	conn, err := relay.NewConnection(items, totalCount, params, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	// Convert to GraphQL types
	edges := make([]*generated.{{.ResourceName}}Edge, len(conn.Edges))
	for i, edge := range conn.Edges {
		edges[i] = &generated.{{.ResourceName}}Edge{
			Node:   edge.Node,
			Cursor: edge.Cursor,
		}
	}

	// Build scopes with counts
	scopeDefs := get{{.ResourceName}}ScopeDefinitions()
	scopes := make([]*generated.Scope, len(scopeDefs))
	for i, scopeDef := range scopeDefs {
		scopes[i] = &generated.Scope{
			ID:    scopeDef.ID,
			Name:  scopeDef.Name,
			Count: scopeCounts[scopeDef.ID],
		}
	}

	// Build filters from definitions
	filterDefs := get{{.ResourceName}}FilterDefinitions()
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
			ID:           filterDef.ID,
			Name:         filterDef.Name,
			Type:         generated.FilterType(filterDef.Type),
			Options:      options,
			DefaultValue: filterDef.DefaultValue,
			Placeholder:  filterDef.Placeholder,
		})
	}

	return &generated.{{.ResourceName}}Connection{
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
`
