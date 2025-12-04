import {
  useMutation,
  type MutationHookOptions,
  type MutationResult,
} from "@apollo/client";
import type { DocumentNode } from "graphql";

/**
 * Hook for executing GraphQL mutations with Apollo Client
 * This is a wrapper around Apollo's useMutation for convenience
 *
 * @example
 * ```ts
 * import { gql } from "@apollo/client";
 *
 * const CREATE_ORGANIZATION = gql`
 *   mutation CreateOrganization($name: String!, $slug: String!) {
 *     createOrganization(name: $name, slug: $slug) {
 *       id
 *       name
 *       slug
 *     }
 *   }
 * `;
 *
 * const [createOrganization, { loading, error }] = useGraphQLMutation(
 *   CREATE_ORGANIZATION,
 *   {
 *     refetchQueries: [{ query: GET_ORGANIZATIONS }],
 *   }
 * );
 *
 * // Usage
 * createOrganization({ variables: { name: "New Org", slug: "new-org" } });
 * ```
 */
export const useGraphQLMutation = <TData = unknown, TVariables = unknown>(
  mutation: DocumentNode,
  options?: Omit<MutationHookOptions<TData, TVariables>, "mutation">
): [typeof mutate, MutationResult<TData>] => {
  const [mutate, result] = useMutation<TData, TVariables>(mutation, options);
  return [mutate, result];
};
