/** @format */

import { z } from "zod";
import type { CrudConfig } from "@/shared/lib/crud";
import { Badge } from "@/components/base/badges/badges";
import type { Strategy } from "@/entities/strategy";
import {
  GET_ALL_STRATEGIES_QUERY,
  GET_STRATEGY_QUERY,
  CREATE_STRATEGY_MUTATION,
  UPDATE_STRATEGY_MUTATION,
} from "@/entities/strategy";
import { UserSelectField } from "@/entities/user";

/**
 * CRUD Configuration for Strategy entity
 *
 * @example
 * ```tsx
 * import { Crud } from '@/shared/ui/crud';
 * import { strategyCrudConfig } from '@/entities/strategy';
 *
 * function StrategiesPage() {
 *   return <Crud config={strategyCrudConfig} />;
 * }
 * ```
 */
export const strategyCrudConfig: CrudConfig<Strategy> = {
  // Resource names
  resourceName: "Strategy",
  resourceNamePlural: "Strategies",

  // Base path for navigation
  basePath: "/strategies",

  // GraphQL operations
  graphql: {
    list: {
      query: GET_ALL_STRATEGIES_QUERY,
      dataPath: "strategies",
      variables: {},
      useConnection: true, // Relay pagination
    },
    show: {
      query: GET_STRATEGY_QUERY,
      variables: { id: "" },
      dataPath: "strategy",
    },
    create: {
      mutation: CREATE_STRATEGY_MUTATION,
      variables: {},
      dataPath: "createStrategy",
    },
    update: {
      mutation: UPDATE_STRATEGY_MUTATION,
      variables: {},
      dataPath: "updateStrategy",
    },
    destroy: undefined, // No delete - use Close Strategy action instead
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
      key: "status",
      label: "Status",
      sortable: true,
      render: (strategy) => {
        const colorMap = {
          active: "success" as const,
          paused: "warning" as const,
          closed: "gray" as const,
        };
        return (
          <Badge color={colorMap[strategy.status] ?? "gray"} size="sm">
            {strategy.status.charAt(0).toUpperCase() + strategy.status.slice(1)}
          </Badge>
        );
      },
    },
    {
      key: "allocatedCapital",
      label: "Capital",
      sortable: true,
      render: (strategy) =>
        `$${Number(strategy.allocatedCapital).toLocaleString()}`,
    },
    {
      key: "totalPnL",
      label: "PnL",
      sortable: true,
      render: (strategy) => {
        const pnl = Number(strategy.totalPnL);
        const className = pnl >= 0 ? "text-success" : "text-error";
        return (
          <span className={className}>
            {pnl >= 0 ? "+" : ""}
            {pnl.toFixed(2)}%
          </span>
        );
      },
    },
    {
      key: "createdAt",
      label: "Created",
      sortable: true,
      render: (strategy) => new Date(strategy.createdAt).toLocaleDateString(),
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
      name: "userID",
      label: "User",
      type: "custom",
      helperText: "Select the user who owns this strategy",
      validation: z.string().uuid("Please select a user"),
      render: (props) => <UserSelectField {...props} />,
      colSpan: 12,
    },
    {
      name: "name",
      label: "Strategy Name",
      type: "text",
      placeholder: "e.g., Conservative Growth",
      helperText: "A descriptive name for your strategy",
      validation: z
        .string()
        .min(1, "Name is required")
        .max(100, "Name too long"),
      colSpan: 12,
    },
    {
      name: "description",
      label: "Description",
      type: "textarea",
      placeholder: "Describe the strategy's goals and approach...",
      helperText: "Optional description of your strategy",
      validation: z.string().min(1, "Description is required"),
      colSpan: 12,
    },
    {
      name: "allocatedCapital",
      label: "Allocated Capital ($)",
      type: "number",
      placeholder: "10000",
      helperText: "Amount to allocate to this strategy",
      validation: z
        .number()
        .min(100, "Minimum $100")
        .max(10000000, "Maximum $10,000,000"),
      colSpan: 6,
    },
    {
      name: "marketType",
      label: "Market Type",
      type: "select",
      options: [
        { label: "Spot", value: "spot" },
        { label: "Futures", value: "futures" },
      ],
      validation: z.enum(["spot", "futures"]),
      colSpan: 6,
    },
    {
      name: "riskTolerance",
      label: "Risk Tolerance",
      type: "select",
      options: [
        { label: "Conservative", value: "conservative" },
        { label: "Moderate", value: "moderate" },
        { label: "Aggressive", value: "aggressive" },
      ],
      validation: z.enum(["conservative", "moderate", "aggressive"]),
      colSpan: 6,
    },
    {
      name: "rebalanceFrequency",
      label: "Rebalance Frequency",
      type: "select",
      options: [
        { label: "Daily", value: "daily" },
        { label: "Weekly", value: "weekly" },
        { label: "Monthly", value: "monthly" },
        { label: "Never", value: "never" },
      ],
      validation: z.enum(["daily", "weekly", "monthly", "never"]),
      colSpan: 6,
    },
  ],

  // Custom actions (overridden in StrategyManager component with real mutations)
  actions: [],

  // Bulk actions (overridden in StrategyManager component with real mutations)
  bulkActions: [],

  // Feature flags
  enableSelection: true,
  enableSearch: true,
  defaultPageSize: 20,

  // Custom messages
  emptyStateMessage:
    "No strategies yet. Create your first strategy to get started with automated trading.",
  errorMessage: "Failed to load strategies. Please try again.",

  // Data transformations
  transformBeforeEdit: (strategy) => ({
    userID: strategy.userID,
    name: strategy.name,
    description: strategy.description,
    allocatedCapital: Number(strategy.allocatedCapital),
    marketType: strategy.marketType || "",
    riskTolerance: strategy.riskTolerance || "",
    rebalanceFrequency: strategy.rebalanceFrequency || "",
  }),

  transformBeforeCreate: (data) => ({
    userID: data.userID as string,
    name: data.name as string,
    description: data.description as string,
    allocatedCapital: String(data.allocatedCapital),
    marketType: data.marketType as string,
    riskTolerance: data.riskTolerance as string,
    rebalanceFrequency: data.rebalanceFrequency as string,
    targetAllocations: null,
  }),

  transformBeforeUpdate: (data) => ({
    name: data.name as string,
    description: data.description as string,
    riskTolerance: data.riskTolerance as string,
    rebalanceFrequency: data.rebalanceFrequency as string,
    targetAllocations: null,
  }),
};
