/** @format */

"use client";

import { CrudMultiSelectFilter } from "../views/CrudMultiSelectFilter";
import type { FilterComponentProps } from "./types";

/**
 * Multi-select filter component
 */
export function MultiSelectFilter({
  filter,
  value,
  onChange,
}: FilterComponentProps) {
  return (
    <CrudMultiSelectFilter filter={filter} value={value} onChange={onChange} />
  );
}

