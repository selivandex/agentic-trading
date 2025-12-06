/** @format */

"use client";

import { Input } from "@/components/base/input/input";
import type { FilterComponentProps } from "./types";

/**
 * Number filter component
 */
export function NumberFilter({ filter, value, onChange }: FilterComponentProps) {
  return (
    <Input
      type="number"
      placeholder={filter.placeholder || filter.name}
      value={value ? String(value) : ""}
      onChange={(newValue) =>
        onChange(filter.id, newValue ? Number(newValue) : null)
      }
      label={filter.name}
      min={filter.min}
      max={filter.max}
    />
  );
}

