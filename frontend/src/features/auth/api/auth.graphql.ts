/** @format */

import { gql } from "@apollo/client";

/**
 * Authentication GraphQL Mutations
 */

// Sign In mutation
export const SIGN_IN_MUTATION = gql`
  mutation SignIn($email: String!, $password: String!) {
    signIn(input: { email: $email, password: $password }) {
      accessToken
      expiresIn
      user {
        id
        email
        firstName
        lastName
        confirmedAt
        createdAt
      }
    }
  }
`;

// Sign Up mutation
export const SIGN_UP_MUTATION = gql`
  mutation SignUp(
    $email: String!
    $password: String!
    $firstName: String
    $lastName: String
  ) {
    signUp(
      input: {
        email: $email
        password: $password
        passwordConfirmation: $password
        firstName: $firstName
        lastName: $lastName
      }
    ) {
      accessToken
      expiresIn
      user {
        id
        email
        firstName
        lastName
        confirmedAt
        createdAt
      }
    }
  }
`;

// Forgot Password mutation
export const FORGOT_PASSWORD_MUTATION = gql`
  mutation ForgotPassword($email: String!) {
    forgotPassword(input: { email: $email }) {
      emailSent
    }
  }
`;

// Reset Password mutation
export const RESET_PASSWORD_MUTATION = gql`
  mutation ResetPassword(
    $password: String!
    $passwordConfirmation: String!
    $resetPasswordToken: String!
  ) {
    resetPassword(
      input: {
        password: $password
        passwordConfirmation: $passwordConfirmation
        resetPasswordToken: $resetPasswordToken
      }
    ) {
      id
      email
      firstName
      lastName
      confirmedAt
      createdAt
    }
  }
`;

// Sign Out mutation
export const SIGN_OUT_MUTATION = gql`
  mutation SignOut {
    logout {
      signedOut
    }
  }
`;





