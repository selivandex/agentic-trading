/** @format */

import { gql } from "@apollo/client";

/**
 * Authentication GraphQL Mutations
 */

// Login mutation
export const LOGIN_MUTATION = gql`
  mutation Login($email: String!, $password: String!) {
    login(input: { email: $email, password: $password }) {
      token
      user {
        id
        telegramID
        telegramUsername
        email
        firstName
        lastName
        languageCode
        isActive
        isPremium
        createdAt
        updatedAt
      }
    }
  }
`;

// Logout mutation
export const LOGOUT_MUTATION = gql`
  mutation Logout {
    logout
  }
`;







