package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"prometheus/pkg/logger"
)

// TestAnalyzeTableFromDB_Users tests schema analysis on real users table
func TestAnalyzeTableFromDB_Users(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run")
	}

	// Load .env file
	_ = godotenv.Load("../../.env")

	ctx := context.Background()

	// Connect to test database
	db, err := connectDB()
	require.NoError(t, err, "Failed to connect to database")
	defer db.Close()

	// Test connection
	err = db.Ping()
	require.NoError(t, err, "Failed to ping database")

	// Analyze users table
	schema, err := analyzeTableFromDB(ctx, db, "users")
	require.NoError(t, err, "Failed to analyze users table")
	require.NotNil(t, schema, "Schema should not be nil")

	// Assertions
	t.Run("basic info", func(t *testing.T) {
		assert.Equal(t, "users", schema.Name)
		assert.NotEmpty(t, schema.Columns, "Should have columns")
	})

	t.Run("columns", func(t *testing.T) {
		// Should have at least these essential columns
		columnNames := make(map[string]bool)
		for _, col := range schema.Columns {
			columnNames[col.Name] = true
		}

		assert.True(t, columnNames["id"], "Should have id column")
		assert.True(t, columnNames["telegram_id"], "Should have telegram_id column")
		assert.True(t, columnNames["first_name"], "Should have first_name column")
		assert.True(t, columnNames["last_name"], "Should have last_name column")
		assert.True(t, columnNames["email"], "Should have email column")
		assert.True(t, columnNames["is_active"], "Should have is_active column")
		assert.True(t, columnNames["is_premium"], "Should have is_premium column")
		assert.True(t, columnNames["settings"], "Should have settings column")
		assert.True(t, columnNames["created_at"], "Should have created_at column")
		assert.True(t, columnNames["updated_at"], "Should have updated_at column")

		t.Logf("Found %d columns", len(schema.Columns))
		for _, col := range schema.Columns {
			t.Logf("  - %s: %s (Go: %s, GraphQL: %s, TS: %s, Nullable: %v, PK: %v, FK: %v)",
				col.Name, col.Type, col.GoType, col.GraphQLType, col.TSType,
				col.Nullable, col.IsPrimaryKey, col.IsForeignKey)
		}
	})

	t.Run("primary key", func(t *testing.T) {
		var idColumn *Column
		for i := range schema.Columns {
			if schema.Columns[i].Name == "id" {
				idColumn = &schema.Columns[i]
				break
			}
		}

		require.NotNil(t, idColumn, "id column should exist")
		assert.True(t, idColumn.IsPrimaryKey, "id should be primary key")
		assert.Equal(t, "uuid", idColumn.Type)
		assert.Equal(t, "uuid.UUID", idColumn.GoType)
		assert.Equal(t, "UUID!", idColumn.GraphQLType)
		assert.Equal(t, "string", idColumn.TSType)
	})

	t.Run("boolean columns", func(t *testing.T) {
		var isActiveCol *Column
		for i := range schema.Columns {
			if schema.Columns[i].Name == "is_active" {
				isActiveCol = &schema.Columns[i]
				break
			}
		}

		require.NotNil(t, isActiveCol, "is_active column should exist")
		assert.Contains(t, isActiveCol.Type, "bool", "is_active should be boolean type")
		// In real DB is_active is nullable, so Go type is *bool
		if isActiveCol.Nullable {
			assert.Equal(t, "*bool", isActiveCol.GoType)
			assert.Equal(t, "Boolean", isActiveCol.GraphQLType)
		} else {
			assert.Equal(t, "bool", isActiveCol.GoType)
			assert.Equal(t, "Boolean!", isActiveCol.GraphQLType)
		}
		assert.Equal(t, "boolean", isActiveCol.TSType)
	})

	t.Run("nullable columns", func(t *testing.T) {
		var emailCol *Column
		for i := range schema.Columns {
			if schema.Columns[i].Name == "email" {
				emailCol = &schema.Columns[i]
				break
			}
		}

		require.NotNil(t, emailCol, "email column should exist")
		assert.True(t, emailCol.Nullable, "email should be nullable")
		assert.Equal(t, "*string", emailCol.GoType, "Nullable string should be *string in Go")
	})

	t.Run("jsonb columns", func(t *testing.T) {
		var settingsCol *Column
		for i := range schema.Columns {
			if schema.Columns[i].Name == "settings" {
				settingsCol = &schema.Columns[i]
				break
			}
		}

		require.NotNil(t, settingsCol, "settings column should exist")
		assert.Contains(t, settingsCol.Type, "json", "settings should be jsonb type")
		assert.Equal(t, "map[string]any", settingsCol.GoType)
		assert.Equal(t, "JSONObject", settingsCol.GraphQLType)
	})

	t.Run("foreign keys", func(t *testing.T) {
		// Check if foreign keys were detected
		if len(schema.ForeignKeys) > 0 {
			t.Logf("Found %d foreign keys:", len(schema.ForeignKeys))
			for _, fk := range schema.ForeignKeys {
				t.Logf("  - %s → %s.%s", fk.Column, fk.ReferencedTable, fk.ReferencedColumn)
			}
		}

		// limit_profile_id should be a foreign key
		var hasLimitProfileFK bool
		for _, fk := range schema.ForeignKeys {
			if fk.Column == "limit_profile_id" {
				hasLimitProfileFK = true
				assert.Equal(t, "limit_profiles", fk.ReferencedTable)
				assert.Equal(t, "id", fk.ReferencedColumn)
			}
		}
		if !hasLimitProfileFK {
			t.Log("Warning: limit_profile_id foreign key not detected")
		}
	})

	t.Run("indexes", func(t *testing.T) {
		// Check if indexes were detected
		if len(schema.Indexes) > 0 {
			t.Logf("Found %d indexes:", len(schema.Indexes))
			for _, idx := range schema.Indexes {
				t.Logf("  - %s: %v (unique: %v)", idx.Name, idx.Columns, idx.Unique)
			}
		}
	})
}

