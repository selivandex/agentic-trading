/** @format */

import { gql } from "@apollo/client";

/**
 * User GraphQL Queries and Mutations
 */

// Fragment for User fields
export const USER_FRAGMENT = gql`
  fragment UserFields on User {
    id
    telegramID
    telegramUsername
    email
    firstName
    lastName
    languageCode
    isActive
    isPremium
    limitProfileID
    settings {
      defaultAIProvider
      defaultAIModel
      riskLevel
      maxPositions
      maxPortfolioRisk
      maxDailyDrawdown
      maxConsecutiveLoss
      notificationsOn
      dailyReportTime
      timezone
      circuitBreakerOn
      maxPositionSizeUSD
      maxTotalExposureUSD
      minPositionSizeUSD
      maxLeverageMultiple
      allowedExchanges
    }
    createdAt
    updatedAt
  }
`;

// Get current user (me)
export const GET_ME_QUERY = gql`
  ${USER_FRAGMENT}
  query GetMe {
    me {
      ...UserFields
    }
  }
`;

// Get user by ID
export const GET_USER_QUERY = gql`
  ${USER_FRAGMENT}
  query GetUser($id: UUID!) {
    user(id: $id) {
      ...UserFields
    }
  }
`;

// Get user by telegram ID
export const GET_USER_BY_TELEGRAM_ID_QUERY = gql`
  ${USER_FRAGMENT}
  query GetUserByTelegramID($telegramID: String!) {
    userByTelegramID(telegramID: $telegramID) {
      ...UserFields
    }
  }
`;

// Update user settings
export const UPDATE_USER_SETTINGS_MUTATION = gql`
  ${USER_FRAGMENT}
  mutation UpdateUserSettings(
    $userID: UUID!
    $input: UpdateUserSettingsInput!
  ) {
    updateUserSettings(userID: $userID, input: $input) {
      ...UserFields
    }
  }
`;

// Set user active/inactive
export const SET_USER_ACTIVE_MUTATION = gql`
  ${USER_FRAGMENT}
  mutation SetUserActive($userID: UUID!, $isActive: Boolean!) {
    setUserActive(userID: $userID, isActive: $isActive) {
      ...UserFields
    }
  }
`;


