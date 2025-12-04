/**
 * Apollo Provider with Dynamic Context Headers
 *
 * Provides Apollo Client with dynamic organization/project context.
 * Features:
 * - Single Apollo Client instance with cache persistence
 * - Batch queries for performance
 * - Automatic Persisted Queries (APQ)
 * - Lazy initialization on first render
 *
 * @format
 */

"use client";

import { type ReactNode, useState } from "react";
import {
  ApolloProvider as ApolloClientProvider,
  ApolloClient,
  NormalizedCacheObject,
} from "@apollo/client";
import { getApolloClient } from "./apollo-client-base";

interface ApolloProviderProps {
  children: ReactNode;
}

/**
 * Apollo Provider with lazy initialization
 *
 * Initializes Apollo Client with in-memory cache on first render.
 * No loading state needed - initialization is synchronous and instant.
 */
export const ApolloProvider = ({ children }: ApolloProviderProps) => {
  const [client] = useState<ApolloClient<NormalizedCacheObject>>(() => {
    // Initialize Apollo Client synchronously on mount
    return getApolloClient();
  });

  return (
    <ApolloClientProvider client={client}>{children}</ApolloClientProvider>
  );
};
