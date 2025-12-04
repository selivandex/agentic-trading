/** @format */

import * as Sentry from "@sentry/nextjs";
import { env } from "./src/shared/config";

/**
 * Sentry Edge Runtime Configuration
 *
 * Initializes Sentry for Edge Runtime (middleware, edge API routes).
 * This file is automatically imported by Next.js instrumentation.
 *
 * Note: Edge runtime has limited APIs compared to Node.js.
 *
 * @see https://docs.sentry.io/platforms/javascript/guides/nextjs/manual-setup/#edge-runtime
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
    // Edge runtime: minimal tracing
    tracesSampleRate: IS_PRODUCTION ? 0.05 : 1.0,

    // --------------------------
    // Data Filtering
    // --------------------------
    beforeSend(event) {
      // Don't send events in development
      if (!IS_PRODUCTION) {
        return null;
      }

      return event;
    },
  });
}
