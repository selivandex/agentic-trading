/** @format */

import { gql } from "@apollo/client";

/**
 * GraphQL operations for Screen entity
 */

// Fragment for common screen fields
export const SCREEN_FRAGMENT = gql`
  fragment ScreenFields on Screen {
    id
    uid
    name
    status
    kind
    url
    appUsersCount
    scenariosCount
    macrosCount
    createdAt
    updatedAt
    previewImage {
      url
      thumbnailUrl
      width
      height
      mimeType
      size
    }
  }
`;

// Query: List screens for project
export const GET_SCREENS_QUERY = gql`
  query GetScreens(
    $first: Int
    $after: String
    $last: Int
    $before: String
  ) {
    screens(
      first: $first
      after: $after
      last: $last
      before: $before
    ) {
      edges {
        node {
          ...ScreenFields
        }
        cursor
      }
      pageInfo {
        hasNextPage
        hasPreviousPage
        startCursor
        endCursor
      }
    }
  }
  ${SCREEN_FRAGMENT}
`;

// Mutation: Create screen
export const CREATE_SCREEN_MUTATION = gql`
  mutation CreateScreen($input: CreateScreenInput!) {
    createScreen(input: $input) {
      ...ScreenFields
    }
  }
  ${SCREEN_FRAGMENT}
`;

// Mutation: Update screen
export const UPDATE_SCREEN_MUTATION = gql`
  mutation UpdateScreen($input: UpdateScreenInput!) {
    updateScreen(input: $input) {
      ...ScreenFields
    }
  }
  ${SCREEN_FRAGMENT}
`;

// Mutation: Delete screen
export const DELETE_SCREEN_MUTATION = gql`
  mutation DeleteScreen($input: DeleteScreenInput!) {
    deleteScreen(input: $input) {
      id
    }
  }
`;

// Mutation: Archive screen
export const ARCHIVE_SCREEN_MUTATION = gql`
  mutation ArchiveScreen($input: ArchiveScreenInput!) {
    archiveScreen(input: $input) {
      ...ScreenFields
    }
  }
  ${SCREEN_FRAGMENT}
`;

// Mutation: Unarchive screen
export const UNARCHIVE_SCREEN_MUTATION = gql`
  mutation UnarchiveScreen($input: UnarchiveScreenInput!) {
    unarchiveScreen(input: $input) {
      ...ScreenFields
    }
  }
  ${SCREEN_FRAGMENT}
`;

// Mutation: Publish screen
export const PUBLISH_SCREEN_MUTATION = gql`
  mutation PublishScreen($input: PublishScreenInput!) {
    publishScreen(input: $input) {
      ...ScreenFields
    }
  }
  ${SCREEN_FRAGMENT}
`;
