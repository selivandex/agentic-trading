import {
  useQuery,
  type QueryHookOptions,
  type OperationVariables,
} from "@apollo/client";
import type { DocumentNode } from "graphql";

/**
 * Hook for executing GraphQL queries with Apollo Client
 * This is a wrapper around Apollo's useQuery for convenience
 *
 * @example
 * ```ts
 * import { gql } from "@apollo/client";
 *
 * const GET_ORGANIZATIONS = gql`
 *   query GetOrganizations {
 *     organizations {
 *       id
 *       name
 *       slug
 *     }
 *   }
 * `;
 *
 * const { data, loading, error } = useGraphQLQuery(GET_ORGANIZATIONS);
 * ```
 */
export const useGraphQLQuery = <
  TData = unknown,
  TVariables extends OperationVariables = OperationVariables,
>(
  query: DocumentNode,
  options?: Omit<QueryHookOptions<TData, TVariables>, "query">
) => {
  return useQuery<TData, TVariables>(query, {
    ...options,
  });
};
