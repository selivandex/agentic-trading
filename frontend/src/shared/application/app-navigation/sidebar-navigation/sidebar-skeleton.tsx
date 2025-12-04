/** @format */

"use client";

import { cx } from "@/utils/cx";

interface SidebarSkeletonProps {
  /** Additional CSS classes to apply to the sidebar. */
  className?: string;
  /** Number of navigation items to show */
  itemsCount?: number;
}

export const SidebarSkeleton = ({
  className,
  itemsCount = 7,
}: SidebarSkeletonProps) => {
  // Predefined widths for skeleton items to avoid using Math.random() in render
  const widths = ["60%", "75%", "65%", "80%", "70%", "85%", "68%"];

  return (
    <aside
      className={cx(
        "flex h-full w-64 flex-col border-r border-secondary bg-primary",
        className
      )}
    >
      {/* Navigation items */}
      <nav className="flex-1 overflow-y-auto">
        <div className="mt-4 flex flex-col gap-1 px-2 lg:px-4">
          {Array.from({ length: itemsCount }).map((_, index) => (
            <div
              key={index}
              className="flex h-10 items-center gap-3 rounded-md px-3"
            >
              {/* Icon skeleton */}
              <div className="size-5 animate-pulse rounded bg-secondary" />
              {/* Label skeleton */}
              <div
                className="h-4 animate-pulse rounded bg-secondary"
                style={{
                  width: widths[index % widths.length],
                }}
              />
            </div>
          ))}
        </div>
      </nav>

      {/* Account card skeleton */}
      <div className="border-t border-secondary p-4">
        <div className="relative flex items-center gap-3 rounded-xl p-3 ring-1 ring-secondary ring-inset">
          {/* Avatar skeleton */}
          <div className="size-10 shrink-0 animate-pulse rounded-full bg-secondary" />

          {/* Text content skeleton */}
          <div className="flex flex-1 flex-col gap-1.5">
            {/* Name skeleton */}
            <div className="h-4 w-24 animate-pulse rounded bg-secondary" />
            {/* Subtitle skeleton */}
            <div className="h-3 w-32 animate-pulse rounded bg-secondary" />
          </div>

          {/* Chevron button skeleton */}
          <div className="absolute top-1.5 right-1.5">
            <div className="size-7 animate-pulse rounded-md bg-secondary" />
          </div>
        </div>
      </div>
    </aside>
  );
};
