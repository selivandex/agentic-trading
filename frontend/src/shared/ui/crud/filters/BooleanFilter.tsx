/** @format */

"use client";

import type { FilterComponentProps } from "./types";

/**
 * Boolean (checkbox) filter component
 */
export function BooleanFilter({
  filter,
  value,
  onChange,
}: FilterComponentProps) {
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
}

