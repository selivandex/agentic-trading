/** @format */

"use client";

import type { DateValue } from "react-aria-components";
import { DateRangePicker } from "@/shared/application/date-picker/date-range-picker";
import type { FilterComponentProps } from "./types";

/**
 * Date range filter component
 */
export function DateRangeFilter({
  filter,
  value,
  onChange,
}: FilterComponentProps) {
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
