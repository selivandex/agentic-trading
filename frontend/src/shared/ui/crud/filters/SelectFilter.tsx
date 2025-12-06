/** @format */

"use client";

import { Select } from "@/shared/base/select";
import type { FilterComponentProps } from "./types";

/**
 * Select filter component
 */
export function SelectFilter({ filter, value, onChange }: FilterComponentProps) {
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
}
