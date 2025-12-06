/** @format */

"use client";

import { Input } from "@/components/base/input/input";
import type { FilterComponentProps } from "./types";

/**
 * Date filter component
 */
export function DateFilter({ filter, value, onChange }: FilterComponentProps) {
  return (
    <Input
      type="date"
      label={filter.name}
      value={value ? String(value) : ""}
      onChange={(newValue: string) => onChange(filter.id, newValue)}
      placeholder={filter.placeholder}
    />
  );
}

