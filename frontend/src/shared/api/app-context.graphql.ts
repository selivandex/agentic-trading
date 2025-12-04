/** @format */

import { gql } from "@apollo/client";

/**
 * GraphQL query to get current application context
 *
 * Simple health check query for now
 */
export const GET_CURRENT_CONTEXT_QUERY = gql`
  query GetCurrentContext {
    health
  }
`;
