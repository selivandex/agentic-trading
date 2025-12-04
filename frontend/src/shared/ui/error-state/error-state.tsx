/** @format */

"use client";

import { AlertCircle, RefreshCw04 } from "@untitledui/icons";
import { Button } from "@/shared/base/buttons";
import { cx } from "@/utils/cx";

export interface ErrorStateProps {
  /** Error title */
  title?: string;
  /** Error message */
  message: string;
  /** Visual variant */
  variant?: "error" | "warning" | "info";
  /** Show retry button */
  showRetry?: boolean;
  /** Retry button callback */
  onRetry?: () => void;
  /** Custom retry button label */
  retryLabel?: string;
  /** Additional CSS classes */
  className?: string;
}

const VARIANT_STYLES = {
  error: {
    container: "border-error-300 bg-error-50",
    icon: "text-error-600",
    title: "text-error-900",
    message: "text-error-700",
  },
  warning: {
    container: "border-warning-300 bg-warning-50",
    icon: "text-warning-600",
    title: "text-warning-900",
    message: "text-warning-700",
  },
  info: {
    container: "border-blue-300 bg-blue-50",
    icon: "text-blue-600",
    title: "text-blue-900",
    message: "text-blue-700",
  },
};

/**
 * ErrorState - Display error messages with optional retry
 *
 * Generic error display component for showing API errors, loading failures, etc.
 *
 * @example
 * ```tsx
 * <ErrorState
 *   title="Failed to load"
 *   message="Could not load scenarios. Please try again."
 *   showRetry
 *   onRetry={() => refetch()}
 * />
 * ```
 */
export const ErrorState = ({
  title = "Error",
  message,
  variant = "error",
  showRetry = false,
  onRetry,
  retryLabel = "Try again",
  className,
}: ErrorStateProps) => {
  const styles = VARIANT_STYLES[variant];

  return (
    <div
      className={cx("rounded-lg border p-4", styles.container, className)}
      role="alert"
      aria-live="polite"
    >
      <div className="flex items-start gap-3">
        {/* Icon */}
        <div className="flex-shrink-0">
          <AlertCircle className={cx("h-5 w-5", styles.icon)} />
        </div>

        {/* Content */}
        <div className="flex-1 space-y-1">
          <h3 className={cx("text-sm font-semibold", styles.title)}>{title}</h3>
          <p className={cx("text-sm", styles.message)}>{message}</p>

          {/* Retry button */}
          {showRetry && onRetry && (
            <div className="mt-3">
              <Button
                color="secondary"
                size="sm"
                onClick={onRetry}
                iconLeading={RefreshCw04}
              >
                {retryLabel}
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
