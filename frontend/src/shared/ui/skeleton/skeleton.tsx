/**
 * Loading Skeleton Components
 *
 * Placeholder components that display while content is loading
 * Provides better UX than spinners for content-heavy interfaces
 *
 * @format
 */

"use client";

import { cx } from "@/utils/cx";

interface SkeletonProps {
  className?: string;
  variant?: "text" | "circular" | "rectangular";
  width?: string | number;
  height?: string | number;
  animation?: "pulse" | "wave" | "none";
}

/**
 * Base Skeleton component
 */
export const Skeleton = ({
  className,
  variant = "rectangular",
  width,
  height,
  animation = "pulse",
}: SkeletonProps) => {
  const variantClasses = {
    text: "h-4 rounded",
    circular: "rounded-full",
    rectangular: "rounded-lg",
  };

  const animationClasses = {
    pulse: "animate-pulse",
    wave: "animate-shimmer",
    none: "",
  };

  const style: React.CSSProperties = {};
  if (width) style.width = typeof width === "number" ? `${width}px` : width;
  if (height)
    style.height = typeof height === "number" ? `${height}px` : height;

  return (
    <div
      className={cx(
        "bg-tertiary",
        variantClasses[variant],
        animationClasses[animation],
        className
      )}
      style={style}
      aria-label="Loading..."
      role="status"
    />
  );
};

/**
 * Table row skeleton
 */
export const TableRowSkeleton = ({ columns = 5 }: { columns?: number }) => {
  return (
    <tr className="border-b border-secondary">
      {Array.from({ length: columns }).map((_, i) => (
        <td key={i} className="px-6 py-4">
          <Skeleton variant="text" className="h-4 w-full" />
        </td>
      ))}
    </tr>
  );
};

/**
 * Card skeleton
 */
export const CardSkeleton = () => {
  return (
    <div className="rounded-lg border border-secondary bg-primary p-6 shadow-sm">
      <div className="space-y-4">
        <Skeleton variant="text" className="h-6 w-3/4" />
        <Skeleton variant="text" className="h-4 w-full" />
        <Skeleton variant="text" className="h-4 w-5/6" />
        <div className="flex gap-2 pt-2">
          <Skeleton variant="rectangular" className="h-9 w-20" />
          <Skeleton variant="rectangular" className="h-9 w-20" />
        </div>
      </div>
    </div>
  );
};

/**
 * Avatar skeleton
 */
export const AvatarSkeleton = ({
  size = "md",
}: {
  size?: "sm" | "md" | "lg";
}) => {
  const sizes = {
    sm: "h-8 w-8",
    md: "h-10 w-10",
    lg: "h-12 w-12",
  };

  return <Skeleton variant="circular" className={sizes[size]} />;
};

/**
 * List item skeleton
 */
export const ListItemSkeleton = () => {
  return (
    <div className="flex items-center gap-4 py-4">
      <AvatarSkeleton />
      <div className="flex-1 space-y-2">
        <Skeleton variant="text" className="h-4 w-1/3" />
        <Skeleton variant="text" className="h-3 w-2/3" />
      </div>
    </div>
  );
};

/**
 * Scenario Table skeleton
 */
export const ScenarioTableSkeleton = ({ rows = 5 }: { rows?: number }) => {
  return (
    <div className="rounded-xl border border-secondary bg-primary shadow-sm">
      {/* Header */}
      <div className="border-b border-secondary px-6 py-5">
        <div className="flex items-center justify-between">
          <div className="space-y-2">
            <Skeleton variant="text" className="h-6 w-32" />
            <Skeleton variant="text" className="h-4 w-64" />
          </div>
          <Skeleton variant="rectangular" className="h-10 w-36" />
        </div>
      </div>

      {/* Table */}
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="border-b border-secondary bg-secondary_subtle">
            <tr>
              <th className="px-6 py-3">
                <Skeleton variant="text" className="h-4 w-16" />
              </th>
              <th className="px-6 py-3">
                <Skeleton variant="text" className="h-4 w-16" />
              </th>
              <th className="px-6 py-3">
                <Skeleton variant="text" className="h-4 w-24" />
              </th>
              <th className="px-6 py-3">
                <Skeleton variant="text" className="h-4 w-20" />
              </th>
              <th className="px-6 py-3">
                <Skeleton variant="text" className="h-4 w-20" />
              </th>
              <th className="px-6 py-3">
                <Skeleton variant="text" className="h-4 w-8" />
              </th>
            </tr>
          </thead>
          <tbody>
            {Array.from({ length: rows }).map((_, i) => (
              <TableRowSkeleton key={i} columns={6} />
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};
