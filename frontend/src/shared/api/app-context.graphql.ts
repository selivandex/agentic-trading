/** @format */

import { gql } from "@apollo/client";

/**
 * GraphQL query to get current application context
 * 
 * Returns current organization with its projects based on headers:
 * - X-Flowly-Organization
 * - X-Flowly-Project
 * 
 * This query is used once per page load to initialize app context.
 */
export const GET_CURRENT_CONTEXT_QUERY = gql`
  query GetCurrentContext($first: Int) {
    organization {
      id
      name
      slug
      timezone
      createdAt
      updatedAt
      projects(first: $first) {
        edges {
          node {
            id
            name
            slug
            createdAt
            updatedAt
          }
        }
      }
    }
  }
`;

