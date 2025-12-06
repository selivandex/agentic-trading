<!-- @format -->

# CRUD Library

Generic, type-safe CRUD system for Next.js + GraphQL + FSD architecture.

## Features

- ✅ **Type-safe**: Full TypeScript support with generics
- ✅ **GraphQL Integration**: Built on Apollo Client
- ✅ **Form Validation**: React Hook Form + Zod
- ✅ **Customizable**: Columns, forms, actions, filters
- ✅ **Built-in Features**: Pagination, sorting, filtering, search
- ✅ **FSD Architecture**: Follows Feature-Sliced Design principles
- ✅ **Optimistic Updates**: Automatic cache updates

## Quick Start

### 1. Define GraphQL Operations

```typescript
// entities/strategy/api/strategy.graphql.ts
import { gql } from "@apollo/client";

export const GET_STRATEGIES = gql`
  query GetStrategies($first: Int, $after: String) {
    strategies(first: $first, after: $after) {
      edges {
        node {
          id
          name
          status
          createdAt
        }
        cursor
      }
      pageInfo {
        hasNextPage
        hasPreviousPage
        startCursor
        endCursor
      }
      totalCount
    }
  }
`;

export const GET_STRATEGY = gql`
  query GetStrategy($id: UUID!) {
    strategy(id: $id) {
      id
      name
      description
      status
      createdAt
    }
  }
`;

export const CREATE_STRATEGY = gql`
  mutation CreateStrategy($input: CreateStrategyInput!) {
    createStrategy(input: $input) {
      id
      name
      status
    }
  }
`;

export const UPDATE_STRATEGY = gql`
  mutation UpdateStrategy($id: UUID!, $input: UpdateStrategyInput!) {
    updateStrategy(id: $id, input: $input) {
      id
      name
      status
    }
  }
`;

export const DELETE_STRATEGY = gql`
  mutation DeleteStrategy($id: UUID!) {
    deleteStrategy(id: $id)
  }
`;
```

### 2. Create CRUD Configuration

```typescript
// entities/strategy/lib/crud-config.ts
import { z } from "zod";
import type { CrudConfig } from "@/shared/lib/crud";
import type { Strategy } from "../model/types";
import {
  GET_STRATEGIES,
  GET_STRATEGY,
  CREATE_STRATEGY,
  UPDATE_STRATEGY,
  DELETE_STRATEGY,
} from "../api/strategy.graphql";

export const strategyCrudConfig: CrudConfig<Strategy> = {
  resourceName: "Strategy",
  resourceNamePlural: "Strategies",

  // GraphQL operations
  graphql: {
    list: {
      query: GET_STRATEGIES,
      dataPath: "strategies",
    },
    show: {
      query: GET_STRATEGY,
      variables: { id: "" },
      dataPath: "strategy",
    },
    create: {
      mutation: CREATE_STRATEGY,
      variables: {},
      dataPath: "createStrategy",
    },
    update: {
      mutation: UPDATE_STRATEGY,
      variables: {},
      dataPath: "updateStrategy",
    },
    destroy: {
      mutation: DELETE_STRATEGY,
      variables: { id: "" },
      dataPath: "deleteStrategy",
    },
  },

  // Table columns
  columns: [
    {
      key: "name",
      label: "Name",
      sortable: true,
    },
    {
      key: "status",
      label: "Status",
      sortable: true,
      render: (strategy) => (
        <Badge color={strategy.status === "active" ? "success" : "gray"}>
          {strategy.status}
        </Badge>
      ),
    },
    {
      key: "createdAt",
      label: "Created",
      sortable: true,
      render: (strategy) => new Date(strategy.createdAt).toLocaleDateString(),
    },
    {
      key: "actions",
      label: "",
    },
  ],

  // Form fields
  formFields: [
    {
      name: "name",
      label: "Strategy Name",
      type: "text",
      placeholder: "Enter strategy name",
      validation: z.string().min(1, "Name is required"),
      colSpan: 12,
    },
    {
      name: "description",
      label: "Description",
      type: "textarea",
      placeholder: "Enter strategy description",
      validation: z.string().optional(),
      colSpan: 12,
    },
    {
      name: "status",
      label: "Status",
      type: "select",
      options: [
        { label: "Active", value: "active" },
        { label: "Paused", value: "paused" },
        { label: "Closed", value: "closed" },
      ],
      colSpan: 6,
    },
  ],

  // Optional: Custom actions
  actions: [
    {
      key: "pause",
      label: "Pause",
      icon: <PauseIcon />,
      onClick: async (strategy) => {
        // Custom action handler
        await pauseStrategy(strategy.id);
      },
      hidden: (strategy) => strategy.status !== "active",
    },
  ],

  // Optional: Enable features
  enableSelection: true,
  enableSearch: true,
  defaultPageSize: 20,
};
```

