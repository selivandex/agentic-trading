/** @format */

import * as Sentry from "@sentry/nextjs";
import { env } from "./src/shared/config";

/**
 * Sentry Client-Side Configuration
 *
 * Initializes Sentry for browser/client-side error tracking.
 * This file is automatically imported by Next.js instrumentation.
 *
 * Features enabled:
 * - Error tracking
 * - Performance monitoring
 * - Session replay (with privacy controls)
 * - Breadcrumbs (navigation, clicks, console)
 *
 * @see https://docs.sentry.io/platforms/javascript/guides/nextjs/
 */

const SENTRY_DSN = env.sentryDsn;
const NODE_ENV = env.nodeEnv;
const IS_PRODUCTION = env.isProduction;

// Only initialize if DSN is configured
if (SENTRY_DSN) {
  Sentry.init({
    // --------------------------
    // Core Configuration
    // --------------------------
    dsn: SENTRY_DSN,
    environment: NODE_ENV || "development",

    // Enable debug mode in development
    debug: !IS_PRODUCTION,

    // Enable/disable Sentry
    enabled: IS_PRODUCTION,

    // --------------------------
    // Performance Monitoring
    // --------------------------
    // Percentage of transactions to track for performance monitoring
    // 1.0 = 100%, 0.1 = 10%
    tracesSampleRate: IS_PRODUCTION ? 0.1 : 1.0,

    // --------------------------
    // Session Replay
    // --------------------------
    // Percentage of sessions to record
    replaysSessionSampleRate: IS_PRODUCTION ? 0.1 : 0,

    // Percentage of sessions with errors to record
    replaysOnErrorSampleRate: 1.0,

    // --------------------------
    // Integrations
    // --------------------------
    integrations: [
      // Session replay with privacy controls
      Sentry.replayIntegration({
        // Mask all text content for privacy
        maskAllText: true,
        // Block all media (images, videos) for privacy
        blockAllMedia: true,
        // Mask all inputs for security
        maskAllInputs: true,
      }),

      // Browser tracing for performance monitoring
      Sentry.browserTracingIntegration({
        // Track navigation timing
        enableInp: true,
        // Track long tasks
        enableLongTask: true,
        // Track interactions
        traceFetch: true,
        traceXHR: true,
      }),
    ],

    // --------------------------
    // Data Filtering
    // --------------------------
    // Filter out sensitive data before sending
    beforeSend(event, _hint) {
      // Don't send events in development (unless debug is explicitly enabled)
      if (!IS_PRODUCTION && !env.apolloDebugEnabled) {
        return null;
      }

      // Filter out known browser extension errors
      if (
        event.exception?.values?.[0]?.value?.includes("chrome-extension://")
      ) {
        return null;
      }

      // Filter out network errors (handled by Apollo)
      if (event.tags?.errorType === "network") {
        return null;
      }

      return event;
    },

    // --------------------------
    // Ignore Errors
    // --------------------------
    ignoreErrors: [
      // Browser extension errors
      "Non-Error promise rejection captured",
      "ResizeObserver loop limit exceeded",
      "ResizeObserver loop completed with undelivered notifications",

      // Network errors (handled separately)
      "NetworkError",
      "Failed to fetch",
      "Load failed",
      "Network request failed",

      // AbortController errors (expected)
      "AbortError",
      "The operation was aborted",

      // React errors (handled by error boundaries)
      "Minified React error",
    ],

    // --------------------------
    // Denylist for URLs
    // --------------------------
    denyUrls: [
      // Browser extensions
      /extensions\//i,
      /^chrome:\/\//i,
      /^moz-extension:\/\//i,
    ],
  });
}
