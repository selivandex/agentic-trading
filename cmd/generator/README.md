# Resource Generator

Automatic CRUD generator based on PostgreSQL schema.

## Features

- 🔍 Analyzes PostgreSQL table schema via `information_schema`
- 🏗️ Generates Clean Architecture layers (domain → repository → service → API)
- 📊 Auto-detects scopes from enum columns
- 🔎 Auto-generates filters based on column types
- 📝 Creates GraphQL schema with Relay pagination
- ⚛️ Generates React CRUD with FSD architecture
- 🎨 Follows project conventions (AGENTS.md rules)

## Usage

```bash
# Generate full CRUD for a table
make generate-resource table=orders

# Custom resource name
make generate-resource table=trading_orders resource=Order

# Dry run (show what would be generated)
make generate-resource table=positions --dry-run

# Skip frontend generation
make generate-resource table=sessions --backend-only
```

## What Gets Generated

### Backend (Go)

1. **Domain Layer** (`internal/domain/{resource}/`)
   - Entity with proper Go types mapped from PostgreSQL
   - Repository interface
   - Domain service with validation logic

2. **Repository Layer** (`internal/repository/postgres/`)
   - CRUD operations
   - `GetWithScope()` - filters by detected scopes
   - `GetScopes()` - GROUP BY for scope counts
   - Search across text columns

3. **Service Layer** (`internal/services/{resource}/`)
   - Application service with logging
   - Wraps domain service
   - Event publishing hooks (TODO comments)

4. **GraphQL Layer** (`internal/api/graphql/`)
   - Schema with types, queries, mutations
   - Connection types with scopes and filters
   - Resolvers with pagination
   - Scope definitions (from enums)
   - Filter definitions (from column types)

### Frontend (React/Next.js)

1. **Entity Layer** (`frontend/src/entities/{resource}/`)
   - TypeScript types
   - GraphQL queries/mutations
   - CRUD config with:
     - Table columns (from PG columns)
     - Form fields (from PG columns + validation)
     - Filters (from column types)
     - Scopes (from detected enums)
   - Manager component with actions

2. **Pages** (`frontend/src/app/(dashboard)/{resources}/`)
   - List page with search/filters/pagination
   - Detail page
   - Edit page
   - New page

## Column Type Mapping

### PostgreSQL → Go

| PostgreSQL Type       | Go Type            | GraphQL Type |
|----------------------|-------------------|--------------|
| `uuid`               | `uuid.UUID`       | `UUID!`      |
| `varchar`, `text`    | `string`          | `String!`    |
| `integer`, `bigint`  | `int`, `int64`    | `Int!`       |
| `decimal`, `numeric` | `decimal.Decimal` | `Decimal!`   |
| `boolean`            | `bool`            | `Boolean!`   |
| `timestamp`          | `time.Time`       | `Time!`      |
| `jsonb`              | `map[string]any`  | `JSONObject` |
| enum                 | Custom type       | Enum         |

### PostgreSQL → React Form Fields

| Column Type     | Form Field Type | Filter Type      |
|----------------|----------------|------------------|
| `varchar(N)`   | `text`         | `search`         |
| `text`         | `textarea`     | `search`         |
| `integer`      | `number`       | `number_range`   |
| `decimal`      | `number`       | `number_range`   |
| `boolean`      | `checkbox`     | `select`         |
| `timestamp`    | `datetime`     | `date_range`     |
| enum           | `select`       | `multiselect`    |
| foreign key    | `select`       | Custom query     |

## Scope Detection

Generator automatically creates scopes from:

1. **Boolean columns**: `is_active` → scopes: `active`, `inactive`
2. **Enum columns**: `status` ENUM('pending', 'active', 'closed') → scopes for each value
3. **Foreign keys**: Creates scope per related entity (optional)
4. **Always includes**: `all` scope

## Filter Detection

Generator creates filters based on column characteristics:

- **Text columns**: Full-text search
- **Numeric columns**: Range filters (min/max)
- **Boolean columns**: Toggle/select
- **Enum columns**: Multi-select
- **Timestamp columns**: Date range
- **Foreign keys**: Dropdown with API query