### 3. Use in Page

```tsx
// app/(dashboard)/strategies/page.tsx
"use client";

import { Crud } from "@/shared/ui/crud";
import { strategyCrudConfig } from "@/entities/strategy";

export default function StrategiesPage() {
  return <Crud config={strategyCrudConfig} />;
}
```

## API Reference

### Types

#### `CrudConfig<TEntity>`

Main configuration object for CRUD operations.

```typescript
interface CrudConfig<TEntity extends CrudEntity> {
  // Resource names
  resourceName: string;
  resourceNamePlural: string;

  // GraphQL operations
  graphql: CrudGraphQLConfig<TEntity>;

  // Table configuration
  columns: CrudColumn<TEntity>[];

  // Form configuration
  formFields: CrudFormField<TEntity>[];

  // Optional: Row actions
  actions?: CrudAction<TEntity>[];

  // Optional: Bulk actions
  bulkActions?: CrudAction<TEntity>[];

  // Optional: Filters
  filters?: CrudFilter[];

  // Optional: Custom messages
  emptyStateMessage?: string;
  errorMessage?: string;

  // Optional: Feature flags
  enableSelection?: boolean;
  enableSearch?: boolean;
  defaultPageSize?: number;

  // Optional: Data transformations
  transformBeforeEdit?: (entity: TEntity) => Record<string, unknown>;
  transformBeforeCreate?: (
    data: Record<string, unknown>
  ) => Record<string, unknown>;
  transformBeforeUpdate?: (
    data: Record<string, unknown>
  ) => Record<string, unknown>;
  formValidationSchema?: ZodSchema;
}
```

#### `CrudColumn<TEntity>`

Table column definition.

```typescript
interface CrudColumn<TEntity> {
  key: string;
  label: string;
  tooltip?: string;
  render?: (entity: TEntity) => ReactNode;
  sortable?: boolean;
  width?: string | number;
  hideOnMobile?: boolean;
}
```

#### `CrudFormField<TEntity>`

Form field definition.

```typescript
interface CrudFormField<TEntity> {
  name: string;
  label: string;
  type:
    | "text"
    | "email"
    | "password"
    | "number"
    | "textarea"
    | "select"
    | "checkbox"
    | "radio"
    | "date"
    | "datetime"
    | "custom";
  placeholder?: string;
  helperText?: string;
  validation?: ZodSchema;
  options?: Array<{ label: string; value: string | number }>;
  render?: (props: CrudFormFieldRenderProps<TEntity>) => ReactNode;
  disabled?: boolean;
  hidden?: boolean;
  colSpan?: number; // 1-12
  defaultValue?: unknown;
}
```

### Components

#### `<Crud>`

Main CRUD component that orchestrates all views.

```tsx
<Crud
  config={crudConfig}
  mode="index" // "index" | "show" | "new" | "edit"
  entityId="123" // For show/edit modes
/>
```

#### `<CrudTable>`

Table view component (used internally by `<Crud>`).

#### `<CrudForm>`

Form component for create/edit (used internally by `<Crud>`).

