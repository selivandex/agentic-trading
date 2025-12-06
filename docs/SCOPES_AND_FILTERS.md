# Dynamic Scopes and Filters System

## Overview

This document describes the dynamic scopes and filters system implemented for GraphQL queries. The system allows resources to define their own scopes (tabs) and filters, which are returned in GraphQL responses for frontend to render dynamically.

## Architecture

The system follows Clean Architecture principles:

1. **GraphQL Schema** (`internal/api/graphql/schema/*.graphql`) - Defines types `Scope` and `Filter`
2. **Relay Package** (`pkg/relay/`) - Shared utilities for scopes and filters
3. **Repository Layer** (`internal/repository/postgres/`) - SQL-based filtering with WHERE clauses
4. **Service Layer** (`internal/services/`) - Business logic for applying filters
5. **Resolver Layer** (`internal/api/graphql/resolvers/`) - Coordinates between layers

## Scopes

**Scopes** are predefined filter tabs (e.g., "All", "Active", "Paused", "Closed") with item counts.

### GraphQL Type

```graphql
type Scope {
  id: String!
  name: String!
  count: Int!
}
```

### Usage Example

Query:
```graphql
query {
  strategies(scope: "active", first: 10) {
    edges {
      node {
        id
        name
      }
    }
    scopes {
      id
      name
      count
    }
    totalCount
  }
}
```

Response:
```json
{
  "data": {
    "strategies": {
      "scopes": [
        { "id": "all", "name": "All", "count": 15 },
        { "id": "active", "name": "Active", "count": 8 },
        { "id": "paused", "name": "Paused", "count": 4 },
        { "id": "closed", "name": "Closed", "count": 3 }
      ],
      "totalCount": 8
    }
  }
}
```

### Implementation

1. Define scopes in `*_scopes.go`:
```go
func getStrategyScopeDefinitions() []relay.Scope {
    return []relay.Scope{
        {ID: "all", Name: "All", Count: 0},
        {ID: "active", Name: "Active", Count: 0},
    }
}
```

2. Service calculates counts via SQL `GROUP BY`:
```go
func (s *Service) GetStrategiesScopes(ctx context.Context, userID uuid.UUID) (map[string]int, error) {
    statusCounts, err := s.strategyRepo.CountByStatus(ctx, userID)
    // ... map to scope IDs
    return result, nil
}
```

## Filters

**Filters** are dynamic filter definitions that frontend can render as form inputs.

### GraphQL Types

```graphql
enum FilterType {
  TEXT
  NUMBER
  DATE
  SELECT
  MULTISELECT
  BOOLEAN
  DATE_RANGE
  NUMBER_RANGE
}

type FilterOption {
  value: String!
  label: String!
}

type Filter {
  id: String!
  name: String!
  type: FilterType!
  options: [FilterOption!]
  defaultValue: String
  placeholder: String
}
```

### Usage Example

Query:
```graphql
query {
  strategies(
    filters: {
      risk_tolerance: "CONSERVATIVE"
      min_capital: 1000
    }
    first: 10
  ) {
    edges {
      node {
        id
        name
      }
    }
    filters {
      id
      name
      type
      options {
        value
        label
      }
      placeholder
    }
  }
}
```

Response:
```json
{
  "data": {
    "strategies": {
      "filters": [
        {
          "id": "risk_tolerance",
          "name": "Risk Tolerance",
          "type": "SELECT",
          "options": [
            { "value": "CONSERVATIVE", "label": "Conservative" },
            { "value": "MODERATE", "label": "Moderate" },
            { "value": "AGGRESSIVE", "label": "Aggressive" }
          ],
          "optionsQuery": null,
          "placeholder": "Select risk tolerance"
        },
        {
          "id": "user_id",
          "name": "User",
          "type": "SELECT",
          "options": null,
          "optionsQuery": "users",
          "optionsQueryArgs": {
            "first": 100,
            "role": "ADMIN"
          },
          "placeholder": "Select user"
        },
        {
          "id": "min_capital",
          "name": "Minimum Capital",
          "type": "NUMBER",
          "options": null,
          "optionsQuery": null,
          "placeholder": "Enter minimum capital"
        }
      ]
    }
  }
}
```

**Frontend logic:**
- If `optionsQuery` is `null` → use static `options`
- If `optionsQuery` is not `null` → fetch options dynamically:
  ```graphql
  query {
    users(first: 100, role: "ADMIN") {
      edges {
        node {
          id
          name
        }
      }
    }
  }
  ```

### Static vs Dynamic Options

