/** @format */

import { gql } from "@apollo/client";

/**
 * Agent GraphQL Queries and Mutations  
 * Auto-generated from table: agents
 */

// Fragment for Agent fields
export const AGENT_FRAGMENT = gql`
  fragment AgentFields on Agent {
    id
    identifier
    name
    description
    category
    systemPrompt
    instructions
    modelProvider
    modelName
    temperature
    maxTokens
    availableTools
    maxCostPerRun
    timeoutSeconds
    isActive
    version
    createdAt
    updatedAt
  }
`;

// Get Agent by ID
export const GET_AGENT_QUERY = gql`
  ${AGENT_FRAGMENT}
  query GetAgent($id: UUID!) {
    agent(id: $id) {
      ...AgentFields
    }
  }
`;

// Get all Agents
export const GET_ALL_AGENTS_QUERY = gql`
  ${AGENT_FRAGMENT}
  query GetAllAgents(
    $scope: String
    $search: String
    $filters: JSONObject
    $first: Int
    $after: String
    $last: Int
    $before: String
  ) {
    agents(
      scope: $scope
      search: $search
      filters: $filters
      first: $first
      after: $after
      last: $last
      before: $before
    ) {
      edges {
        node {
          ...AgentFields
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

// Create Agent
export const CREATE_AGENT_MUTATION = gql`
  ${AGENT_FRAGMENT}
  mutation CreateAgent($input: CreateAgentInput!) {
    createAgent(input: $input) {
      ...AgentFields
    }
  }
`;

// Update Agent
export const UPDATE_AGENT_MUTATION = gql`
  ${AGENT_FRAGMENT}
  mutation UpdateAgent($id: Int!, $input: UpdateAgentInput!) {
    updateAgent(id: $id, input: $input) {
      ...AgentFields
    }
  }
`;

// Delete Agent
export const DELETE_AGENT_MUTATION = gql`
  mutation DeleteAgent($id: Int!) {
    deleteAgent(id: $id)
  }
`;
