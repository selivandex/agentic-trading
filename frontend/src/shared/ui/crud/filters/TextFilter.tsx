/** @format */

"use client";

import { Input } from "@/components/base/input/input";
import type { FilterComponentProps } from "./types";

/**
 * Text filter component
 */
export function TextFilter({ filter, value, onChange }: FilterComponentProps) {
  return (
    <Input
      type="text"
      placeholder={filter.placeholder || filter.name}
      value={String(value || "")}
      onChange={(newValue) => onChange(filter.id, newValue)}
      label={filter.name}
    />
  );
}