// TestDetectScopes_Users tests scope detection for users table
func TestDetectScopes_Users(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run")
	}

	_ = godotenv.Load("../../.env")
	ctx := context.Background()
	zapLogger, _ := zap.NewDevelopment()
	log := &logger.Logger{SugaredLogger: zapLogger.Sugar()}

	gen := NewGenerator(log)

	// Connect to DB
	db, err := connectDB()
	require.NoError(t, err)
	defer db.Close()
	gen.db = db

	// Analyze schema
	schema, err := analyzeTableFromDB(ctx, db, "users")
	require.NoError(t, err)

	// Detect scopes
	scopes := gen.detectScopes(schema)

	t.Run("has all scope", func(t *testing.T) {
		var hasAll bool
		for _, scope := range scopes {
			if scope.ID == "all" {
				hasAll = true
				assert.Equal(t, "All", scope.Name)
				assert.Empty(t, scope.Where)
			}
		}
		assert.True(t, hasAll, "Should have 'all' scope")
	})

	t.Run("has active/inactive scopes", func(t *testing.T) {
		scopeIDs := make(map[string]bool)
		for _, scope := range scopes {
			scopeIDs[scope.ID] = true
		}

		assert.True(t, scopeIDs["active"], "Should have 'active' scope from is_active column")
		assert.True(t, scopeIDs["not_active"], "Should have 'not_active' scope")
	})

	t.Run("has premium/free scopes", func(t *testing.T) {
		scopeIDs := make(map[string]bool)
		for _, scope := range scopes {
			scopeIDs[scope.ID] = true
		}

		assert.True(t, scopeIDs["premium"], "Should have 'premium' scope from is_premium column")
		assert.True(t, scopeIDs["not_premium"], "Should have 'not_premium' scope")
	})

	t.Logf("Detected %d scopes:", len(scopes))
	for _, scope := range scopes {
		t.Logf("  - %s: %s (WHERE: %s)", scope.ID, scope.Name, scope.Where)
	}
}

