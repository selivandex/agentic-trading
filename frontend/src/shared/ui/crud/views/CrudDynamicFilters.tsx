/** @format */

"use client";

import type { DateValue } from "react-aria-components";
import { Input } from "@/components/base/input/input";
import { Select } from "@/shared/base/select";
import { Slider } from "@/shared/base/slider";
import { DateRangePicker } from "@/shared/application/date-picker/date-range-picker";
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

    // Allow custom rendering
    if (config.customRenderer) {
      const customRender = config.customRenderer(filter, value, (newValue) =>
        onChange(filter.id, newValue)
      );
      if (customRender) {
        return customRender;
      }
    }

    // Default rendering by type
    switch (filter.type) {
      case "TEXT":
        return (
          <Input
            type="text"
            placeholder={filter.placeholder || filter.name}
            value={String(value || "")}
            onChange={(newValue) => onChange(filter.id, newValue)}
            label={filter.name}
          />
        );

      case "NUMBER":
        return (
          <Input
            type="number"
            placeholder={filter.placeholder || filter.name}
            value={value ? String(value) : ""}
            onChange={(newValue) =>
              onChange(filter.id, newValue ? Number(newValue) : null)
            }
            label={filter.name}
          />
        );

      case "SELECT":
        return (
          <Select
            label={filter.name}
            placeholder={filter.placeholder || `Select ${filter.name}`}
            selectedKey={value ? String(value) : undefined}
            onSelectionChange={(key) => onChange(filter.id, key)}
          >
            {filter.options?.map((option) => (
              <Select.Item key={option.value} id={option.value}>
                {option.label}
              </Select.Item>
            ))}
          </Select>
        );

      case "MULTISELECT":
        // TODO: Implement multiselect (needs different Select API)
        return (
          <div className="text-sm text-quaternary">
            Multiselect filter not implemented yet
          </div>
        );

      case "BOOLEAN":
        return (
          <div className="flex items-center gap-2">
            <label className="flex items-center gap-2 text-sm text-secondary">
              <input
                type="checkbox"
                checked={Boolean(value)}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                  onChange(filter.id, e.target.checked)
                }
                className="h-4 w-4 rounded border-border-secondary"
              />
              {filter.name}
            </label>
          </div>
        );

      case "DATE":
        return (
          <Input
            type="date"
            label={filter.name}
            value={value ? String(value) : ""}
            onChange={(newValue: string) => onChange(filter.id, newValue)}
            placeholder={filter.placeholder}
          />
        );

      case "DATE_RANGE": {
        const dateRangeValue = value as
          | { start: DateValue; end: DateValue }
          | null
          | undefined;
        return (
          <div>
            <label className="mb-1.5 block text-sm font-medium text-secondary">
              {filter.name}
            </label>
            <DateRangePicker
              value={dateRangeValue ?? null}
              onChange={(range) => onChange(filter.id, range)}
              onApply={() => {
                // Applied when user clicks Apply button
              }}
            />
          </div>
        );
      }

      case "NUMBER_RANGE": {
        // Parse min/max from value or use defaults
        const rangeValue = value as [number, number] | null | undefined;
        const currentValues = rangeValue || [0, 100];

        return (
          <div className="space-y-3">
            <label className="block text-sm font-medium text-secondary">
              {filter.name}
            </label>
            <Slider
              value={currentValues}
              onChange={(newValues) => onChange(filter.id, newValues)}
              minValue={0}
              maxValue={100}
              labelPosition="bottom"
              labelFormatter={(val) => String(val)}
              formatOptions={{ style: "decimal" }}
            />
            <div className="grid grid-cols-2 gap-2 text-xs text-tertiary">
              <div>Min: {currentValues[0]}</div>
              <div className="text-right">Max: {currentValues[1]}</div>
            </div>
          </div>
        );
      }

      default:
        return null;
    }
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
