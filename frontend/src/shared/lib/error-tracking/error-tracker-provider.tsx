/** @format */

"use client";

import { createContext, useMemo } from "react";
import { env, isSentryConfigured } from "@/shared/config";
import type { ErrorTracker } from "./error-tracker";
import { NoopErrorTracker } from "./error-tracker";
import { SentryErrorTracker } from "./sentry-tracker";
import { ConsoleErrorTracker } from "./console-tracker";

/**
 * Error Tracker Context
 */
export const ErrorTrackerContext = createContext<ErrorTracker>(
  new NoopErrorTracker()
);

/**
 * Error Tracker Provider Props
 */
export interface ErrorTrackerProviderProps {
  children: React.ReactNode;
  /** Force specific tracker (for testing) */
  tracker?: ErrorTracker;
}

/**
 * Error Tracker Provider
 *
 * Provides error tracker instance to the application via React context.
 *
 * Automatically selects appropriate tracker based on environment:
 * - Sentry: If NEXT_PUBLIC_SENTRY_DSN is configured (via instrumentation.ts)
 * - Console: Development fallback or when Sentry is not available
 * - Noop: If error tracking is disabled
 *
 * Note: Sentry SDK is initialized globally via instrumentation.ts.
 * This provider just wraps the initialized SDK with our abstraction layer.
 *
 * @example
 * ```tsx
 * <ErrorTrackerProvider>
 *   <App />
 * </ErrorTrackerProvider>
 * ```
 */
export const ErrorTrackerProvider = ({
  children,
  tracker,
}: ErrorTrackerProviderProps) => {
  const errorTracker = useMemo(() => {
    // Use provided tracker (for testing)
    if (tracker) {
      return tracker;
    }

    const isProduction = env.isProduction;
    const isDevelopment = env.isDevelopment;
    const hasSentryDsn = isSentryConfigured();

    // Production with Sentry configured
    if (isProduction && hasSentryDsn) {
      try {
        return new SentryErrorTracker();
      } catch (error) {
        console.warn(
          "Failed to initialize Sentry error tracker, falling back to console:",
          error
        );
        return new ConsoleErrorTracker();
      }
    }

    // Development with Sentry configured (optional)
    if (isDevelopment && hasSentryDsn) {
      try {
        return new SentryErrorTracker();
      } catch (_error) {
        // Silent fallback to console in development
        return new ConsoleErrorTracker();
      }
    }

    // Development fallback
    if (isDevelopment) {
      return new ConsoleErrorTracker();
    }

    // Disabled error tracking
    return new NoopErrorTracker();
  }, [tracker]);

  return (
    <ErrorTrackerContext.Provider value={errorTracker}>
      {children}
    </ErrorTrackerContext.Provider>
  );
};
