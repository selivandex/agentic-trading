<!-- @format -->

# CRUD System Documentation

Generic, type-safe CRUD system for Next.js + GraphQL + FSD.

## Overview

This CRUD system provides a complete solution for Create, Read, Update, Delete operations with:

- **Zero boilerplate**: Define config once, get full CRUD UI
- **Type-safe**: Full TypeScript support with generics
- **GraphQL integration**: Built on Apollo Client
- **Form validation**: React Hook Form + Zod
- **FSD compliant**: Follows Feature-Sliced Design architecture
- **Customizable**: Every aspect can be customized

## Architecture

### File Structure

```
frontend/src/
├── shared/
│   ├── lib/
│   │   └── crud/                    # CRUD logic layer
│   │       ├── types.ts             # TypeScript definitions
│   │       ├── context.tsx          # React context + provider
│   │       ├── use-crud-query.ts    # Query hooks
│   │       ├── use-crud-mutations.ts # Mutation hooks
│   │       ├── utils.ts             # Helper functions
│   │       ├── README.md            # Detailed docs
│   │       └── index.ts             # Public API
│   └── ui/
│       └── crud/                    # CRUD UI layer
│           ├── Crud.tsx             # Main orchestrator component
│           ├── CrudTable.tsx        # List/index view
│           ├── CrudForm.tsx         # Create/edit forms
│           ├── CrudShow.tsx         # Detail view
│           └── index.ts             # Public API
└── entities/
    └── {entity}/
        └── lib/
            └── crud-config.ts       # Entity-specific CRUD config
```

## Quick Start

### 1. Import and Use

The simplest way to use CRUD:

```tsx
import { Crud } from "@/shared/ui/crud";
import { strategyCrudConfig } from "@/entities/strategy";

export default function StrategiesPage() {
  return <Crud config={strategyCrudConfig} />;
}
```

That's it! You get:

- ✅ Table view with sorting, pagination, search
- ✅ Create form with validation
- ✅ Edit form pre-filled with data
- ✅ Detail view
- ✅ Delete functionality
- ✅ Custom actions

## Configuration

### Basic Configuration

```typescript
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
  // Resource names (displayed in UI)
  resourceName: "Strategy",
  resourceNamePlural: "Strategies",

  // GraphQL operations
  graphql: {
    list: {
      query: GET_STRATEGIES,
      dataPath: "strategies", // Path in response: data.strategies
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

  // Table columns (what to display in list view)
  columns: [
    {
      key: "name",
      label: "Name",
      sortable: true,
    },
    {
      key: "status",
      label: "Status",
      render: (strategy) => <Badge>{strategy.status}</Badge>,
    },
    {
      key: "actions", // Special column for actions
      label: "",
    },
  ],

  // Form fields (for create/edit)
  formFields: [
    {
      name: "name",
      label: "Strategy Name",
      type: "text",
      validation: z.string().min(1, "Required"),
    },
    {
      name: "description",
      label: "Description",
      type: "textarea",
    },
  ],
};
```

## Features

### 1. Table View (Index)

**Automatic Features:**

- Pagination
- Sorting (if `sortable: true`)
- Row selection (if `enableSelection: true`)
- Search (if `enableSearch: true`)
- Custom column rendering
- Row actions dropdown

**Column Configuration:**

```typescript
columns: [
  {
    key: "name",
    label: "Name",
    sortable: true,
    width: "30%",
    hideOnMobile: false,
  },
  {
    key: "amount",
    label: "Amount",
    sortable: true,
    render: (entity) => `$${entity.amount.toFixed(2)}`,
  },
];
```

### 2. Create/Edit Forms

**Automatic Features:**

- Form validation (Zod)
- Error messages
- Field types: text, email, number, textarea, select, checkbox, date, datetime
- Custom fields via render function
- Responsive grid layout (12 columns)

**Form Field Types:**

```typescript
formFields: [
  // Text input
  {
    name: "name",
    label: "Name",
    type: "text",
    placeholder: "Enter name",
    helperText: "This will be visible to users",
    validation: z.string().min(1),
    colSpan: 6, // Half width
  },

  // Select dropdown
  {
    name: "status",
    label: "Status",
    type: "select",
    options: [
      { label: "Active", value: "active" },
      { label: "Inactive", value: "inactive" },
    ],
    colSpan: 6,
  },

  // Textarea
  {
    name: "description",
    label: "Description",
    type: "textarea",
    colSpan: 12, // Full width
  },

  // Number input
  {
    name: "amount",
    label: "Amount",
    type: "number",
    validation: z.number().min(0),
  },

  // Checkbox
  {
    name: "isActive",
    label: "Active",
    type: "checkbox",
    defaultValue: false,
  },

  // Date picker
  {
    name: "startDate",
    label: "Start Date",
    type: "date",
  },

  // Custom field
  {
    name: "custom",
    label: "Custom Field",
    type: "custom",
    render: ({ value, onChange, error }) => (
      <CustomComponent value={value} onChange={onChange} error={error} />
    ),
  },
];
```

### 3. Detail View (Show)

Displays all form fields in a read-only format.

Automatic formatting:

- Dates → `toLocaleDateString()`
- Numbers → `toLocaleString()`
- Booleans → "Yes" / "No"
- Objects → JSON pretty-print

### 4. Custom Actions