#### `<CrudShow>`

Detail view component (used internally by `<Crud>`).

### Hooks

#### `useCrudContext()`

Access CRUD state and actions.

```tsx
const { config, state, actions } = useCrudContext<Strategy>();

// Navigate between views
actions.goToIndex();
actions.goToShow(id);
actions.goToNew();
actions.goToEdit(id);

// CRUD operations
await actions.create(data);
await actions.update(id, data);
await actions.destroy(id);
```

#### `useCrudListQuery()`

Fetch list of entities.

```tsx
const { entities, loading, error, refetch } = useCrudListQuery(config, state);
```

#### `useCrudShowQuery()`

Fetch single entity.

```tsx
const { entity, loading, error, refetch } = useCrudShowQuery(config, id);
```

#### `useCrudMutations()`

CRUD mutations (create, update, delete).

```tsx
const { create, update, destroy, isLoading } = useCrudMutations(config);
```

## Advanced Usage

### Custom Form Fields

```typescript
{
  name: "customField",
  label: "Custom Field",
  type: "custom",
  render: ({ value, onChange, error }) => (
    <CustomComponent
      value={value}
      onChange={onChange}
      error={error}
    />
  ),
}
```

### Data Transformation

```typescript
const config: CrudConfig<Strategy> = {
  // ... other config

  // Transform entity before loading into edit form
  transformBeforeEdit: (entity) => ({
    ...entity,
    // Convert nested objects to strings for form
    targetAllocations: JSON.stringify(entity.targetAllocations),
  }),

  // Transform form data before sending to API
  transformBeforeUpdate: (data) => ({
    ...data,
    // Parse strings back to objects
    targetAllocations: JSON.parse(data.targetAllocations as string),
  }),
};
```

### Custom Actions

```typescript
{
  actions: [
    {
      key: "export",
      label: "Export",
      icon: <DownloadIcon />,
      onClick: async (entity) => {
        await exportEntity(entity);
      },
    },
    {
      key: "archive",
      label: "Archive",
      icon: <ArchiveIcon />,
      onClick: async (entity) => {
        await archiveEntity(entity);
      },
      destructive: true,
      disabled: (entity) => entity.status === "archived",
      hidden: (entity) => entity.type === "system",
    },
  ],
}
```

## Best Practices

1. **Type Safety**: Always define entity types
2. **Validation**: Use Zod schemas for form validation
3. **Error Handling**: Provide custom error messages
4. **Performance**: Use pagination for large datasets
5. **UX**: Provide loading and empty states
6. **Testing**: Test CRUD configurations separately

## Architecture

The CRUD system follows FSD architecture:

```
shared/
├── lib/
│   └── crud/
│       ├── types.ts          # TypeScript types
│       ├── context.tsx       # React context
│       ├── use-crud-query.ts # Query hooks
│       ├── use-crud-mutations.ts # Mutation hooks
│       ├── utils.ts          # Helper functions
│       └── index.ts          # Public API
└── ui/
    └── crud/
        ├── Crud.tsx          # Main component
        ├── CrudTable.tsx     # Table view
        ├── CrudForm.tsx      # Form view
        ├── CrudShow.tsx      # Detail view
        └── index.ts          # Public API
```

## Migration Guide

To migrate existing CRUD implementations:

1. Define GraphQL operations
2. Create `CrudConfig` object
3. Replace custom components with `<Crud>`
4. Update routes to use CRUD modes

## Troubleshooting

### "useCrudContext must be used within CrudProvider"

Ensure components are wrapped with `<CrudProvider>` or use `<Crud>` component.

### Form validation not working

Check that Zod schemas are properly defined in `formFields[].validation`.

### GraphQL query not updating

Verify `dataPath` in GraphQL config matches your API response structure.

## Contributing

When adding new features:

1. Update types in `types.ts`
2. Implement in components
3. Update this documentation
4. Add tests

## License

Internal use only.
