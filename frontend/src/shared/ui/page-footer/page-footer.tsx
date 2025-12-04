/** @format */

"use client";

import type { ReactNode } from "react";
import { cx } from "@/utils/cx";
import { Skeleton } from "@/shared/ui/skeleton/skeleton";

export interface PageFooterProps {
  /** Background variant */
  background?: "transparent" | "primary";
  /** Action buttons */
  actions?: ReactNode;
  /** Additional className */
  className?: string;
}

export const PageFooter = ({
  background = "primary",
  actions,
  className,
}: PageFooterProps) => {
  const paddingClasses = "px-4 lg:px-8";

  return (
    <div
      className={cx(
        "relative w-full flex items-center justify-end py-4 border-t border-secondary",
        background === "primary" ? "bg-primary" : "bg-transparent",
        paddingClasses,
        className
      )}
    >
      {/* Actions */}
      {actions && <div className="flex items-center gap-3">{actions}</div>}
    </div>
  );
};

PageFooter.displayName = "PageFooter";

/**
 * PageFooterSkeleton - Loading state for PageFooter component
 *
 * Accepts the same props as PageFooter and automatically displays
 * skeleton placeholders for the provided layout configuration.
 */
export type PageFooterSkeletonProps = Omit<PageFooterProps, "actions"> & {
  /** Number of action buttons to show */
  actionCount?: number;
};

export const PageFooterSkeleton = ({
  background = "primary",
  actionCount = 2,
  className,
}: PageFooterSkeletonProps) => {
  const paddingClasses = "px-4 lg:px-8";

  return (
    <div
      className={cx(
        "relative w-full flex items-center justify-end py-4 border-t border-secondary",
        background === "primary" ? "bg-primary" : "bg-transparent",
        paddingClasses,
        className
      )}
    >
      {/* Actions Skeleton */}
      {actionCount > 0 && (
        <div className="flex items-center gap-3">
          {Array.from({ length: actionCount }).map((_, index) => (
            <Skeleton key={index} className="h-10 w-24 rounded-lg" />
          ))}
        </div>
      )}
    </div>
  );
};

PageFooterSkeleton.displayName = "PageFooterSkeleton";
