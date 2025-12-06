/** @format */

import type { FilterMetadata } from "@/shared/lib/crud/types";

/**
 * Common props for all filter components
 */
export interface FilterComponentProps {
  /** Filter metadata from backend */
  filter: FilterMetadata;
  /** Current filter value */
  value: unknown;
  /** Callback when filter value changes */
  onChange: (filterId: string, value: unknown) => void;
}
