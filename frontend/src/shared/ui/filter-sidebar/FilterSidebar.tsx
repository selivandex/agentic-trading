/** @format */

"use client";

import { useFilterSidebar } from "@/shared/lib/filter-sidebar-context";
import { cx } from "@/utils/cx";

/**
 * FilterSidebar Component
 *
 * Second left sidebar for displaying filters, same width as main sidebar
 */
export function FilterSidebar() {
  const { content } = useFilterSidebar();

  if (!content || !content.isVisible) {
    return null;
  }

  return (
    <aside
      className={cx(
        "flex h-full w-64 flex-col border-r border-secondary bg-primary"
      )}
    >
      <div className="flex-1 overflow-y-auto p-6">{content.component}</div>
    </aside>
  );
}
