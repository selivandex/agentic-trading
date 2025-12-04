/** @format */

import { logger } from "./src/shared/lib/logger";

/**
 * Next.js Instrumentation
 *
 * This file is automatically loaded by Next.js when the application starts.
 * It's used to initialize monitoring and observability tools like Sentry.
 *
 * The `register()` function is called once when the Next.js server starts,
 * for both the Node.js and Edge runtimes.
 *
 * @see https://nextjs.org/docs/app/building-your-application/optimizing/instrumentation
 * @see https://docs.sentry.io/platforms/javascript/guides/nextjs/manual-setup/#use-instrumentationts
 */

/**
 * Register instrumentation
 *
 * Called once when the Next.js server starts.
 * Initializes Sentry for the appropriate runtime.
 */
export async function register() {
  // Server runtime (Node.js)
  if (process.env.NEXT_RUNTIME === "nodejs") {
    logger.info("🚀 Initializing Sentry for Node.js runtime");
    await import("./sentry.server.config");
  }

  // Edge runtime (Middleware, Edge API routes)
  if (process.env.NEXT_RUNTIME === "edge") {
    logger.info("🚀 Initializing Sentry for Edge runtime");
    await import("./sentry.edge.config");
  }

  // Client-side initialization is handled by Next.js automatically
  // by importing sentry.client.config.ts
}

/**
 * Handle request errors
 *
 * Optional: Called when an unhandled error occurs during request processing.
 * Can be used for custom error handling or logging.
 *
 * @param error - The error that occurred
 * @param request - The incoming request
 * @param context - Additional context about the error
 */
export async function onRequestError(
  error: Error & { digest?: string },
  request: {
    path: string;
    method: string;
    headers: { [key: string]: string | string[] | undefined };
  },
  context: {
    routerKind: "Pages Router" | "App Router";
    routePath: string;
    routeType: "render" | "route" | "action" | "middleware";
    renderSource:
      | "react-server-components"
      | "react-server-components-payload"
      | "server-rendering";
    revalidateReason: "on-demand" | "stale" | undefined;
    renderType: "dynamic" | "dynamic-resume";
  }
) {
  // Log unhandled request errors
  // Sentry will automatically capture the error
  logger.error("❌ Unhandled request error:", {
    error: error.message,
    digest: error.digest,
    path: request.path,
    method: request.method,
    routePath: context.routePath,
    routeType: context.routeType,
    routerKind: context.routerKind,
    renderSource: context.renderSource,
  });
}
