/** @format */

import { gql } from "@apollo/client";
import { USER_FRAGMENT } from "@/entities/user";

/**
 * Strategy GraphQL Queries and Mutations
 */

// Fragment for Strategy fields
export const STRATEGY_FRAGMENT = gql`
  fragment StrategyFields on Strategy {
    id
    userID
    name
    description
    status
    allocatedCapital
    currentEquity
    cashReserve
    marketType
    riskTolerance
    rebalanceFrequency
    targetAllocations
    totalPnL
    totalPnLPercent
    sharpeRatio
    maxDrawdown
    winRate
    createdAt
    updatedAt
    closedAt
    lastRebalancedAt
    reasoningLog
  }
`;

// Fragment with user relation
export const STRATEGY_WITH_USER_FRAGMENT = gql`
  ${STRATEGY_FRAGMENT}
  ${USER_FRAGMENT}
  fragment StrategyWithUserFields on Strategy {
    ...StrategyFields
    user {
      ...UserFields
    }
  }
`;

// Get strategy by ID
export const GET_STRATEGY_QUERY = gql`
  ${STRATEGY_WITH_USER_FRAGMENT}
  query GetStrategy($id: UUID!) {
    strategy(id: $id) {
      ...StrategyWithUserFields
    }
  }
`;

// Get user strategies
export const GET_USER_STRATEGIES_QUERY = gql`
  ${STRATEGY_FRAGMENT}
  query GetUserStrategies(
    $userID: UUID!
    $scope: String
    $status: StrategyStatus
    $search: String
    $first: Int
    $after: String
    $last: Int
    $before: String
  ) {
    userStrategies(
      userID: $userID
      scope: $scope
      status: $status
      search: $search
      first: $first
      after: $after
      last: $last
      before: $before
    ) {
      edges {
        node {
          ...StrategyFields
        }
        cursor
      }
      pageInfo {
        hasNextPage
        hasPreviousPage
        startCursor
        endCursor
      }
      totalCount
      scopes {
        id
        name
        count
      }
      filters {
        id
        name
        type
        options {
          value
          label
        }
        defaultValue
        placeholder
        min
        max
      }
    }
  }
`;

// Get all strategies
export const GET_ALL_STRATEGIES_QUERY = gql`
  ${STRATEGY_WITH_USER_FRAGMENT}
  query GetAllStrategies(
    $scope: String
    $status: StrategyStatus
    $search: String
    $first: Int
    $after: String
    $last: Int
    $before: String
  ) {
    strategies(
      scope: $scope
      status: $status
      search: $search
      first: $first
      after: $after
      last: $last
      before: $before
    ) {
      edges {
        node {
          ...StrategyWithUserFields
        }
        cursor
      }
      pageInfo {
        hasNextPage
        hasPreviousPage
        startCursor
        endCursor
      }
      totalCount
      scopes {
        id
        name
        count
      }
      filters {
        id
        name
        type
        options {
          value
          label
        }
        defaultValue
        placeholder
        min
        max
      }
    }
  }
`;

// Create strategy
export const CREATE_STRATEGY_MUTATION = gql`
  ${STRATEGY_FRAGMENT}
  mutation CreateStrategy($userID: UUID!, $input: CreateStrategyInput!) {
    createStrategy(userID: $userID, input: $input) {
      ...StrategyFields
    }
  }
`;

// Update strategy
export const UPDATE_STRATEGY_MUTATION = gql`
  ${STRATEGY_FRAGMENT}
  mutation UpdateStrategy($id: UUID!, $input: UpdateStrategyInput!) {
    updateStrategy(id: $id, input: $input) {
      ...StrategyFields
    }
  }
`;

// Pause strategy
export const PAUSE_STRATEGY_MUTATION = gql`
  ${STRATEGY_FRAGMENT}
  mutation PauseStrategy($id: UUID!) {
    pauseStrategy(id: $id) {
      ...StrategyFields
    }
  }
`;

// Resume strategy
export const RESUME_STRATEGY_MUTATION = gql`
  ${STRATEGY_FRAGMENT}
  mutation ResumeStrategy($id: UUID!) {
    resumeStrategy(id: $id) {
      ...StrategyFields
    }
  }
`;

// Close strategy
export const CLOSE_STRATEGY_MUTATION = gql`
  ${STRATEGY_FRAGMENT}
  mutation CloseStrategy($id: UUID!) {
    closeStrategy(id: $id) {
      ...StrategyFields
    }
  }
`;

// Delete strategy (hard delete, only for closed strategies)
export const DELETE_STRATEGY_MUTATION = gql`
  mutation DeleteStrategy($id: UUID!) {
    deleteStrategy(id: $id)
  }
`;

// Batch delete strategies (hard delete, only for closed strategies)
export const BATCH_DELETE_STRATEGIES_MUTATION = gql`
  mutation BatchDeleteStrategies($ids: [UUID!]!) {
    batchDeleteStrategies(ids: $ids)
  }
`;

// Batch pause strategies
export const BATCH_PAUSE_STRATEGIES_MUTATION = gql`
  mutation BatchPauseStrategies($ids: [UUID!]!) {
    batchPauseStrategies(ids: $ids)
  }
`;

// Batch resume strategies
export const BATCH_RESUME_STRATEGIES_MUTATION = gql`
  mutation BatchResumeStrategies($ids: [UUID!]!) {
    batchResumeStrategies(ids: $ids)
  }
`;

// Batch close strategies
export const BATCH_CLOSE_STRATEGIES_MUTATION = gql`
  mutation BatchCloseStrategies($ids: [UUID!]!) {
    batchCloseStrategies(ids: $ids)
  }
`;
