/** @format */

/**
 * Environment Configuration
 *
 * Centralized type-safe access to environment variables.
 * All environment variables should be accessed through this module.
 *
 * @example
 * ```ts
 * import { env } from "@/shared/config";
 *
 * if (env.isDevelopment) {
 *   console.log('Dev mode');
 * }
 *
 * const client = new SentryClient(env.sentryDsn);
 * ```
 */
export const env = {
  // --------------------------
  // Sentry Error Tracking
  // --------------------------
  /** Sentry DSN for error tracking */
  sentryDsn: process.env.NEXT_PUBLIC_SENTRY_DSN,
  /** Sentry auth token for source maps upload */
  sentryAuthToken: process.env.SENTRY_AUTH_TOKEN,
  /** Sentry organization name */
  sentryOrg: process.env.SENTRY_ORG,
  /** Sentry project name */
  sentryProject: process.env.SENTRY_PROJECT,

  // --------------------------
  // Backend API
  // --------------------------
  /** Backend GraphQL API URL (server-side only) */
  backendGraphQLUrl:
    process.env.BACKEND_GRAPHQL_URL || "http://api.lvh.me:3000/graphql",

  // --------------------------
  // Authentication
  // --------------------------
  /** NextAuth secret for JWT signing */
  nextAuthSecret: process.env.NEXTAUTH_SECRET,
  /** NextAuth URL (base URL of the application) */
  nextAuthUrl: process.env.NEXTAUTH_URL || "http://localhost:3001",

  // --------------------------
  // Environment Detection
  // --------------------------
  /** Node environment (development, production, test) */
  nodeEnv: process.env.NODE_ENV,
  /** Is development environment */
  isDevelopment: process.env.NODE_ENV === "development",
  /** Is production environment */
  isProduction: process.env.NODE_ENV === "production",
  /** Is test environment */
  isTest: process.env.NODE_ENV === "test",

  // --------------------------
  // Feature Flags
  // --------------------------
  /** Enable Apollo Client debug logging */
  apolloDebugEnabled: process.env.NEXT_PUBLIC_APOLLO_DEBUG === "true",
} as const;

/**
 * Type-safe environment configuration
 */
export type Env = typeof env;

/**
 * Validate required environment variables
 *
 * Call this during application startup to ensure all required
 * environment variables are set.
 *
 * @throws {Error} If required environment variables are missing
 */
export const validateEnv = (): void => {
  const errors: string[] = [];

  // Required in production
  if (env.isProduction) {
    if (!env.nextAuthSecret) {
      errors.push("NEXTAUTH_SECRET is required in production");
    }
  }

  // Required for Sentry
  if (env.sentryAuthToken && (!env.sentryOrg || !env.sentryProject)) {
    errors.push(
      "SENTRY_ORG and SENTRY_PROJECT are required when SENTRY_AUTH_TOKEN is set"
    );
  }

  if (errors.length > 0) {
    throw new Error(
      `Environment validation failed:\n${errors.map((e) => `  - ${e}`).join("\n")}`
    );
  }
};

/**
 * Check if Sentry is configured
 */
export const isSentryConfigured = (): boolean => {
  return Boolean(env.sentryDsn);
};
