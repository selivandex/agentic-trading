/** @format */

import { z } from "zod";
import type { CrudConfig } from "@/shared/lib/crud";
import { Badge } from "@/components/base/badges/badges";
import type { Agent } from "@/entities/agent";
import {
  GET_ALL_AGENTS_QUERY,
  GET_AGENT_QUERY,
  CREATE_AGENT_MUTATION,
  UPDATE_AGENT_MUTATION,
} from "@/entities/agent";

/**
 * CRUD Configuration for Agent entity
 * Auto-generated from table: agents
 */
export const agentCrudConfig: CrudConfig<Agent> = {
  // Resource names
  resourceName: "Agent",
  resourceNamePlural: "Agents",

  // Base path for navigation
  basePath: "/agents",

  // Display name for breadcrumbs and titles
  getDisplayName: (entity) => entity.name,

  // GraphQL operations
  graphql: {
    list: {
      query: GET_ALL_AGENTS_QUERY,
      dataPath: "agents",
      variables: {},
      useConnection: true,
    },
    show: {
      query: GET_AGENT_QUERY,
      variables: { id: "" },
      dataPath: "agent",
    },
    create: {
      mutation: CREATE_AGENT_MUTATION,
      variables: {},
      dataPath: "createAgent",
    },
    update: {
      mutation: UPDATE_AGENT_MUTATION,
      variables: {},
      dataPath: "updateAgent",
    },
    destroy: undefined,
  },

  // Table columns
  columns: [
    {
      key: "name",
      label: "Name",
      sortable: true,
      width: "25%",
    },
    {
      key: "actions",
      label: "",
      width: "10%",
    },
  ],

  // Form fields
  formFields: [],

  // Custom actions (will be overridden in Manager)
  actions: [],
  bulkActions: [],

  // Tabs configuration
  tabs: {
    enabled: true,
    type: "underline",
    size: "md",
    filterVariable: "scope",
  },

  // Dynamic filters configuration
  dynamicFilters: {
    enabled: true,
  },

  // Feature flags
  enableSelection: true,
  enableSearch: true,
  defaultPageSize: 20,

  // Custom messages
  emptyStateMessage: "No agents yet. Create your first agent to get started.",
  errorMessage: "Failed to load agents. Please try again.",
};
