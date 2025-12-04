/**
 * Server-Side Apollo Client Configuration
 *
 * Dedicated Apollo Client for server-side operations:
 * - Next-Auth JWT callbacks
 * - Server Components
 * - Server Actions
 * - API routes
 *
 * Key differences from client-side Apollo:
 * - NO batching (to avoid GET request conflicts)
 * - NO cache persistence
 * - Direct connection to Rails API (no Next.js proxy)
 * - Minimal cache (InMemoryCache without persistence)
 * - NO APQ (Automatic Persisted Queries)
 *
 * @format
 */

import { ApolloClient, createHttpLink, from, type OperationVariables } from "@apollo/client";
import { setContext } from "@apollo/client/link/context";
import { onError } from "@apollo/client/link/error";
import { createApolloCache } from "./apollo-cache-config";

// Backend GraphQL endpoint - direct connection (no proxy)
const BACKEND_GRAPHQL_URL =
  process.env.BACKEND_GRAPHQL_URL || "http://localhost:8080/graphql";

// Simple HTTP link - no batching for server-side requests
const httpLink = createHttpLink({
  uri: BACKEND_GRAPHQL_URL,
  credentials: "include",
});

// Auth link for server-side requests
// Expects accessToken to be passed via context
const authLink = setContext(async (_, { headers, accessToken, ...context }) => {
  const contextHeaders: Record<string, string> = {};

  // If accessToken is provided, add it as Authorization Bearer header
  if (accessToken) {
    contextHeaders["Authorization"] = `Bearer ${accessToken}`;
  }

  // Support for additional headers passed via context (takes precedence)
  if (headers) {
    Object.assign(contextHeaders, headers);
  }

  return {
    ...context,
    headers: {
      ...headers,
      ...contextHeaders,
    },
  };
});

// Error link for server-side logging
const errorLink = onError(({ graphQLErrors, networkError, operation }) => {
  if (graphQLErrors) {
    graphQLErrors.forEach(({ message, locations, path, extensions }) => {
      console.error(
        `[Server GraphQL error]: ${message}`,
        {
          operation: operation.operationName,
          path,
          locations,
          extensions,
        }
      );
    });
  }

  if (networkError) {
    console.error(
      `[Server Network error]: ${networkError}`,
      {
        operation: operation.operationName,
      }
    );
  }
});

/**
 * Create Server-Side Apollo Client
 *
 * Creates a new Apollo Client instance for each server-side request.
 * No singleton pattern - each request gets its own client.
 *
 * @example
 * ```ts
 * // In JWT callback
 * const client = createServerApolloClient();
 * const { data } = await client.query({
 *   query: GetCurrentUserDocument,
 *   context: {
 *     accessToken: token.accessToken, // Pass JWT token
 *   },
 * });
 * ```
 */
export const createServerApolloClient = () => {
  return new ApolloClient({
    link: from([
      errorLink,
      authLink,
      httpLink, // Simple HTTP link - no batching
    ]),
    cache: createApolloCache(), // Fresh cache per request
    ssrMode: true, // Enable SSR mode
    defaultOptions: {
      watchQuery: {
        errorPolicy: "all",
        fetchPolicy: "network-only", // Always fetch fresh data
      },
      query: {
        errorPolicy: "all",
        fetchPolicy: "network-only", // No cache for server-side
      },
      mutate: {
        errorPolicy: "all",
        fetchPolicy: "network-only",
      },
    },
    // Disable dev tools on server
    devtools: {
      enabled: true,
    },
  });
};

/**
 * Helper to execute GraphQL query on server
 *
 * @example
 * ```ts
 * const user = await serverQuery({
 *   query: GetCurrentUserDocument,
 *   accessToken: token.accessToken,
 * });
 * ```
 */
export const serverQuery = async <TData = unknown, TVariables extends OperationVariables = OperationVariables>({
  query,
  variables,
  accessToken,
}: {
  query: Parameters<ReturnType<typeof createServerApolloClient>["query"]>[0]["query"];
  variables?: TVariables;
  accessToken?: string;
}) => {
  const client = createServerApolloClient();

  const { data } = await client.query<TData, TVariables>({
    query,
    variables,
    context: {
      accessToken,
    },
  });

  return data;
};

/**
 * Helper to execute GraphQL mutation on server
 *
 * @example
 * ```ts
 * const result = await serverMutation({
 *   mutation: LoginDocument,
 *   variables: { email, password },
 * });
 * ```
 */
export const serverMutation = async <TData = unknown, TVariables extends OperationVariables = OperationVariables>({
  mutation,
  variables,
  accessToken,
}: {
  mutation: Parameters<ReturnType<typeof createServerApolloClient>["mutate"]>[0]["mutation"];
  variables?: TVariables;
  accessToken?: string;
}) => {
  const client = createServerApolloClient();

  const { data } = await client.mutate<TData, TVariables>({
    mutation,
    variables,
    context: {
      accessToken,
    },
  });

  return data;
};