// TestDetectFilters_Users tests filter detection for users table
func TestDetectFilters_Users(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run")
	}

	_ = godotenv.Load("../../.env")
	ctx := context.Background()
	zapLogger, _ := zap.NewDevelopment()
	log := &logger.Logger{SugaredLogger: zapLogger.Sugar()}

	gen := NewGenerator(log)

	// Connect to DB
	db, err := connectDB()
	require.NoError(t, err)
	defer db.Close()
	gen.db = db

	// Analyze schema
	schema, err := analyzeTableFromDB(ctx, db, "users")
	require.NoError(t, err)

	// Detect filters
	filters := gen.detectFilters(schema)

	t.Run("has filters", func(t *testing.T) {
		assert.NotEmpty(t, filters, "Should detect some filters")
	})

	t.Run("has boolean filters", func(t *testing.T) {
		filterIDs := make(map[string]bool)
		for _, filter := range filters {
			filterIDs[filter.ID] = true
		}

		assert.True(t, filterIDs["is_active"], "Should have is_active filter")
		assert.True(t, filterIDs["is_premium"], "Should have is_premium filter")
	})

	t.Run("has date range filters", func(t *testing.T) {
		filterIDs := make(map[string]bool)
		for _, filter := range filters {
			filterIDs[filter.ID] = true
		}

		assert.True(t, filterIDs["created_at_range"], "Should have created_at_range filter")
		assert.True(t, filterIDs["updated_at_range"], "Should have updated_at_range filter")
	})

	t.Logf("Detected %d filters:", len(filters))
	for _, filter := range filters {
		t.Logf("  - %s: %s (type: %s, column: %s)", filter.ID, filter.Name, filter.Type, filter.Column)
		if len(filter.Options) > 0 {
			t.Logf("    Options: %v", filter.Options)
		}
	}
}

// TestFullGeneration_DryRun tests the complete generation pipeline in dry-run mode
func TestFullGeneration_DryRun(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run")
	}

	_ = godotenv.Load("../../.env")
	ctx := context.Background()
	zapLogger, _ := zap.NewDevelopment()
	log := &logger.Logger{SugaredLogger: zapLogger.Sugar()}

	gen := NewGenerator(log)

	config := &Config{
		TableName:   "users",
		DryRun:      true, // Don't create files
		BackendOnly: false,
	}

	err := gen.Generate(ctx, config)
	assert.NoError(t, err, "Generation should complete without errors")
}

