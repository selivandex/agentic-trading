/** @format */

import { z } from "zod";
import Link from "next/link";
import type { CrudConfig } from "@/shared/lib/crud";
import { Badge } from "@/components/base/badges/badges";
import type { Agent } from "@/entities/agent";
import {
  GET_ALL_AGENTS_QUERY,
  GET_AGENT_QUERY,
  CREATE_AGENT_MUTATION,
  UPDATE_AGENT_MUTATION,
  DELETE_AGENT_MUTATION,
} from "@/entities/agent";

/**
 * CRUD Configuration for Agent entity
 * Enhanced with comprehensive form fields and table columns
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
    destroy: {
      mutation: DELETE_AGENT_MUTATION,
      variables: { id: "" },
      dataPath: "deleteAgent",
    },
  },

  // Table columns
  columns: [
    {
      key: "identifier",
      label: "Identifier",
      sortable: true,
      width: "15%",
      render: (agent) => (
        <Link
          href={`/agents/${agent.id}`}
          className="font-mono text-sm text-primary hover:underline"
          onClick={(e) => e.stopPropagation()}
        >
          {agent.identifier}
        </Link>
      ),
    },
    {
      key: "category",
      label: "Category",
      sortable: true,
      render: (agent) => {
        const colorMap = {
          coordinator: "blue" as const,
          expert: "purple" as const,
          specialist: "success" as const,
        };
        const category = agent.category || "unknown";
        return (
          <Badge
            color={colorMap[category as keyof typeof colorMap] ?? "gray"}
            size="sm"
          >
            {category.charAt(0).toUpperCase() + category.slice(1)}
          </Badge>
        );
      },
    },
    {
      key: "modelProvider",
      label: "Provider",
      sortable: true,
      render: (agent) => {
        if (!agent.modelProvider) return "—";
        const colorMap = {
          openai: "blue" as const,
          anthropic: "purple" as const,
          google: "warning" as const,
          deepseek: "success" as const,
        };
        const provider = agent.modelProvider.toLowerCase();
        return (
          <Badge
            color={colorMap[provider as keyof typeof colorMap] ?? "gray"}
            size="sm"
          >
            {agent.modelProvider}
          </Badge>
        );
      },
      hideOnMobile: true,
    },
    {
      key: "modelName",
      label: "Model",
      sortable: true,
      render: (agent) => (
        <span className="font-mono text-xs">{agent.modelName || "—"}</span>
      ),
      hideOnMobile: true,
    },
    {
      key: "isActive",
      label: "Status",
      sortable: true,
      render: (agent) => (
        <Badge color={agent.isActive ? "success" : "gray"} size="sm">
          {agent.isActive ? "Active" : "Inactive"}
        </Badge>
      ),
    },
    {
      key: "version",
      label: "Version",
      sortable: true,
      render: (agent) => `v${agent.version || 1}`,
      hideOnMobile: true,
    },
    {
      key: "createdAt",
      label: "Created",
      sortable: true,
      render: (agent) => new Date(agent.createdAt).toLocaleDateString(),
      hideOnMobile: true,
    },
    {
      key: "actions",
      label: "",
      width: "10%",
    },
  ],

  // Form fields
  formFields: [
    {
      name: "identifier",
      label: "Identifier",
      type: "text",
      placeholder: "e.g., portfolio_manager",
      helperText: "Unique identifier for the agent (snake_case)",
      validation: z
        .string()
        .min(1, "Identifier is required")
        .max(100, "Identifier too long")
        .regex(/^[a-z_]+$/, "Use lowercase and underscores only"),
      colSpan: 6,
    },
    {
      name: "name",
      label: "Display Name",
      type: "text",
      placeholder: "e.g., Portfolio Manager",
      helperText: "Human-readable name for the agent",
      validation: z
        .string()
        .min(1, "Name is required")
        .max(100, "Name too long"),
      colSpan: 6,
    },
    {
      name: "description",
      label: "Description",
      type: "textarea",
      placeholder: "Describe the agent's purpose and capabilities...",
      helperText: "Optional description of what this agent does",
      validation: z.string().optional(),
      colSpan: 12,
    },
    {
      name: "category",
      label: "Category",
      type: "select",
      options: [
        { label: "Coordinator", value: "coordinator" },
        { label: "Expert", value: "expert" },
        { label: "Specialist", value: "specialist" },
      ],
      helperText: "Agent's role in the system",
      validation: z.enum(["coordinator", "expert", "specialist"]),
      colSpan: 4,
    },
    {
      name: "modelProvider",
      label: "Model Provider",
      type: "select",
      options: [
        { label: "OpenAI", value: "openai" },
        { label: "Anthropic", value: "anthropic" },
        { label: "Google", value: "google" },
        { label: "DeepSeek", value: "deepseek" },
      ],
      helperText: "AI provider for this agent",
      validation: z.enum(["openai", "anthropic", "google", "deepseek"]),
      colSpan: 4,
    },
    {
      name: "modelName",
      label: "Model Name",
      type: "text",
      placeholder: "e.g., gpt-4o",
      helperText: "Specific model to use",
      validation: z.string().min(1, "Model name is required"),
      colSpan: 4,
    },
    {
      name: "temperature",
      label: "Temperature",
      type: "number",
      placeholder: "0.7",
      helperText: "Randomness in responses (0.0 - 2.0)",
      validation: z.number().min(0).max(2).optional(),
      colSpan: 4,
    },
    {
      name: "maxTokens",
      label: "Max Tokens",
      type: "number",
      placeholder: "4096",
      helperText: "Maximum tokens per response",
      validation: z.number().min(1).max(200000).optional(),
      colSpan: 4,
    },
    {
      name: "timeoutSeconds",
      label: "Timeout (seconds)",
      type: "number",
      placeholder: "300",
      helperText: "Maximum execution time",
      validation: z.number().min(1).max(3600).optional(),
      colSpan: 4,
    },
    {
      name: "maxCostPerRun",
      label: "Max Cost Per Run ($)",
      type: "number",
      placeholder: "10.00",
      helperText: "Maximum cost allowed per execution",
      validation: z.number().min(0).max(1000).optional(),
      colSpan: 6,
    },
    {
      name: "isActive",
      label: "Active",
      type: "checkbox",
      helperText: "Whether this agent is currently active",
      validation: z.boolean().optional(),
      defaultValue: true,
      colSpan: 6,
    },
    {
      name: "systemPrompt",
      label: "System Prompt",
      type: "textarea",
      placeholder: "You are an expert portfolio manager...",
      helperText: "Base system prompt for the agent",
      validation: z.string().min(1, "System prompt is required"),
      colSpan: 12,
    },
    {
      name: "instructions",
      label: "Instructions",
      type: "textarea",
      placeholder: "Additional instructions...",
      helperText: "Optional detailed instructions",
      validation: z.string().optional(),
      colSpan: 12,
    },
  ],

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

  // Data transformations
  transformBeforeEdit: (agent) => ({
    identifier: agent.identifier,
    name: agent.name,
    description: agent.description || "",
    category: agent.category || "",
    systemPrompt: agent.systemPrompt,
    instructions: agent.instructions || "",
    modelProvider: agent.modelProvider || "",
    modelName: agent.modelName || "",
    temperature: agent.temperature ?? 0.7,
    maxTokens: agent.maxTokens ?? 4096,
    maxCostPerRun: agent.maxCostPerRun ?? 10,
    timeoutSeconds: agent.timeoutSeconds ?? 300,
    isActive: agent.isActive ?? true,
  }),

  transformBeforeCreate: (data) => ({
    identifier: data.identifier as string,
    name: data.name as string,
    description: data.description as string,
    category: data.category as string,
    systemPrompt: data.systemPrompt as string,
    instructions: data.instructions as string,
    modelProvider: data.modelProvider as string,
    modelName: data.modelName as string,
    temperature: data.temperature as number,
    maxTokens: data.maxTokens as number,
    maxCostPerRun: data.maxCostPerRun as number,
    timeoutSeconds: data.timeoutSeconds as number,
    isActive: data.isActive as boolean,
    availableTools: null,
  }),

  transformBeforeUpdate: (data) => ({
    identifier: data.identifier as string,
    name: data.name as string,
    description: data.description as string,
    category: data.category as string,
    systemPrompt: data.systemPrompt as string,
    instructions: data.instructions as string,
    modelProvider: data.modelProvider as string,
    modelName: data.modelName as string,
    temperature: data.temperature as number,
    maxTokens: data.maxTokens as number,
    maxCostPerRun: data.maxCostPerRun as number,
    timeoutSeconds: data.timeoutSeconds as number,
    isActive: data.isActive as boolean,
  }),
};
