/** @format */

import type { ComponentType } from "react";
import type { FilterComponentProps } from "./types";
import { TextFilter } from "./TextFilter";
import { NumberFilter } from "./NumberFilter";
import { SelectFilter } from "./SelectFilter";
import { MultiSelectFilter } from "./MultiSelectFilter";
import { BooleanFilter } from "./BooleanFilter";
import { DateFilter } from "./DateFilter";
import { DateRangeFilter } from "./DateRangeFilter";
import { NumberRangeFilter } from "./NumberRangeFilter";

/**
 * Registry of filter components by type
 * Maps filter type to corresponding React component
 *
 * Benefits:
 * - Open-Closed Principle: Easy to add new filter types
 * - Single Responsibility: Each component handles one filter type
 * - DRY: No switch statements, single source of truth
 */
export const FILTER_COMPONENTS: Record<
  string,
  ComponentType<FilterComponentProps>
> = {
  TEXT: TextFilter,
  NUMBER: NumberFilter,
  SELECT: SelectFilter,
  MULTISELECT: MultiSelectFilter,
  BOOLEAN: BooleanFilter,
  DATE: DateFilter,
  DATE_RANGE: DateRangeFilter,
  NUMBER_RANGE: NumberRangeFilter,
};

// Export types
export type { FilterComponentProps } from "./types";

// Export individual components for direct usage if needed
export {
  TextFilter,
  NumberFilter,
  SelectFilter,
  MultiSelectFilter,
  BooleanFilter,
  DateFilter,
  DateRangeFilter,
  NumberRangeFilter,
};
