package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"prometheus/pkg/logger"
)

// Config holds generator configuration
type Config struct {
	TableName    string
	ResourceName string
	DryRun       bool
	BackendOnly  bool
	FrontendOnly bool
}

// Generator generates CRUD code from database schema
type Generator struct {
	log *logger.Logger
	db  *sql.DB
}

// NewGenerator creates a new generator instance
func NewGenerator(log *logger.Logger) *Generator {
	return &Generator{
		log: log,
	}
}

// Generate runs the full generation pipeline
func (g *Generator) Generate(ctx context.Context, config *Config) error {
	g.log.Infow("🔍 Analyzing table schema", "table", config.TableName)

	// Step 1: Analyze table schema
	schema, err := g.analyzeSchema(ctx, config.TableName)
	if err != nil {
		return fmt.Errorf("failed to analyze schema: %w", err)
	}

	g.log.Infow("Schema analyzed",
		"columns", len(schema.Columns),
		"enums", len(schema.Enums),
		"foreign_keys", len(schema.ForeignKeys),
	)

	// Step 2: Detect scopes
	scopes := g.detectScopes(schema)
	g.log.Infow("Detected scopes", "count", len(scopes), "scopes", scopes)

	// Step 3: Detect filters
	filters := g.detectFilters(schema)
	g.log.Infow("Detected filters", "count", len(filters))

	// Step 4: Generate backend (if not frontend-only)
	if !config.FrontendOnly {
		g.log.Infow("📝 Generating backend...")
		if err := g.generateBackend(ctx, config, schema, scopes, filters); err != nil {
			return fmt.Errorf("failed to generate backend: %w", err)
		}
	}

	// Step 5: Generate frontend (if not backend-only)
	if !config.BackendOnly {
		g.log.Infow("⚛️  Generating frontend...")
		if err := g.generateFrontend(ctx, config, schema, scopes, filters); err != nil {
			return fmt.Errorf("failed to generate frontend: %w", err)
		}
	}

	return nil
}

// TableSchema represents analyzed database table structure
type TableSchema struct {
	Name        string
	Columns     []Column
	Enums       []EnumType
	ForeignKeys []ForeignKey
	Indexes     []Index
}

// Column represents a table column
type Column struct {
	Name         string
	Type         string // PostgreSQL type
	GoType       string // Mapped Go type
	GraphQLType  string // Mapped GraphQL type
	TSType       string // Mapped TypeScript type
	Nullable     bool
	DefaultValue *string
	IsPrimaryKey bool
	IsForeignKey bool
	IsEnum       bool
	EnumValues   []string
}

// EnumType represents an enum in database
type EnumType struct {
	Name   string
	Values []string
}

// ForeignKey represents a foreign key constraint
type ForeignKey struct {
	Column           string
	ReferencedTable  string
	ReferencedColumn string
}

// Index represents a database index
type Index struct {
	Name    string
	Columns []string
	Unique  bool
}

// Scope represents a filter scope (tab)
type Scope struct {
	ID    string
	Name  string
	Where string // SQL WHERE clause
}

// Filter represents a dynamic filter
type Filter struct {
	ID          string
	Name        string
	Type        string // select, multiselect, number_range, date_range, text
	Column      string
	Options     []FilterOption // For select/multiselect
	Placeholder string
}

// FilterOption represents an option in select filter
type FilterOption struct {
	Value string
	Label string
}

// analyzeSchema queries PostgreSQL information_schema to get table structure
func (g *Generator) analyzeSchema(ctx context.Context, tableName string) (*TableSchema, error) {
	// TODO: Connect to database and query information_schema
	// For now, return mock data
	return &TableSchema{
		Name: tableName,
		Columns: []Column{
			{
				Name:         "id",
				Type:         "uuid",
				GoType:       "uuid.UUID",
				GraphQLType:  "UUID!",
				TSType:       "string",
				IsPrimaryKey: true,
			},
			{
				Name:        "name",
				Type:        "varchar",
				GoType:      "string",
				GraphQLType: "String!",
				TSType:      "string",
			},
			{
				Name:        "status",
				Type:        "varchar",
				GoType:      "string",
				GraphQLType: "String!",
				TSType:      "string",
				IsEnum:      true,
				EnumValues:  []string{"active", "paused", "closed"},
			},
		},
		Enums: []EnumType{
			{
				Name:   "status",
				Values: []string{"active", "paused", "closed"},
			},
		},
	}, nil
}

