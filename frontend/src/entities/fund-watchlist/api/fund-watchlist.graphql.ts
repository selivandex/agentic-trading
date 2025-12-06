/** @format */

import { gql } from "@apollo/client";

/**
 * Fund Watchlist GraphQL Queries and Mutations
 */

// Fragment for FundWatchlist fields
export const FUND_WATCHLIST_FRAGMENT = gql`
  fragment FundWatchlistFields on FundWatchlist {
    id
    symbol
    marketType
    category
    tier
    isActive
    isPaused
    pausedReason
    lastAnalyzedAt
    createdAt
    updatedAt
  }
`;

// Get watchlist item by ID
export const GET_FUND_WATCHLIST_QUERY = gql`
  ${FUND_WATCHLIST_FRAGMENT}
  query GetFundWatchlist($id: UUID!) {
    fundWatchlist(id: $id) {
      ...FundWatchlistFields
    }
  }
`;

// Get watchlist item by symbol
export const GET_FUND_WATCHLIST_BY_SYMBOL_QUERY = gql`
  ${FUND_WATCHLIST_FRAGMENT}
  query GetFundWatchlistBySymbol($symbol: String!, $marketType: String!) {
    fundWatchlistBySymbol(symbol: $symbol, marketType: $marketType) {
      ...FundWatchlistFields
    }
  }
`;

// Get all watchlist items (legacy, use fundWatchlistsConnection)
export const GET_FUND_WATCHLISTS_QUERY = gql`
  ${FUND_WATCHLIST_FRAGMENT}
  query GetFundWatchlists(
    $limit: Int
    $offset: Int
    $isActive: Boolean
    $category: String
    $tier: Int
  ) {
    fundWatchlists(
      limit: $limit
      offset: $offset
      isActive: $isActive
      category: $category
      tier: $tier
    ) {
      ...FundWatchlistFields
    }
  }
`;

// Get all watchlist items with Relay Connection (for CRUD)
export const GET_FUND_WATCHLISTS_CONNECTION_QUERY = gql`
  ${FUND_WATCHLIST_FRAGMENT}
  query GetFundWatchlistsConnection(
    $scope: String
    $isActive: Boolean
    $category: String
    $tier: Int
    $search: String
    $filters: JSONObject
    $first: Int
    $after: String
    $last: Int
    $before: String
  ) {
    fundWatchlistsConnection(
      scope: $scope
      isActive: $isActive
      category: $category
      tier: $tier
      search: $search
      filters: $filters
      first: $first
      after: $after
      last: $last
      before: $before
    ) {
      edges {
        node {
          ...FundWatchlistFields
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

// Get monitored symbols
export const GET_MONITORED_SYMBOLS_QUERY = gql`
  ${FUND_WATCHLIST_FRAGMENT}
  query GetMonitoredSymbols($marketType: String) {
    monitoredSymbols(marketType: $marketType) {
      ...FundWatchlistFields
    }
  }
`;

// Create watchlist item
export const CREATE_FUND_WATCHLIST_MUTATION = gql`
  ${FUND_WATCHLIST_FRAGMENT}
  mutation CreateFundWatchlist($input: CreateFundWatchlistInput!) {
    createFundWatchlist(input: $input) {
      ...FundWatchlistFields
    }
  }
`;

// Update watchlist item
export const UPDATE_FUND_WATCHLIST_MUTATION = gql`
  ${FUND_WATCHLIST_FRAGMENT}
  mutation UpdateFundWatchlist(
    $id: UUID!
    $input: UpdateFundWatchlistInput!
  ) {
    updateFundWatchlist(id: $id, input: $input) {
      ...FundWatchlistFields
    }
  }
`;

// Delete watchlist item
export const DELETE_FUND_WATCHLIST_MUTATION = gql`
  mutation DeleteFundWatchlist($id: UUID!) {
    deleteFundWatchlist(id: $id)
  }
`;

// Toggle pause
export const TOGGLE_FUND_WATCHLIST_PAUSE_MUTATION = gql`
  ${FUND_WATCHLIST_FRAGMENT}
  mutation ToggleFundWatchlistPause(
    $id: UUID!
    $isPaused: Boolean!
    $reason: String
  ) {
    toggleFundWatchlistPause(id: $id, isPaused: $isPaused, reason: $reason) {
      ...FundWatchlistFields
    }
  }
`;

// Pause watchlist item
export const PAUSE_FUND_WATCHLIST_MUTATION = gql`
  ${FUND_WATCHLIST_FRAGMENT}
  mutation PauseFundWatchlist($id: UUID!, $reason: String) {
    toggleFundWatchlistPause(id: $id, isPaused: true, reason: $reason) {
      ...FundWatchlistFields
    }
  }
`;

// Resume watchlist item
export const RESUME_FUND_WATCHLIST_MUTATION = gql`
  ${FUND_WATCHLIST_FRAGMENT}
  mutation ResumeFundWatchlist($id: UUID!) {
    toggleFundWatchlistPause(id: $id, isPaused: false, reason: null) {
      ...FundWatchlistFields
    }
  }
`;