Add custom buttons to rows or bulk operations:

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
      destructive: true, // Shows in red
      disabled: (entity) => entity.status === "archived",
      hidden: (entity) => !entity.canArchive,
    },
  ],

  bulkActions: [
    {
      key: "bulkDelete",
      label: "Delete Selected",
      icon: <TrashIcon />,
      onClick: async (entities) => {
        await deleteMultiple(entities.map(e => e.id));
      },
      destructive: true,
    },
  ],
}
```

### 5. Data Transformations

Transform data between GraphQL and forms:

```typescript
{
  // Transform entity before loading into edit form
  transformBeforeEdit: (entity) => ({
    ...entity,
    // Convert Decimal to number
    amount: Number(entity.amount),
    // Parse JSON string
    config: JSON.parse(entity.config),
  }),

  // Transform form data before sending to create mutation
  transformBeforeCreate: (data) => ({
    ...data,
    // Convert number to string
    amount: String(data.amount),
    // Stringify JSON
    config: JSON.stringify(data.config),
  }),

  // Transform form data before sending to update mutation
  transformBeforeUpdate: (data) => ({
    ...data,
    amount: String(data.amount),
  }),
}
```

### 6. Search and Filters

Enable search:

```typescript
{
  enableSearch: true, // Adds search input to table header
}
```

Add custom filters:

```typescript
{
  filters: [
    {
      key: "status",
      label: "Status",
      type: "select",
      options: [
        { label: "All", value: "" },
        { label: "Active", value: "active" },
        { label: "Paused", value: "paused" },
      ],
    },
    {
      key: "dateRange",
      label: "Date Range",
      type: "daterange",
    },
  ],
}
```

## Advanced Usage

### Programmatic Navigation

Use the context to navigate programmatically:

```tsx
import { useCrudContext } from "@/shared/lib/crud";

function CustomComponent() {
  const { actions } = useCrudContext();

  return <button onClick={() => actions.goToNew()}>Create New</button>;
}
```

### Custom Mutations

Override default mutation behavior:

```tsx
import { useCrudMutations } from "@/shared/lib/crud";

function CustomForm() {
  const { create, update } = useCrudMutations(config);

  const handleSubmit = async (data) => {
    // Custom logic before create
    const enrichedData = {
      ...data,
      timestamp: new Date().toISOString(),
    };

    await create(enrichedData);

    // Custom logic after create
    trackEvent("entity_created");
  };

  return <form onSubmit={handleSubmit}>...</form>;
}
```

### Standalone Components

Use CRUD components individually:

```tsx
import { CrudTable, CrudForm, CrudShow } from "@/shared/ui/crud";
import { CrudProvider } from "@/shared/lib/crud";

function CustomCrudPage() {
  return (
    <CrudProvider config={config}>
      {/* Custom layout with individual components */}
      <div className="custom-layout">
        <CrudTable />
      </div>
    </CrudProvider>
  );
}
```

## Best Practices

### 1. Type Safety

Always define entity types:

```typescript
interface Strategy extends CrudEntity {
  id: string;
  name: string;
  status: "active" | "paused" | "closed";
  // ... other fields
}

const config: CrudConfig<Strategy> = {
  // TypeScript will enforce correct types
};
```

### 2. Validation

Use Zod for comprehensive validation:

```typescript
{
  name: "email",
  label: "Email",
  type: "email",
  validation: z
    .string()
    .email("Invalid email")
    .min(1, "Email is required"),
}
```

### 3. Error Handling

Provide user-friendly error messages:

```typescript
{
  emptyStateMessage: "No strategies found. Create one to get started.",
  errorMessage: "Failed to load strategies. Please refresh the page.",
}
```

### 4. Performance

For large datasets, use pagination:

```typescript
{
  defaultPageSize: 20, // Reasonable default
}
```

### 5. Accessibility

The system uses React Aria under the hood for accessibility.
Ensure you provide meaningful labels:

```typescript
{
  label: "Strategy Name", // Good
  // vs
  label: "Name", // Less clear
}
```

## Examples

See complete examples:

- `/app/(dashboard)/strategies-crud-example/page.tsx`
- `/entities/strategy/lib/crud-config.ts`

## API Reference

Full API documentation: `/shared/lib/crud/README.md`

## Troubleshooting

### Common Issues

**1. "useCrudContext must be used within CrudProvider"**

✅ Solution: Wrap components with `<CrudProvider>` or use `<Crud>` component

**2. Form validation not working**

✅ Solution: Check Zod schemas in `formFields[].validation`

**3. GraphQL query returns undefined**

✅ Solution: Verify `dataPath` matches your API response structure

**4. Table columns not rendering**

✅ Solution: Ensure entity has matching property keys

## Migration Checklist

To migrate existing CRUD to new system:

- [ ] Define GraphQL operations (queries + mutations)
- [ ] Create `CrudConfig` object in entity's `lib/crud-config.ts`
- [ ] Export config from entity's `lib/index.ts`
- [ ] Replace page component with `<Crud config={...} />`
- [ ] Test all CRUD operations
- [ ] Update navigation links
- [ ] Remove old CRUD components

## Performance Optimization

The CRUD system includes:

- ✅ Optimistic updates (cache management)
- ✅ Query deduplication (Apollo)
- ✅ Pagination (reduces payload)
- ✅ Lazy loading (code splitting)
- ✅ Memoization (React hooks)

## Contributing

When extending the CRUD system:

1. Update types in `shared/lib/crud/types.ts`
2. Implement in components
3. Add tests
4. Update this documentation
5. Create example usage

## Support

For questions or issues:

- Check examples in codebase
- Review type definitions
- Read inline documentation
- Ask in team chat

## Future Enhancements

Planned features:

- [ ] Advanced filtering UI
- [ ] Export to CSV/Excel
- [ ] Bulk edit operations
- [ ] Drag-and-drop reordering
- [ ] Real-time updates (subscriptions)
- [ ] Audit log integration
- [ ] Template presets

---

**Version:** 1.0.0
**Last Updated:** 2024-12-05
**Maintainer:** Development Team
