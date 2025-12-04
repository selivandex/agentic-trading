/** @format */

import { gql } from "@apollo/client";
import { USER_FRAGMENT } from "@/entities/user";

/**
 * Current User GraphQL Query
 */

// Get current authenticated user
export const GET_CURRENT_USER_QUERY = gql`
  ${USER_FRAGMENT}
  query GetCurrentUser {
    me {
      ...UserFields
    }
  }
`;
