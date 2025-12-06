# Frontend: Dynamic Filters Implementation Guide

## Overview

This guide shows how to implement dynamic filters on the frontend using the `optionsQuery` field returned by the backend.

## Filter Types

### Static Options (Predefined)
```typescript
{
  id: "risk_tolerance",
  name: "Risk Tolerance",
  type: "SELECT",
  options: [
    { value: "CONSERVATIVE", label: "Conservative" },
    { value: "MODERATE", label: "Moderate" }
  ],
  optionsQuery: null
}
```

**Frontend renders:** Regular select dropdown with provided options.

### Dynamic Options (Fetched from Backend)
```typescript
{
  id: "user_id",
  name: "User",
  type: "SELECT",
  options: null,
  optionsQuery: "users",
  optionsQueryArgs: {
    first: 100,
    role: "ADMIN"
  }
}
```

**Frontend logic:**
1. Detect `optionsQuery` is not null
2. Call the specified GraphQL query
3. Transform results into `{value, label}` format
4. Render select dropdown with fetched options

## Implementation Example

### 1. Filter Hook

```typescript
import { useQuery } from '@apollo/client';
import { useMemo } from 'react';

interface Filter {
  id: string;
  name: string;
  type: 'SELECT' | 'MULTISELECT' | 'TEXT' | 'NUMBER';
  options?: { value: string; label: string }[];
  optionsQuery?: string;
  optionsQueryArgs?: Record<string, any>;
  placeholder?: string;
}

export function useDynamicFilterOptions(filter: Filter) {
  // Skip query if static options provided
  const shouldFetch = !filter.options && filter.optionsQuery;

  const query = useMemo(() => {
    if (!shouldFetch || !filter.optionsQuery) return null;

    // Build GraphQL query dynamically
    // This is simplified - you may want to use gql`` tagged template
    return `
      query Get${capitalize(filter.optionsQuery)}($args: JSONObject) {
        ${filter.optionsQuery}(args: $args) {
          edges {
            node {
              id
              name
            }
          }
        }
      }
    `;
  }, [filter.optionsQuery, shouldFetch]);

  const { data, loading, error } = useQuery(query, {
    variables: { args: filter.optionsQueryArgs },
    skip: !shouldFetch,
  });

  const options = useMemo(() => {
    // Use static options if provided
    if (filter.options) {
      return filter.options;
    }

    // Transform dynamic data to options format
    if (data && filter.optionsQuery) {
      const edges = data[filter.optionsQuery]?.edges || [];
      return edges.map((edge: any) => ({
        value: edge.node.id,
        label: edge.node.name,
      }));
    }

    return [];
  }, [filter.options, filter.optionsQuery, data]);

  return {
    options,
    loading,
    error,
    isDynamic: shouldFetch,
  };
}
```

### 2. Filter Component

```typescript
import { Select } from '@/shared/ui/select';
import { Input } from '@/shared/ui/input';
import { useDynamicFilterOptions } from './use-dynamic-filter-options';

interface FilterFieldProps {
  filter: Filter;
  value: any;
  onChange: (value: any) => void;
}

export function FilterField({ filter, value, onChange }: FilterFieldProps) {
  const { options, loading } = useDynamicFilterOptions(filter);

  switch (filter.type) {
    case 'SELECT':
    case 'MULTISELECT':
      return (
        <Select
          value={value}
          onChange={onChange}
          options={options}
          loading={loading}
          multiple={filter.type === 'MULTISELECT'}
          placeholder={filter.placeholder}
        />
      );

    case 'NUMBER':
      return (
        <Input
          type="number"
          value={value}
          onChange={(e) => onChange(parseFloat(e.target.value))}
          placeholder={filter.placeholder}
        />
      );

    case 'TEXT':
      return (
        <Input
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={filter.placeholder}
        />
      );

    default:
      return null;
  }
}
```

### 3. Usage in List View

```typescript
import { useQuery } from '@apollo/client';
import { useState } from 'react';
import { FilterField } from './FilterField';

const GET_STRATEGIES = gql`
  query GetStrategies($filters: JSONObject, $first: Int) {
    strategies(filters: $filters, first: $first) {
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
        options { value label }
        optionsQuery
        optionsQueryArgs
        placeholder
      }
    }
  }
`;

export function StrategiesPage() {
  const [filterValues, setFilterValues] = useState<Record<string, any>>({});

  const { data, loading } = useQuery(GET_STRATEGIES, {
    variables: {
      filters: filterValues,
      first: 20,
    },
  });

  const filters = data?.strategies?.filters || [];

  return (
    <div>
      {/* Filter Bar */}
      <div className="filters">
        {filters.map((filter) => (
          <FilterField
            key={filter.id}
            filter={filter}
            value={filterValues[filter.id]}
            onChange={(value) =>
              setFilterValues(prev => ({ ...prev, [filter.id]: value }))
            }
          />
        ))}
      </div>

      {/* Results */}
      <div className="results">
        {data?.strategies?.edges.map(({ node }) => (
          <div key={node.id}>{node.name}</div>
        ))}
      </div>
    </div>
  );
}
```

## Best Practices

1. **Cache Dynamic Options** - Use Apollo cache to avoid refetching options on every render
2. **Debounce Text Filters** - Add debouncing for TEXT/NUMBER inputs to reduce API calls
3. **Loading States** - Show skeleton loaders while options are fetching
4. **Error Handling** - Display error messages if options query fails
5. **Type Safety** - Generate TypeScript types from GraphQL schema

## Example Backend Queries for Dynamic Filters

### Users Filter
```graphql
query GetUsers($first: Int, $role: String) {
  users(first: $first, role: $role) {
    edges {
      node {
        id
        name
        email
      }
    }
  }
}
```

### Categories Filter
```graphql
query GetCategories {
  categories {
    edges {
      node {
        id
        name
      }
    }
  }
}
```

## Filter Value Format

When sending filters back to backend:

```typescript
// Single select
{ user_id: "uuid-123" }

// Multiselect
{ rebalance_frequency: ["DAILY", "WEEKLY"] }

// Number
{ min_capital: 1000 }

// Text
{ search: "BTC" }
```

## TypeScript Types

```typescript
export enum FilterType {
  TEXT = 'TEXT',
  NUMBER = 'NUMBER',
  DATE = 'DATE',
  SELECT = 'SELECT',
  MULTISELECT = 'MULTISELECT',
  BOOLEAN = 'BOOLEAN',
  DATE_RANGE = 'DATE_RANGE',
  NUMBER_RANGE = 'NUMBER_RANGE',
}

export interface FilterOption {
  value: string;
  label: string;
}

export interface Filter {
  id: string;
  name: string;
  type: FilterType;
  options?: FilterOption[];
  optionsQuery?: string;
  optionsQueryArgs?: Record<string, any>;
  defaultValue?: string;
  placeholder?: string;
}
```

