/** @format */

"use client";

import { useContext } from "react";
import { ErrorTrackerContext } from "./error-tracker-provider";
import type { ErrorTracker } from "./error-tracker";

/**
 * Hook to access error tracker instance
 *
 * Returns the error tracker provided by ErrorTrackerProvider.
 * Must be used within ErrorTrackerProvider context.
 *
 * @returns Error tracker instance
 *
 * @example
 * ```tsx
 * const errorTracker = useErrorTracker();
 *
 * try {
 *   // some code
 * } catch (error) {
 *   errorTracker.captureException(error as Error, {
 *     tags: { component: 'MyComponent' },
 *     extra: { userId: user.id }
 *   });
 * }
 * ```
 */
export const useErrorTracker = (): ErrorTracker => {
  const tracker = useContext(ErrorTrackerContext);

  if (!tracker) {
    throw new Error(
      "useErrorTracker must be used within ErrorTrackerProvider"
    );
  }

  return tracker;
};
