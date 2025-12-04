/**
 * Client-Side Apollo Client Configuration
 *
 * Dedicated Apollo Client for client-side operations (React components).
 * This should ONLY be used in browser environment.
 *
 * For server-side operations (JWT callbacks, Server Components, API routes),
 * use `apollo-server-client.ts` instead.
 *
 * Features:
 * - Batch Link: Combines multiple queries into single HTTP request
 * - Cache Persistence: Offline-first with localStorage
 * - Smart Retry Logic: Only retry idempotent operations
 * - Dynamic Context Headers: Organization/Project injection
 *
 * @format
 */

import { ApolloClient, createHttpLink, from, split, NormalizedCacheObject } from "@apollo/client";
import { setContext } from "@apollo/client/link/context";
import { onError } from "@apollo/client/link/error";
import { RetryLink } from "@apollo/client/link/retry";
import { BatchHttpLink } from "@apollo/client/link/batch-http";
import { createApolloCache, initializeCache } from "./apollo-cache-config";
import { performanceLink } from "./apollo-performance-link";
import {
  isContextError,
  isAuthError,
  isForbiddenError,
} from "./graphql-error-codes";
import { getCurrentRouteParams } from "./route-params-context";

// GraphQL endpoint URL
// Client-side ONLY: Always use Next.js proxy for proper auth flow
const GRAPHQL_ENDPOINT = "/api/graphql";

// Single HTTP link for mutations (not batched)
const httpLink = createHttpLink({
  uri: GRAPHQL_ENDPOINT,
  credentials: "include", // Include cookies for session-based auth
});

// Batch HTTP link for queries (combines multiple queries)
const batchHttpLink = new BatchHttpLink({
  uri: GRAPHQL_ENDPOINT,
  credentials: "include",
  batchMax: 10, // Max 10 operations per batch
  batchInterval: 50, // Wait 50ms before sending batch (balances latency vs batching)
  // Debug batching in development (disabled - too noisy)
  // ...(process.env.NODE_ENV === "development" && {
  //   fetch: async (uri, options) => {
  //     console.log("🔵 Batch request:", {
  //       uri,
  //       body: JSON.parse(options?.body as string || "[]"),
  //     });
  //
  //     const response = await fetch(uri, options);
  //     const data = await response.clone().json();
  //
  //     console.log("🟢 Batch response:", data);
  //
  //     return response;
  //   },
  // }),
});

// Split: mutations use single HTTP, queries use batch HTTP
// Client-side only: batch queries for better performance
const httpLinkRouter = split(
  (operation) => {
    const definition = operation.query.definitions[0];
    return (
      definition.kind === "OperationDefinition" &&
      definition.operation === "mutation"
    );
  },
  httpLink, // Mutations - single request
  batchHttpLink // Queries - batched for better performance
);

// Auth link with dynamic context headers
// JWT is handled by /api/graphql proxy via httpOnly cookies
// Organization/Project headers are provided by RouteParamsProvider
const authLink = setContext((operation, { headers, ...context }) => {
  const contextHeaders: Record<string, string> = {};

  // Get organization_id and project_id from route params context
  // These are set by RouteParamsProvider in project layout
  if (typeof window !== "undefined") {
    const { organizationId, projectId } = getCurrentRouteParams();

    // Add headers if both params are present
    if (organizationId && projectId) {
      // Organization/Project headers not needed for trading platform

      // Debug: Log headers in development
      if (process.env.NODE_ENV === "development") {
        console.log(`[Apollo] Setting headers for ${operation.operationName}:`, {
          organization: organizationId,
          project: projectId,
        });
      }
    } else if (process.env.NODE_ENV === "development") {
      // Warn if headers are missing in development
      console.warn(`[Apollo] Missing route params for ${operation.operationName}:`, {
        organizationId,
        projectId,
      });
    }
  }

  // Support for dynamic headers passed via query context (takes precedence)
  // Example: client.query({ query: MY_QUERY, context: { headers: { 'X-Custom-Header': 'value' } } })
  if (context.headers) {
    Object.assign(contextHeaders, context.headers);
  }

  return {
    headers: {
      ...headers,
      ...contextHeaders,
    },
    // Support for custom URI override (used for auth operations)
    uri: context.uri || undefined,
    // Support for custom fetch options
    fetchOptions: context.fetchOptions || {},
  };
});

// Prevent multiple simultaneous redirects
let redirectInProgress = false;

/**
 * Centralized error handler
 * Redirects to appropriate error page based on error code
 */