Filters can have either **static options** (predefined) or **dynamic options** (fetched from backend):

#### Static Options
Best for enums and fixed lists:
```go
{
    ID:   "risk_tolerance",
    Name: "Risk Tolerance",
    Type: relay.FilterTypeSelect,
    Options: []relay.FilterOption{
        {Value: "CONSERVATIVE", Label: "Conservative"},
        {Value: "MODERATE", Label: "Moderate"},
    },
}
```

#### Dynamic Options
Best for data that changes (users, categories, etc.):
```go
{
    ID:           "user_id",
    Name:         "User",
    Type:         relay.FilterTypeSelect,
    OptionsQuery: strPtr("users"), // GraphQL query name
    OptionsQueryArgs: map[string]interface{}{
        "first": 100,
        "role":  "ADMIN",
    },
    Placeholder: strPtr("Select user"),
}
```

Frontend will call the specified query to fetch options dynamically.

### Implementation

1. Define filters in `*_filters.go`:
```go
func getStrategyFilterDefinitions() []relay.FilterDefinition {
    return []relay.FilterDefinition{
        // Static options
        {
            ID:   "risk_tolerance",
            Name: "Risk Tolerance",
            Type: relay.FilterTypeSelect,
            Options: []relay.FilterOption{
                {Value: "CONSERVATIVE", Label: "Conservative"},
                {Value: "MODERATE", Label: "Moderate"},
            },
            Placeholder: strPtr("Select risk tolerance"),
        },
        // Dynamic options
        {
            ID:           "user_id",
            Name:         "User",
            Type:         relay.FilterTypeSelect,
            OptionsQuery: strPtr("users"),
            OptionsQueryArgs: map[string]interface{}{
                "first": 100,
            },
            Placeholder:  strPtr("Select user"),
        },
        // Number filter
        {
            ID:          "min_capital",
            Name:        "Minimum Capital",
            Type:        relay.FilterTypeNumber,
            Placeholder: strPtr("Enter minimum capital"),
        },
    }
}
```

2. Extend `FilterOptions` in domain repository interface:
```go
type FilterOptions struct {
    UserID          *uuid.UUID
    Status          *StrategyStatus
    RiskTolerance   *RiskTolerance
    MinCapital      *decimal.Decimal
    // ... more filters
}
```

3. Implement SQL WHERE in repository:
```go
func (r *StrategyRepository) GetWithFilter(ctx context.Context, filter strategy.FilterOptions) ([]*strategy.Strategy, error) {
    query := "SELECT * FROM user_strategies WHERE 1=1"

    if filter.RiskTolerance != nil {
        query += " AND risk_tolerance = $" + fmt.Sprintf("%d", argIdx)
        args = append(args, *filter.RiskTolerance)
    }

    if filter.MinCapital != nil {
        query += " AND allocated_capital >= $" + fmt.Sprintf("%d", argIdx)
        args = append(args, *filter.MinCapital)
    }
    // ...
}
```

4. Service parses JSON filters and applies them:
```go
func (s *Service) applyFiltersToOptions(filter *strategy.FilterOptions, filters map[string]interface{}) {
    for filterID, filterValue := range filters {
        switch filterID {
        case "risk_tolerance":
            if val, ok := filterValue.(string); ok {
                rt := strategy.RiskTolerance(val)
                filter.RiskTolerance = &rt
            }
        case "min_capital":
            if val, ok := filterValue.(float64); ok {
                minCap := decimal.NewFromFloat(val)
                filter.MinCapital = &minCap
            }
        }
    }
}
```

## Adding Scopes/Filters to New Resource

1. **GraphQL Schema**: Add `scopes` and `filters` fields to Connection type
2. **Create `*_scopes.go`**: Define scope definitions (metadata only)
3. **Create `*_filters.go`**: Define filter definitions
4. **Domain Layer**: Extend `FilterOptions` struct
5. **Repository**: Implement SQL WHERE for new filters
6. **Service**: Add methods for scope counts and filter parsing
7. **Resolver**: Wire up scopes/filters in connection response

## Benefits

✅ **SQL-based filtering** - Efficient database queries, not in-memory
✅ **Dynamic frontend** - Frontend automatically renders filters
✅ **Type-safe** - Full TypeScript/Go type safety
✅ **Reusable** - Shared utilities in `pkg/relay`
✅ **Testable** - Comprehensive test coverage
✅ **Clean Architecture** - Proper separation of concerns

## Testing

Run tests:
```bash
go test ./pkg/relay/... -v
```

All scope and filter utilities have comprehensive test coverage.
