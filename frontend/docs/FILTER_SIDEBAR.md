<!-- @format -->

# Filter Sidebar System

## Overview

The Filter Sidebar system provides a consistent way to display filters in a dedicated left sidebar, positioned between the main navigation sidebar and the content area.

## Architecture

### Components

1. **FilterSidebarContext** (`@/shared/lib/filter-sidebar-context`)
   - Manages the global state of the filter sidebar
   - Provides `useFilterSidebar` hook for components to register/unregister filters
   - Handles visibility state

2. **FilterSidebar** (`@/shared/ui/filter-sidebar`)
   - Renders the actual sidebar component
   - Width: `w-64` (256px) - same as main navigation sidebar
   - Automatically shows/hides based on context state

3. **DashboardLayout Integration**
   - Wraps the entire dashboard in `FilterSidebarProvider`
   - Renders `FilterSidebar` between main sidebar and content

## Usage

### Registering Filters

Components can register filters using the `useFilterSidebar` hook:

```typescript
import { useFilterSidebar } from "@/shared/lib/filter-sidebar-context";
import { useEffect } from "react";

function MyComponent() {
  const { setContent } = useFilterSidebar();

  useEffect(() => {
    // Register filters
    setContent({
      isVisible: true,
      component: (
        <div>
          <h3>Filters</h3>
          {/* Your filter components */}
        </div>
      ),
    });

    // Cleanup on unmount
    return () => {
      setContent(null);
    };
  }, [/* dependencies */]);

  return <div>{/* Your component content */}</div>;
}
```

### CRUD System Integration

The CRUD system automatically registers filters when `dynamicFilters.enabled` is true:

- Filters are displayed in the sidebar
- Apply/Clear buttons are included
- Sidebar automatically hides when navigating away from CRUD list views

## Layout Structure

```
┌─────────────┬─────────────┬────────────────────────┐
│   Main      │   Filter    │      Content           │
│   Sidebar   │   Sidebar   │                        │
│   (w-64)    │   (w-64)    │      (flex-1)          │
│             │             │                        │
│   Nav       │   Filters   │   Page Content         │
│   Items     │   (when     │                        │
│             │   active)   │                        │
│             │             │                        │
└─────────────┴─────────────┴────────────────────────┘
```

## Benefits

1. **Consistent UI**: All filters appear in the same location
2. **More Space**: Content area gets full width when filters are not needed
3. **Better UX**: Filters are always accessible and don't overlap content
4. **Scalable**: Easy to add filters to any page

## Migration Notes

- Old right sidebar implementation in `CrudList.tsx` has been removed
- Filters now render in the left sidebar using the context system
- No changes needed to filter components themselves
