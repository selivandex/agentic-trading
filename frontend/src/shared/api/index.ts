// Apollo Client - Client-side only (browser)
export { 
  apolloClientBase, 
  getApolloClient, 
  clearCache, 
  initializeApolloClient 
} from "./apollo-client-base";

// Apollo Server Client - Server-side only (JWT callbacks, Server Components, API routes)
export { 
  createServerApolloClient, 
  serverQuery, 
  serverMutation 
} from "./apollo-server-client";

// Apollo Provider (wraps app with ApolloProvider)
export { ApolloProvider } from "./apollo-provider";

// Apollo Context Setter (deprecated - use RouteParamsProvider instead)
export { 
  ApolloContextSetter, 
  setApolloContext 
} from "./apollo-context-setter";

// Route Params Context (for Apollo Client headers)
export {
  RouteParamsProvider,
  useRouteParams,
  getCurrentRouteParams,
  setCurrentRouteParams,
} from "./route-params-context";

// Apollo Cache Configuration
export { createApolloCache, initializeCache, CACHE_VERSION } from "./apollo-cache-config";

// GraphQL Error Codes
export * from "./graphql-error-codes";

// Optimistic Updates
export * from "./optimistic-updates";

// GraphQL Hooks
export { useGraphQLQuery } from "./use-graphql-query";
export { useGraphQLMutation } from "./use-graphql-mutation";

// App Context GraphQL Query
export { GET_CURRENT_CONTEXT_QUERY } from "./app-context.graphql";

// Mutation with Toast Notifications
export {
  useMutationWithToast,
  createMutationWithToast,
  ErrorMessages,
  SuccessMessages,
  type MutationWithToast,
} from "./use-mutation-with-toast";

// Generated GraphQL types and hooks
export * from "./generated/graphql";

