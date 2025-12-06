package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

// connectDB establishes connection to PostgreSQL
func connectDB() (*sql.DB, error) {
	// Read from environment or use defaults
	host := getEnv("POSTGRES_HOST", "localhost")
	port := getEnv("POSTGRES_PORT", "5432")
	user := getEnv("POSTGRES_USER", "postgres")
	password := getEnv("POSTGRES_PASSWORD", "postgres")
	dbname := getEnv("POSTGRES_DB", "prometheus")
	sslmode := getEnv("POSTGRES_SSL_MODE", "disable")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// analyzeTableFromDB queries PostgreSQL information_schema to get real table structure
func analyzeTableFromDB(ctx context.Context, db *sql.DB, tableName string) (*TableSchema, error) {
	schema := &TableSchema{
		Name:        tableName,
		Columns:     []Column{},
		Enums:       []EnumType{},
		ForeignKeys: []ForeignKey{},
		Indexes:     []Index{},
	}

	// Query columns
	columnsQuery := `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable,
			c.column_default,
			c.udt_name,
			CASE
				WHEN pk.column_name IS NOT NULL THEN true
				ELSE false
			END as is_primary_key,
			CASE
				WHEN fk.column_name IS NOT NULL THEN true
				ELSE false
			END as is_foreign_key
		FROM information_schema.columns c
		LEFT JOIN (
			SELECT ku.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage ku
				ON tc.constraint_name = ku.constraint_name
			WHERE tc.table_name = $1
				AND tc.constraint_type = 'PRIMARY KEY'
		) pk ON c.column_name = pk.column_name
		LEFT JOIN (
			SELECT ku.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage ku
				ON tc.constraint_name = ku.constraint_name
			WHERE tc.table_name = $1
				AND tc.constraint_type = 'FOREIGN KEY'
		) fk ON c.column_name = fk.column_name
		WHERE c.table_name = $1
		ORDER BY c.ordinal_position
	`

	rows, err := db.QueryContext(ctx, columnsQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var col Column
		var dataType, udtName string
		var isNullable string
		var defaultVal sql.NullString

		err := rows.Scan(
			&col.Name,
			&dataType,
			&isNullable,
			&defaultVal,
			&udtName,
			&col.IsPrimaryKey,
			&col.IsForeignKey,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		col.Type = dataType
		col.Nullable = isNullable == "YES"
		if defaultVal.Valid {
			col.DefaultValue = &defaultVal.String
		}

		// Map PostgreSQL types to Go/GraphQL/TypeScript
		col.GoType = mapPGTypeToGo(dataType, col.Nullable)
		col.GraphQLType = mapPGTypeToGraphQL(dataType, col.Nullable)
		col.TSType = mapPGTypeToTS(dataType, col.Name)

		// Check if it's an enum
		if dataType == "USER-DEFINED" {
			col.IsEnum = true
			col.Type = udtName

			// Get enum values
			enumVals, err := getEnumValues(ctx, db, udtName)
			if err == nil {
				col.EnumValues = enumVals
				// Add to schema enums if not already there
				found := false
				for _, e := range schema.Enums {
					if e.Name == udtName {
						found = true
						break
					}
				}
				if !found {
					schema.Enums = append(schema.Enums, EnumType{
						Name:   udtName,
						Values: enumVals,
					})
				}
			}
		}

		schema.Columns = append(schema.Columns, col)
	}

	// Query foreign keys
	fkQuery := `
		SELECT
			kcu.column_name,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_name = $1
	`

	fkRows, err := db.QueryContext(ctx, fkQuery, tableName)
	if err == nil {
		defer fkRows.Close()
		for fkRows.Next() {
			var fk ForeignKey
			fkRows.Scan(&fk.Column, &fk.ReferencedTable, &fk.ReferencedColumn)
			schema.ForeignKeys = append(schema.ForeignKeys, fk)
		}
	}

	// Query indexes
	idxQuery := `
		SELECT
			i.relname as index_name,
			array_agg(a.attname ORDER BY a.attnum) as column_names,
			ix.indisunique as is_unique
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE t.relname = $1
			AND t.relkind = 'r'
		GROUP BY i.relname, ix.indisunique
	`

	idxRows, err := db.QueryContext(ctx, idxQuery, tableName)
	if err == nil {
		defer idxRows.Close()
		for idxRows.Next() {
			var idx Index
			var cols string
			idxRows.Scan(&idx.Name, &cols, &idx.Unique)
			// Parse PostgreSQL array format: {col1,col2}
			cols = strings.Trim(cols, "{}")
			idx.Columns = strings.Split(cols, ",")
			schema.Indexes = append(schema.Indexes, idx)
		}
	}

	return schema, nil
}

// getEnumValues retrieves enum values from PostgreSQL
func getEnumValues(ctx context.Context, db *sql.DB, enumName string) ([]string, error) {
	query := `
		SELECT e.enumlabel
		FROM pg_type t
		JOIN pg_enum e ON t.oid = e.enumtypid
		WHERE t.typname = $1
		ORDER BY e.enumsortorder
	`

	rows, err := db.QueryContext(ctx, query, enumName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var values []string
	for rows.Next() {
		var val string
		rows.Scan(&val)
		values = append(values, val)
	}

	return values, nil
}

// Type mapping functions
func mapPGTypeToGo(pgType string, nullable bool) string {
	var baseType string
	switch pgType {
	case "uuid":
		baseType = "uuid.UUID"
	case "varchar", "text", "character varying":
		baseType = "string"
	case "integer", "int", "int4":
		baseType = "int"
	case "bigint", "int8":
		baseType = "int64"
	case "smallint", "int2":
		baseType = "int16"
	case "decimal", "numeric":
		baseType = "decimal.Decimal"
	case "boolean", "bool":
		baseType = "bool"
	case "timestamp", "timestamptz", "timestamp without time zone", "timestamp with time zone":
		baseType = "time.Time"
	case "date":
		baseType = "time.Time"
	case "jsonb", "json":
		baseType = "map[string]any"
	case "bytea":
		baseType = "[]byte"
	case "real", "float4":
		baseType = "float32"
	case "double precision", "float8":
		baseType = "float64"
	default:
		baseType = "string"
	}

	if nullable && baseType != "uuid.UUID" && baseType != "time.Time" && !strings.Contains(baseType, "[]") && !strings.Contains(baseType, "map[") {
		return "*" + baseType
	}

	return baseType
}

func mapPGTypeToGraphQL(pgType string, nullable bool) string {
	var baseType string
	switch pgType {
	case "uuid":
		baseType = "UUID"
	case "varchar", "text", "character varying":
		baseType = "String"
	case "integer", "int", "int4", "bigint", "int8", "smallint", "int2":
		baseType = "Int"
	case "decimal", "numeric", "real", "float4", "double precision", "float8":
		baseType = "Float"
	case "boolean", "bool":
		baseType = "Boolean"
	case "timestamp", "timestamptz", "timestamp without time zone", "timestamp with time zone", "date":
		baseType = "Time"
	case "jsonb", "json":
		baseType = "JSONObject"
	default:
		baseType = "String"
	}

	if !nullable {
		return baseType + "!"
	}
	return baseType
}

func mapPGTypeToTS(pgType string, columnName string) string {
	// Map based on actual PostgreSQL type
	// ID columns: uuid → string, integer → number
	switch pgType {
	case "uuid", "varchar", "text", "character varying":
		return "string"
	case "integer", "int", "int4", "bigint", "int8", "smallint", "int2", "decimal", "numeric", "real", "float4", "double precision", "float8":
		return "number"
	case "boolean", "bool":
		return "boolean"
	case "timestamp", "timestamptz", "timestamp without time zone", "timestamp with time zone", "date":
		return "string" // ISO string
	case "jsonb", "json":
		return "Record<string, any>"
	case "bytea":
		return "Uint8Array"
	default:
		return "any"
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
