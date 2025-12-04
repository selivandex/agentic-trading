/** @format */

import { gql } from "@apollo/client";

/**
 * Current User GraphQL Queries
 */

// Get current user query with organizations and projects
export const GET_CURRENT_USER_QUERY = gql`
  query GetCurrentUser {
    currentUser {
      id
      email
      firstName
      lastName
      confirmedAt
      createdAt
      memberships {
        edges {
          node {
            id
            active
            role {
              id
              key
              name
            }
            organization {
              id
              name
              slug
              status
              projects {
                edges {
                  node {
                    id
                    name
                    slug
                  }
                }
              }
            }
          }
        }
      }
    }
  }
`;