// detectScopes analyzes schema and suggests scopes
func (g *Generator) detectScopes(schema *TableSchema) []Scope {
	scopes := []Scope{
		{ID: "all", Name: "All", Where: ""},
	}

	// Detect from boolean columns (is_active, is_deleted, etc.)
	for _, col := range schema.Columns {
		if col.Type == "boolean" && strings.HasPrefix(col.Name, "is_") {
			stateName := strings.TrimPrefix(col.Name, "is_")
			scopes = append(scopes,
				Scope{
					ID:    stateName,
					Name:  toTitle(stateName),
					Where: fmt.Sprintf("%s = true", col.Name),
				},
				Scope{
					ID:    "not_" + stateName,
					Name:  "Not " + toTitle(stateName),
					Where: fmt.Sprintf("%s = false", col.Name),
				},
			)
		}
	}

	// Detect from enum columns
	for _, col := range schema.Columns {
		if col.IsEnum {
			for _, val := range col.EnumValues {
				scopes = append(scopes, Scope{
					ID:    val,
					Name:  toTitle(val),
					Where: fmt.Sprintf("%s = '%s'", col.Name, val),
				})
			}
		}
	}

	return scopes
}

// detectFilters analyzes schema and suggests filters
func (g *Generator) detectFilters(schema *TableSchema) []Filter {
	var filters []Filter

	for _, col := range schema.Columns {
		// Skip primary keys
		if col.IsPrimaryKey {
			continue
		}

		switch col.Type {
		case "boolean":
			filters = append(filters, Filter{
				ID:     col.Name,
				Name:   toTitle(col.Name),
				Type:   "select",
				Column: col.Name,
				Options: []FilterOption{
					{Value: "true", Label: "Yes"},
					{Value: "false", Label: "No"},
				},
			})

		case "varchar", "text":
			if col.IsEnum {
				// Multiselect for enums
				opts := make([]FilterOption, len(col.EnumValues))
				for i, val := range col.EnumValues {
					opts[i] = FilterOption{
						Value: val,
						Label: toTitle(val),
					}
				}
				filters = append(filters, Filter{
					ID:      col.Name,
					Name:    toTitle(col.Name),
					Type:    "multiselect",
					Column:  col.Name,
					Options: opts,
				})
			}

		case "integer", "bigint", "decimal", "numeric":
			filters = append(filters, Filter{
				ID:          col.Name + "_range",
				Name:        toTitle(col.Name) + " Range",
				Type:        "number_range",
				Column:      col.Name,
				Placeholder: "Enter min-max",
			})

		case "timestamp", "timestamptz", "date":
			filters = append(filters, Filter{
				ID:          col.Name + "_range",
				Name:        toTitle(col.Name),
				Type:        "date_range",
				Column:      col.Name,
				Placeholder: "Select date range",
			})
		}
	}

	return filters
}

// generateBackend generates all backend files
func (g *Generator) generateBackend(ctx context.Context, config *Config, schema *TableSchema, scopes []Scope, filters []Filter) error {
	// TODO: Generate files using templates
	g.log.Infow("   ✓ internal/domain/{resource}/entity.go")
	g.log.Infow("   ✓ internal/domain/{resource}/repository.go")
	g.log.Infow("   ✓ internal/repository/postgres/{resource}.go")
	g.log.Infow("   ✓ internal/services/{resource}/service.go")
	g.log.Infow("   ✓ internal/api/graphql/schema/{resource}.graphql")
	g.log.Infow("   ✓ internal/api/graphql/resolvers/{resource}.resolvers.go")
	g.log.Infow("   ✓ internal/api/graphql/resolvers/{resource}_scopes.go")
	g.log.Infow("   ✓ internal/api/graphql/resolvers/{resource}_filters.go")
	g.log.Infow("   ✓ internal/api/graphql/resolvers/{resource}_helpers.go")
	return nil
}

// generateFrontend generates all frontend files
func (g *Generator) generateFrontend(ctx context.Context, config *Config, schema *TableSchema, scopes []Scope, filters []Filter) error {
	// TODO: Generate files using templates
	g.log.Infow("   ✓ frontend/src/entities/{resource}/model/types.ts")
	g.log.Infow("   ✓ frontend/src/entities/{resource}/api/{resource}.graphql.ts")
	g.log.Infow("   ✓ frontend/src/entities/{resource}/lib/crud-config.tsx")
	g.log.Infow("   ✓ frontend/src/entities/{resource}/ui/{Resource}Manager.tsx")
	g.log.Infow("   ✓ frontend/src/app/(dashboard)/{resources}/page.tsx")
	g.log.Infow("   ✓ frontend/src/app/(dashboard)/{resources}/[id]/page.tsx")
	g.log.Infow("   ✓ frontend/src/app/(dashboard)/{resources}/[id]/edit/page.tsx")
	return nil
}

// toTitle converts snake_case to Title Case
func toTitle(s string) string {
	words := strings.Split(s, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}