// TestAnalyzeTableFromDB_Strategies tests schema analysis on user_strategies table
func TestAnalyzeTableFromDB_Strategies(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run")
	}

	_ = godotenv.Load("../../.env")
	ctx := context.Background()

	db, err := connectDB()
	require.NoError(t, err)
	defer db.Close()

	schema, err := analyzeTableFromDB(ctx, db, "user_strategies")
	require.NoError(t, err)

	t.Run("basic info", func(t *testing.T) {
		assert.Equal(t, "user_strategies", schema.Name)
		assert.NotEmpty(t, schema.Columns)
	})

	t.Run("has enums", func(t *testing.T) {
		if len(schema.Enums) > 0 {
			t.Logf("Found %d enums:", len(schema.Enums))
			for _, enum := range schema.Enums {
				t.Logf("  - %s: %v", enum.Name, enum.Values)
			}

			// Should detect strategy_status enum
			var hasStatusEnum bool
			for _, enum := range schema.Enums {
				if enum.Name == "strategy_status" || enum.Name == "status" {
					hasStatusEnum = true
					assert.Contains(t, enum.Values, "active")
					assert.Contains(t, enum.Values, "paused")
					assert.Contains(t, enum.Values, "closed")
				}
			}
			assert.True(t, hasStatusEnum, "Should detect status enum")
		}
	})

	t.Run("has foreign keys", func(t *testing.T) {
		// Should have user_id foreign key
		var hasUserFK bool
		for _, fk := range schema.ForeignKeys {
			t.Logf("FK: %s → %s.%s", fk.Column, fk.ReferencedTable, fk.ReferencedColumn)
			if fk.Column == "user_id" {
				hasUserFK = true
				assert.Equal(t, "users", fk.ReferencedTable)
			}
		}
		assert.True(t, hasUserFK, "Should have user_id foreign key")
	})

	t.Run("column types", func(t *testing.T) {
		for _, col := range schema.Columns {
			// Validate type mappings are not empty
			assert.NotEmpty(t, col.GoType, "Column %s should have Go type", col.Name)
			assert.NotEmpty(t, col.GraphQLType, "Column %s should have GraphQL type", col.Name)
			assert.NotEmpty(t, col.TSType, "Column %s should have TypeScript type", col.Name)

			// Specific validations
			switch col.Name {
			case "id":
				assert.Equal(t, "uuid.UUID", col.GoType)
				assert.Equal(t, "UUID!", col.GraphQLType)
			case "allocated_capital", "current_equity":
				assert.Equal(t, "decimal.Decimal", col.GoType)
				assert.Contains(t, col.GraphQLType, "Decimal")
			case "is_active":
				assert.Equal(t, "bool", col.GoType)
				assert.Equal(t, "Boolean!", col.GraphQLType)
			}
		}
	})
}

// TestDetectScopes_Strategies tests scope detection for strategies
func TestDetectScopes_Strategies(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run")
	}

	_ = godotenv.Load("../../.env")
	ctx := context.Background()
	zapLogger, _ := zap.NewDevelopment()
	log := &logger.Logger{SugaredLogger: zapLogger.Sugar()}

	gen := NewGenerator(log)
	db, err := connectDB()
	require.NoError(t, err)
	defer db.Close()
	gen.db = db

	schema, err := analyzeTableFromDB(ctx, db, "user_strategies")
	require.NoError(t, err)

	scopes := gen.detectScopes(schema)

	t.Run("has status scopes from enum", func(t *testing.T) {
		scopeIDs := make(map[string]bool)
		for _, scope := range scopes {
			scopeIDs[scope.ID] = true
		}

		// Should have scopes for each status enum value
		assert.True(t, scopeIDs["active"] || scopeIDs["ACTIVE"], "Should have active scope")
		assert.True(t, scopeIDs["paused"] || scopeIDs["PAUSED"], "Should have paused scope")
		assert.True(t, scopeIDs["closed"] || scopeIDs["CLOSED"], "Should have closed scope")
	})

	t.Logf("Detected %d scopes for strategies:", len(scopes))
	for _, scope := range scopes {
		t.Logf("  - %s: %s", scope.ID, scope.Name)
	}
}

// TestGenerateFormFields tests form fields generation from schema
func TestGenerateFormFields(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run")
	}

	_ = godotenv.Load("../../.env")
	ctx := context.Background()
	zapLogger, _ := zap.NewDevelopment()
	log := &logger.Logger{SugaredLogger: zapLogger.Sugar()}

	gen := NewGenerator(log)
	db, err := connectDB()
	require.NoError(t, err)
	defer db.Close()
	gen.db = db

	schema, err := analyzeTableFromDB(ctx, db, "user_strategies")
	require.NoError(t, err)

	formFields := gen.generateFormFields(schema)

	t.Run("has form fields", func(t *testing.T) {
		assert.NotEmpty(t, formFields, "Should generate form fields")
		t.Logf("Generated %d form fields", len(formFields))
		for _, field := range formFields {
			t.Logf("  - %s (%s): validation=%s, colSpan=%d",
				field.Name, field.Type, field.Validation, field.ColSpan)
		}
	})

	t.Run("skips auto-generated fields", func(t *testing.T) {
		for _, field := range formFields {
			assert.NotEqual(t, "id", field.Name)
			assert.NotEqual(t, "createdAt", field.Name)
			assert.NotEqual(t, "updatedAt", field.Name)
		}
	})

	t.Run("maps types correctly", func(t *testing.T) {
		fieldsByName := make(map[string]FormField)
		for _, field := range formFields {
			fieldsByName[field.Name] = field
		}

		// String field
		if field, ok := fieldsByName["name"]; ok {
			assert.Equal(t, "text", field.Type)
			assert.NotEmpty(t, field.Validation)
		}

		// Enum field (if has status)
		if field, ok := fieldsByName["status"]; ok {
			assert.Equal(t, "select", field.Type)
			assert.NotEmpty(t, field.Options)
		}

		// Foreign key (if has userId)
		if field, ok := fieldsByName["userId"]; ok {
			assert.Equal(t, "custom", field.Type)
			assert.Contains(t, field.Render, "SelectField")
		}
	})
}

