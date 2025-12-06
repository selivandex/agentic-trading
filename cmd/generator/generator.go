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

	// Connect to database
	db, err := connectDB()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()
	g.db = db

	// Step 1: Analyze table schema from real database
	schema, err := analyzeTableFromDB(ctx, db, config.TableName)
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

// analyzeSchema is deprecated - use analyzeTableFromDB from db.go instead
func (g *Generator) analyzeSchema(ctx context.Context, tableName string) (*TableSchema, error) {
	return analyzeTableFromDB(ctx, g.db, tableName)
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

		case "integer", "bigint", "decimal", "numeric", "smallint", "real", "double precision":
			filters = append(filters, Filter{
				ID:          col.Name + "_range",
				Name:        toTitle(col.Name) + " Range",
				Type:        "number_range",
				Column:      col.Name,
				Placeholder: "Enter min-max",
			})
		}

		// Check for timestamp types (various formats)
		if strings.Contains(col.Type, "timestamp") || col.Type == "date" {
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
	// Determine resource name
	resourceName := config.ResourceName
	if resourceName == "" {
		resourceName = toPascalCase(singularize(schema.Name))
	}
	resourceNameSnake := toSnakeCase(resourceName)

	// Check if ID is UUID
	isUUID := false
	for _, col := range schema.Columns {
		if col.Name == "id" && col.Type == "uuid" {
			isUUID = true
			break
		}
	}

	templateData := map[string]interface{}{
		"TableName":    schema.Name,
		"ResourceName": resourceName,
		"Columns":      schema.Columns,
		"Scopes":       scopes,
		"Filters":      filters,
		"IsUUID":       isUUID,
	}

	files := []struct {
		path     string
		template string
	}{
		{
			path:     fmt.Sprintf("internal/api/graphql/schema/%s.graphql", resourceNameSnake),
			template: backendGraphQLSchemaTemplate,
		},
		{
			path:     fmt.Sprintf("internal/api/graphql/resolvers/%s_scopes.go", resourceNameSnake),
			template: backendScopesTemplate,
		},
		{
			path:     fmt.Sprintf("internal/api/graphql/resolvers/%s_filters.go", resourceNameSnake),
			template: backendFiltersTemplate,
		},
		{
			path:     fmt.Sprintf("internal/api/graphql/resolvers/%s_helpers.go", resourceNameSnake),
			template: backendHelpersTemplate,
		},
	}

	for _, file := range files {
		if err := generateFile(file.path, file.template, templateData, config.DryRun); err != nil {
			return fmt.Errorf("failed to generate %s: %w", file.path, err)
		}
		g.log.Infow("   ✓ " + file.path)
	}

	return nil
}

// generateFrontend generates all frontend files
func (g *Generator) generateFrontend(ctx context.Context, config *Config, schema *TableSchema, scopes []Scope, filters []Filter) error {
	// Determine resource name
	resourceName := config.ResourceName
	if resourceName == "" {
		resourceName = toPascalCase(singularize(schema.Name))
	}
	resourceNamePlural := pluralize(resourceName)
	resourceNameKebab := toKebabCase(resourceName)
	resourceNamePluralKebab := toKebabCase(resourceNamePlural)

	templateData := map[string]interface{}{
		"TableName":               schema.Name,
		"ResourceName":            resourceName,
		"ResourceNamePlural":      resourceNamePlural,
		"ResourceNameKebab":       resourceNameKebab,
		"ResourceNamePluralKebab": resourceNamePluralKebab,
		"Columns":                 schema.Columns,
		"Scopes":                  scopes,
		"Filters":                 filters,
	}

	files := []struct {
		path     string
		template string
	}{
		// Model
		{
			path:     fmt.Sprintf("frontend/src/entities/%s/model/types.ts", resourceNameKebab),
			template: frontendTypesTemplate,
		},
		{
			path:     fmt.Sprintf("frontend/src/entities/%s/model/index.ts", resourceNameKebab),
			template: frontendIndexTemplate,
		},
		// API
		{
			path:     fmt.Sprintf("frontend/src/entities/%s/api/%s.graphql.ts", resourceNameKebab, resourceNameKebab),
			template: frontendGraphQLTemplate,
		},
		{
			path:     fmt.Sprintf("frontend/src/entities/%s/api/index.ts", resourceNameKebab),
			template: frontendIndexTemplate,
		},
		// Lib
		{
			path:     fmt.Sprintf("frontend/src/entities/%s/lib/crud-config.tsx", resourceNameKebab),
			template: frontendCrudConfigTemplate,
		},
		{
			path:     fmt.Sprintf("frontend/src/entities/%s/lib/index.ts", resourceNameKebab),
			template: frontendIndexTemplate,
		},
		// UI
		{
			path:     fmt.Sprintf("frontend/src/entities/%s/ui/%sManager.tsx", resourceNameKebab, resourceName),
			template: frontendManagerTemplate,
		},
		{
			path:     fmt.Sprintf("frontend/src/entities/%s/ui/index.ts", resourceNameKebab),
			template: frontendIndexTemplate,
		},
		// Main index
		{
			path:     fmt.Sprintf("frontend/src/entities/%s/index.ts", resourceNameKebab),
			template: frontendIndexTemplate,
		},
		// Pages
		{
			path:     fmt.Sprintf("frontend/src/app/(dashboard)/%s/page.tsx", resourceNamePluralKebab),
			template: frontendListPageTemplate,
		},
		{
			path:     fmt.Sprintf("frontend/src/app/(dashboard)/%s/[id]/page.tsx", resourceNamePluralKebab),
			template: frontendShowPageTemplate,
		},
		{
			path:     fmt.Sprintf("frontend/src/app/(dashboard)/%s/[id]/edit/page.tsx", resourceNamePluralKebab),
			template: frontendEditPageTemplate,
		},
		{
			path:     fmt.Sprintf("frontend/src/app/(dashboard)/%s/new/page.tsx", resourceNamePluralKebab),
			template: frontendNewPageTemplate,
		},
	}

	// Prepare index.ts exports data
	indexExportsData := map[string]interface{}{
		"Exports": []string{"types"},
	}
	apiIndexData := map[string]interface{}{
		"Exports": []string{resourceNameKebab + ".graphql"},
	}
	libIndexData := map[string]interface{}{
		"Exports": []string{"crud-config"},
	}
	uiIndexData := map[string]interface{}{
		"Exports": []string{resourceName + "Manager"},
	}
	mainIndexData := map[string]interface{}{
		"Exports": []string{"api", "lib", "model", "ui"},
	}

	for _, file := range files {
		// Use appropriate data for index files
		data := templateData
		if strings.HasSuffix(file.path, "/model/index.ts") {
			data = indexExportsData
		} else if strings.HasSuffix(file.path, "/api/index.ts") {
			data = apiIndexData
		} else if strings.HasSuffix(file.path, "/lib/index.ts") {
			data = libIndexData
		} else if strings.HasSuffix(file.path, "/ui/index.ts") {
			data = uiIndexData
		} else if strings.HasSuffix(file.path, fmt.Sprintf("entities/%s/index.ts", resourceNameKebab)) {
			data = mainIndexData
		}

		if err := generateFile(file.path, file.template, data, config.DryRun); err != nil {
			return fmt.Errorf("failed to generate %s: %w", file.path, err)
		}
		g.log.Infow("   ✓ " + file.path)
	}

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