## Configuration File

Create `.generator.yaml` to customize generation:

```yaml
resources:
  orders:
    skip_columns:
      - internal_notes  # Don't expose in API
    
    scopes:
      # Override auto-detected scopes
      custom:
        - id: "pending_payment"
          name: "Pending Payment"
          where: "status = 'pending' AND payment_status = 'unpaid'"
    
    filters:
      # Add custom filters
      custom:
        - id: "high_value"
          name: "High Value Orders"
          type: "boolean"
          where: "total_amount > 1000"
    
    actions:
      # Add custom row actions
      - key: "mark_shipped"
        label: "Mark as Shipped"
        mutation: "markOrderShipped"
    
    frontend:
      columns_order:
        - order_number
        - customer_name
        - status
        - total
        - created_at
```

## Implementation Plan

### Phase 1: Schema Analysis ✅
- [ ] Connect to PostgreSQL
- [ ] Query `information_schema` for table structure
- [ ] Detect column types, constraints, indexes
- [ ] Identify enums, foreign keys, timestamps

### Phase 2: Backend Generation
- [ ] Generate domain entity from columns
- [ ] Generate repository interface
- [ ] Generate postgres repository with scopes/filters
- [ ] Generate domain service
- [ ] Generate application service
- [ ] Generate GraphQL schema
- [ ] Generate GraphQL resolvers
- [ ] Generate scope definitions
- [ ] Generate filter definitions

### Phase 3: Frontend Generation
- [ ] Generate TypeScript types
- [ ] Generate GraphQL queries
- [ ] Generate CRUD config
- [ ] Generate Manager component
- [ ] Generate pages (list/show/edit/new)

### Phase 4: Testing
- [ ] Generate repository tests
- [ ] Generate service tests
- [ ] Generate resolver tests
- [ ] Generate React component tests

### Phase 5: Migration Support
- [ ] Detect changes in existing resources
- [ ] Merge generated code with custom code
- [ ] Generate migration files for schema changes

## Example Output

```bash
$ make generate-resource table=positions

🔍 Analyzing table 'positions'...
   ✓ Found 15 columns
   ✓ Detected 3 enums: status, side, position_type
   ✓ Found 2 foreign keys: strategy_id, user_id
   ✓ Detected 4 scopes: all, open, closed, liquidated

📝 Generating backend...
   ✓ internal/domain/position/entity.go
   ✓ internal/domain/position/repository.go
   ✓ internal/domain/position/service.go
   ✓ internal/repository/postgres/position.go (458 lines)
   ✓ internal/services/position/service.go
   ✓ internal/api/graphql/schema/position.graphql
   ✓ internal/api/graphql/resolvers/position.resolvers.go
   ✓ internal/api/graphql/resolvers/position_scopes.go
   ✓ internal/api/graphql/resolvers/position_filters.go
   ✓ internal/api/graphql/resolvers/position_helpers.go

⚛️  Generating frontend...
   ✓ frontend/src/entities/position/model/types.ts
   ✓ frontend/src/entities/position/api/position.graphql.ts
   ✓ frontend/src/entities/position/lib/crud-config.tsx
   ✓ frontend/src/entities/position/ui/PositionManager.tsx
   ✓ frontend/src/app/(dashboard)/positions/page.tsx
   ✓ frontend/src/app/(dashboard)/positions/[id]/page.tsx
   ✓ frontend/src/app/(dashboard)/positions/[id]/edit/page.tsx

✅ Generated 18 files for resource 'Position'

📋 Next steps:
   1. Run: go run github.com/99designs/gqlgen generate
   2. Review generated files and customize as needed
   3. Run: make lint && make test
   4. Start dev server and test CRUD operations
```

## Limitations

- Does not handle complex relationships (many-to-many)
- Custom business logic must be added manually
- Advanced validation rules need manual implementation
- Complex filters require custom configuration

## Future Enhancements

- [ ] Generate from GraphQL schema (reverse direction)
- [ ] Support for nested resources
- [ ] Batch operation generation
- [ ] Export/import functionality
- [ ] Audit log integration
- [ ] Version control for generated files
- [ ] AI-assisted field label/description generation