const handleErrorRedirect = (
  errorCode: string,
  message: string,
  redirectUrl: string
) => {
  if (typeof window === "undefined" || redirectInProgress) {
    return;
  }

  redirectInProgress = true;
  console.warn(`[Apollo] ${errorCode} - redirecting to ${redirectUrl}`);

  // Clear Apollo cache on auth errors
  if (isAuthError(errorCode)) {
    localStorage.removeItem("apollo-cache-persist");
    localStorage.removeItem("apollo-cache-version");
  }

  // Try soft redirect first (Next.js router), fallback to hard redirect
  import("next/navigation")
    .then(({ redirect }) => {
      try {
        redirect(redirectUrl);
      } catch (error) {
        // If redirect() throws (React error), fallback to hard redirect
        console.error("Soft redirect failed, using hard redirect:", error);
        window.location.href = redirectUrl;
      }
    })
    .catch(() => {
      // If import fails, fallback to hard redirect
      window.location.href = redirectUrl;
    })
    .finally(() => {
      // Reset redirect flag after delay
      setTimeout(() => {
        redirectInProgress = false;
      }, 1000);
    });
};

// Error link for handling GraphQL errors
const errorLink = onError(({ graphQLErrors, networkError, operation }) => {
  if (graphQLErrors) {
    graphQLErrors.forEach(({ message, locations, path, extensions }) => {
      const errorCode = extensions?.code as string | undefined;

      // Log error for debugging
      console.error(
        `[GraphQL error]: Message: ${message}, Location: ${JSON.stringify(locations)}, Path: ${path}`,
        {
          code: errorCode,
          extensions,
        }
      );

      // Skip redirect if no error code
      if (!errorCode) {
        return;
      }

      // Handle authentication errors (401 Unauthorized)
      // Clear Apollo cache and redirect to login
      // NextAuth JWT callback will invalidate session (see auth.ts)
      if (isAuthError(errorCode)) {
        handleErrorRedirect(errorCode, message, "/login");
        return;
      }

      // Handle forbidden errors (403 Forbidden)
      if (isForbiddenError(errorCode)) {
        // Try to extract org and project from current URL
        const pathMatch = window.location.pathname.match(/^\/([^/]+)\/([^/]+)/);

        const redirectUrl = pathMatch
          ? `/${pathMatch[1]}/${pathMatch[2]}/forbidden`
          : "/forbidden";

        handleErrorRedirect(errorCode, message, redirectUrl);
        return;
      }

      // Handle context errors (organization/project not found)
      if (isContextError(errorCode)) {
        // For context errors, don't redirect - let the component handle it
        // This allows showing ProjectNotFound component with proper error message
        console.warn(
          `[Apollo] Context error (${errorCode}): ${message} - component will handle display`
        );
        // Error will bubble up to component via useQuery error state
        return;
      }

      // All other errors are logged but not redirected
      // Components can handle them via error state from useQuery/useMutation
    });
  }

  if (networkError) {
    console.error(`[Network error]: ${networkError}`, {
      operation: operation.operationName,
    });
  }
});

// Idempotent mutations that are safe to retry
// These mutations can be retried on network errors without side effects
const RETRYABLE_MUTATIONS = [
  "ActivateScenario",
  "PauseScenario",
  "ArchiveScenario",
  "UpdateScenario",
  "UpdateProject",
  "UpdateOrganization",
  "UpdateScreen",
  "UpdateLanguage",
  "UpdateMacro",
];

// Retry link for network errors
const retryLink = new RetryLink({
  delay: {
    initial: 300,
    max: 3000,
    jitter: true,
  },
  attempts: {
    max: 3,
    retryIf: (error, operation) => {
      // Only retry on network errors (not GraphQL errors)
      if (!error || error.result) {
        return false;
      }

      // For mutations, only retry idempotent ones
      const isMutation = operation.query.definitions.some(
        (def) => "operation" in def && def.operation === "mutation"
      );

      if (isMutation) {
        const operationName = operation.operationName;
        return operationName ? RETRYABLE_MUTATIONS.includes(operationName) : false;
      }

      // Always retry queries on network errors
      return true;
    },
  },
});

// Automatic Persisted Queries (APQ) - DISABLED
// APQ requires server-side support which Rails GraphQL API doesn't have
// Enabling APQ without server support causes "No query string was present" errors
//
// To enable APQ in the future:
// 1. Add APQ support to Rails GraphQL (apollo-link-persisted-queries gem)
// 2. Uncomment the code below
// 3. Ensure useGETForHashedQueries is false (BatchHttpLink doesn't support GET)
//
// const persistedQueriesLink = isCryptoAvailable()
//   ? createPersistedQueryLink({
//       sha256,
//       useGETForHashedQueries: false,
//     })
//   : null;

