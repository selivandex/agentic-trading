/** @format */

import * as Sentry from "@sentry/nextjs";
import { env } from "./src/shared/config";

/**
 * Sentry Server-Side Configuration
 *
 * Initializes Sentry for server-side error tracking (Next.js API routes, SSR).
 * This file is automatically imported by Next.js instrumentation.
 *
 * Features enabled:
 * - Error tracking
 * - Performance monitoring
 * - Request context tracking
 *
 * @see https://docs.sentry.io/platforms/javascript/guides/nextjs/manual-setup/#server-side
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
    // Server-side: lower sample rate to reduce overhead
    tracesSampleRate: IS_PRODUCTION ? 0.05 : 1.0,

    // --------------------------
    // Data Filtering
    // --------------------------
    // Filter out sensitive data before sending
    beforeSend(event, _hint) {
      // Don't send events in development
      if (!IS_PRODUCTION) {
        return null;
      }

      // Filter GraphQL errors (handled by Apollo)
      if (event.tags?.source === "graphql") {
        return null;
      }

      // Sanitize request headers
      if (event.request?.headers) {
        const { authorization: _authorization, cookie: _cookie, ...safeHeaders } =
          event.request.headers;
        event.request.headers = safeHeaders;
      }

      return event;
    },

    // --------------------------
    // Ignore Errors
    // --------------------------
    ignoreErrors: [
      // Expected Next.js errors
      "ECONNRESET",
      "ETIMEDOUT",
      "ENOTFOUND",

      // GraphQL errors (handled separately)
      "GraphQLError",

      // AbortController errors (expected)
      "AbortError",
      "The operation was aborted",
    ],
  });
}