// TestGenerateDisplayColumns tests display columns generation
func TestGenerateDisplayColumns(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run")
	}

	_ = godotenv.Load("../../.env")
	ctx := context.Background()
	zapLogger, _ := zap.NewDevelopment()
	log := &logger.Logger{SugaredLogger: zapLogger.Sugar()}

	gen := NewGenerator(log)
	db, err := connectDB()
	require.NoError(t, err)
	defer db.Close()
	gen.db = db

	schema, err := analyzeTableFromDB(ctx, db, "user_strategies")
	require.NoError(t, err)

	columns := gen.generateDisplayColumns(schema)

	t.Run("has display columns", func(t *testing.T) {
		assert.NotEmpty(t, columns, "Should generate display columns")
		t.Logf("Generated %d display columns", len(columns))
		for _, col := range columns {
			t.Logf("  - %s (%s): sortable=%v, width=%s",
				col.Key, col.Label, col.Sortable, col.Width)
		}
	})

	t.Run("has actions column", func(t *testing.T) {
		var hasActions bool
		for _, col := range columns {
			if col.Key == "actions" {
				hasActions = true
				assert.Equal(t, "", col.Label)
			}
		}
		assert.True(t, hasActions, "Should have actions column")
	})

	t.Run("limits number of columns", func(t *testing.T) {
		// Should not have too many columns (max ~7)
		assert.LessOrEqual(t, len(columns), 7, "Should limit number of columns")
	})
}

// TestTypeMappings tests PostgreSQL to Go/GraphQL/TS type conversions
func TestTypeMappings(t *testing.T) {
	tests := []struct {
		pgType      string
		nullable    bool
		wantGo      string
		wantGraphQL string
		wantTS      string
	}{
		{"uuid", false, "uuid.UUID", "UUID!", "string"},
		{"varchar", false, "string", "String!", "string"},
		{"varchar", true, "*string", "String", "string"},
		{"integer", false, "int", "Int!", "number"},
		{"integer", true, "*int", "Int", "number"},
		{"boolean", false, "bool", "Boolean!", "boolean"},
		{"boolean", true, "*bool", "Boolean", "boolean"},
		{"timestamp", false, "time.Time", "Time!", "string"},
		{"decimal", false, "decimal.Decimal", "Float!", "number"},
		{"jsonb", false, "map[string]any", "JSONObject!", "Record<string, any>"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_nullable_%v", tt.pgType, tt.nullable), func(t *testing.T) {
			gotGo := mapPGTypeToGo(tt.pgType, tt.nullable)
			gotGraphQL := mapPGTypeToGraphQL(tt.pgType, tt.nullable)
			gotTS := mapPGTypeToTS(tt.pgType, "test_column")

			assert.Equal(t, tt.wantGo, gotGo, "Go type mismatch")
			assert.Equal(t, tt.wantGraphQL, gotGraphQL, "GraphQL type mismatch")
			assert.Equal(t, tt.wantTS, gotTS, "TypeScript type mismatch")
		})
	}
}