const persistedQueriesLink = null; // APQ disabled - server doesn't support it

/**
 * Create Apollo Client instance
 *
 * Creates cache with localStorage persistence for instant UI.
 * Cache is automatically saved after each operation.
 */
const createApolloClient = () => {
  // Initialize cache with localStorage persistence
  const cache = typeof window !== "undefined"
    ? initializeCache()
    : createApolloCache();

  // Build link chain
  const links = [
    performanceLink,
    errorLink,
    retryLink,
    // APQ link would go here (currently disabled - see persistedQueriesLink above)
    ...(persistedQueriesLink ? [persistedQueriesLink] : []),
    authLink,
    httpLinkRouter, // Batch queries, single mutations
  ];

  const client = new ApolloClient({
    link: from(links),
    cache,
    defaultOptions: {
      watchQuery: {
        errorPolicy: "all",
        fetchPolicy: "cache-and-network",
      },
      query: {
        errorPolicy: "all",
        fetchPolicy: "cache-first",
      },
      mutate: {
        errorPolicy: "all",
        fetchPolicy: "network-only", // Always hit network for mutations
      },
    },
    devtools: {
      enabled: process.env.NODE_ENV === "development",
    },
  });

  // Auto-save cache to localStorage after each operation (client-side only)
  if (typeof window !== "undefined") {
    client.onResetStore(() => {
      // Clear localStorage on cache reset (data + version)
      localStorage.removeItem("apollo-cache-persist");
      localStorage.removeItem("apollo-cache-version");
      return Promise.resolve();
    });

    // Debounced save to localStorage
    let saveTimeout: ReturnType<typeof setTimeout> | null = null;
    const debouncedSave = () => {
      if (saveTimeout) {
        clearTimeout(saveTimeout);
      }
      saveTimeout = setTimeout(() => {
        try {
          const data = JSON.stringify(cache.extract());
          localStorage.setItem("apollo-cache-persist", data);
        } catch (error) {
          console.error("Failed to persist Apollo cache:", error);
        }
      }, 1000); // Save after 1 second of inactivity
    };

    // Save cache after query/mutation completion
    // Apollo doesn't have a direct "watch" API, so we use a simple approach:
    // Save on every request completion (debounced to avoid performance issues)
    const saveToStorage = () => debouncedSave();

    // Hook into cache writes by wrapping query/mutate methods
    const originalQuery = client.query.bind(client);
    const originalMutate = client.mutate.bind(client);

    client.query = async (options) => {
      const result = await originalQuery(options);
      saveToStorage();
      return result;
    };

    client.mutate = async (options) => {
      const result = await originalMutate(options);
      saveToStorage();
      return result;
    };
  }

  return client;
};

// Singleton Apollo Client instance
let apolloClientInstance: ApolloClient<NormalizedCacheObject> | null = null;

/**
 * Get or create Apollo Client instance (CLIENT-SIDE ONLY)
 *
 * Lazy initialization pattern - creates client on first access.
 * Returns singleton instance.
 *
 * WARNING: Do NOT use this on server-side (JWT callbacks, Server Components).
 * Use `apollo-server-client.ts` instead for server-side operations.
 */
export const getApolloClient = () => {
  // Client-side: use singleton
  if (!apolloClientInstance) {
    apolloClientInstance = createApolloClient();
  }

  return apolloClientInstance;
};

// Synchronous access to Apollo Client (for compatibility)
// This will be null on first render until initialized
export const apolloClientBase = apolloClientInstance!;

/**
 * Clear Apollo cache and localStorage
 * Use this after logout to prevent stale data
 */
export const clearCache = async () => {
  const client = getApolloClient();

  // Clear localStorage first (cache data + version)
  if (typeof window !== "undefined") {
    localStorage.removeItem("apollo-cache-persist");
    localStorage.removeItem("apollo-cache-version");
  }

  // Then clear in-memory cache
  await client.clearStore();

  if (process.env.NODE_ENV === "development") {
    console.log("✅ Apollo cache cleared (memory + localStorage + version)");
  }
};

/**
 * Initialize Apollo Client on app startup
 * Call this once in root layout (client-side only)
 */
export const initializeApolloClient = () => {
  if (typeof window !== "undefined") {
    apolloClientInstance = createApolloClient();

    if (process.env.NODE_ENV === "development") {
      console.log("✅ Apollo Client initialized (in-memory cache)");
    }
  }
};
