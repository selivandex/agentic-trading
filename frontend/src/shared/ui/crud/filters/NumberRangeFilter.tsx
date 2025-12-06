/** @format */

"use client";

import { Slider } from "@/shared/base/slider";
import type { FilterComponentProps } from "./types";

/**
 * Number range filter component
 */
export function NumberRangeFilter({
  filter,
  value,
  onChange,
}: FilterComponentProps) {
  // Parse min/max from value or use defaults from filter metadata
  const rangeValue = value as [number, number] | null | undefined;
  const minValue = filter.min ?? 0;
  const maxValue = filter.max ?? 100;
  const currentValues = rangeValue || [minValue, maxValue];

  return (
    <div className="space-y-2">
      <label className="block text-sm font-medium text-secondary">
        {filter.name}
      </label>
      <div className="pt-2 pb-6">
        <Slider
          value={currentValues}
          onChange={(newValues) => onChange(filter.id, newValues)}
          minValue={minValue}
          maxValue={maxValue}
          labelPosition="bottom"
          labelFormatter={(val) => String(val)}
          formatOptions={{ style: "decimal" }}
        />
      </div>
    </div>
  );
}
