/** @format */

"use client";

import { FILTER_COMPONENTS } from "../filters";
import type {
  CrudEntity,
  CrudDynamicFiltersConfig,
  FilterMetadata,
} from "@/shared/lib/crud/types";

interface CrudDynamicFiltersProps<_TEntity extends CrudEntity = CrudEntity> {
  /** Filters configuration */
  config: CrudDynamicFiltersConfig;
  /** Filter metadata from backend */
  filters?: FilterMetadata[];
  /** Current filter values */
  values: Record<string, unknown>;
  /** Callback when filter changes */
  onChange: (filterId: string, value: unknown) => void;
}

/**
 * CRUD Dynamic Filters Component
 * Displays filters based on metadata from backend
 *
 * Architecture:
 * - Uses registry pattern to map filter types to components
 * - Each filter type has its own component (SRP)
 * - Easy to extend with new filter types (OCP)
 * - No switch statements or complex conditionals (DRY)
 */
export function CrudDynamicFilters<TEntity extends CrudEntity = CrudEntity>({
  config,
  filters,
  values,
  onChange,
}: CrudDynamicFiltersProps<TEntity>) {
  // If no filters from backend, don't render
  if (!filters || filters.length === 0) {
    return null;
  }

  const renderFilter = (filter: FilterMetadata) => {
    const value = values[filter.id];

    // Allow custom rendering (hook for extensibility)
    if (config.customRenderer) {
      const customRender = config.customRenderer(filter, value, (newValue) =>
        onChange(filter.id, newValue)
      );
      if (customRender) {
        return customRender;
      }
    }

    // Get component from registry
    const FilterComponent = FILTER_COMPONENTS[filter.type];

    // If filter type is not registered, skip rendering
    if (!FilterComponent) {
      console.warn(`Unknown filter type: ${filter.type}`);
      return null;
    }

    // Render the appropriate filter component
    return (
      <FilterComponent filter={filter} value={value} onChange={onChange} />
    );
  };

  return (
    <div className="flex flex-col gap-4">
      {filters.map((filter) => (
        <div key={filter.id} className="w-full">
          {renderFilter(filter)}
        </div>
      ))}
    </div>
  );
}
