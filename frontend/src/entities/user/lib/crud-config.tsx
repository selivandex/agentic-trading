/** @format */

import { z } from "zod";
import type { CrudConfig } from "@/shared/lib/crud";
import { Badge } from "@/components/base/badges/badges";
import type { User } from "../model/types";
import {
  GET_USERS_QUERY,
  GET_USER_QUERY,
  UPDATE_USER_SETTINGS_MUTATION,
  SET_USER_ACTIVE_MUTATION,
} from "../api/user.graphql";

/**
 * CRUD Configuration for User entity
 *
 * Note: Users don't have create/delete operations
 * - Users are created through Telegram/OAuth
 * - Users can only be activated/deactivated, not deleted
 *
 * @example
 * ```tsx
 * import { Crud } from '@/shared/ui/crud';
 * import { userCrudConfig } from '@/entities/user';
 *
 * function UsersPage() {
 *   return <Crud config={userCrudConfig} />;
 * }
 * ```
 */
export const userCrudConfig: CrudConfig<User> = {
  // Resource names
  resourceName: "User",
  resourceNamePlural: "Users",

  // GraphQL operations
  graphql: {
    list: {
      query: GET_USERS_QUERY,
      dataPath: "users",
      variables: {},
    },
    show: {
      query: GET_USER_QUERY,
      variables: { id: "" },
      dataPath: "user",
    },
    // Users don't have create - they register via Telegram/OAuth
    create: {
      mutation: UPDATE_USER_SETTINGS_MUTATION, // Dummy, won't be used
      variables: {},
      dataPath: "updateUserSettings",
    },
    update: {
      mutation: UPDATE_USER_SETTINGS_MUTATION,
      variables: {},
      dataPath: "updateUserSettings",
    },
    // Users don't have delete - only deactivate
    destroy: {
      mutation: SET_USER_ACTIVE_MUTATION, // Will use custom action instead
      variables: { id: "" },
      dataPath: "setUserActive",
    },
  },

  // Table columns
  columns: [
    {
      key: "firstName",
      label: "Name",
      sortable: true,
      render: (user) => `${user.firstName} ${user.lastName}`,
    },
    {
      key: "email",
      label: "Email",
      sortable: true,
      render: (user) => user.email || "—",
    },
    {
      key: "telegramUsername",
      label: "Telegram",
      render: (user) =>
        user.telegramUsername ? `@${user.telegramUsername}` : "—",
      hideOnMobile: true,
    },
    {
      key: "isActive",
      label: "Status",
      sortable: true,
      render: (user) => (
        <Badge color={user.isActive ? "success" : "gray"} size="sm">
          {user.isActive ? "Active" : "Inactive"}
        </Badge>
      ),
    },
    {
      key: "isPremium",
      label: "Premium",
      render: (user) => (
        <Badge color={user.isPremium ? "brand" : "gray"} size="sm">
          {user.isPremium ? "Premium" : "Free"}
        </Badge>
      ),
      hideOnMobile: true,
    },
    {
      key: "settings.riskLevel",
      label: "Risk Level",
      render: (user) => {
        const colorMap = {
          CONSERVATIVE: "success" as const,
          MODERATE: "warning" as const,
          AGGRESSIVE: "error" as const,
        };
        return (
          <Badge
            color={colorMap[user.settings.riskLevel] ?? "gray"}
            size="sm"
          >
            {user.settings.riskLevel}
          </Badge>
        );
      },
      hideOnMobile: true,
    },
    {
      key: "createdAt",
      label: "Joined",
      sortable: true,
      render: (user) => new Date(user.createdAt).toLocaleDateString(),
      hideOnMobile: true,
    },
    {
      key: "actions",
      label: "",
      width: "10%",
    },
  ],

  // Form fields (only for editing settings)
  formFields: [
    {
      name: "settings.riskLevel",
      label: "Risk Level",
      type: "select",
      options: [
        { label: "Conservative", value: "CONSERVATIVE" },
        { label: "Moderate", value: "MODERATE" },
        { label: "Aggressive", value: "AGGRESSIVE" },
      ],
      helperText: "Overall risk tolerance for trading",
      validation: z.enum(["CONSERVATIVE", "MODERATE", "AGGRESSIVE"]),
      colSpan: 6,
    },
    {
      name: "settings.maxPositions",
      label: "Max Positions",
      type: "number",
      placeholder: "10",
      helperText: "Maximum number of concurrent positions",
      validation: z.number().min(1).max(100),
      colSpan: 6,
    },
    {
      name: "settings.maxPortfolioRisk",
      label: "Max Portfolio Risk (%)",
      type: "number",
      placeholder: "10",
      helperText: "Maximum portfolio risk percentage",
      validation: z.number().min(0).max(100),
      colSpan: 6,
    },
    {
      name: "settings.maxDailyDrawdown",
      label: "Max Daily Drawdown (%)",
      type: "number",
      placeholder: "5",
      helperText: "Maximum daily drawdown before circuit breaker",
      validation: z.number().min(0).max(100),
      colSpan: 6,
    },
    {
      name: "settings.maxConsecutiveLoss",
      label: "Max Consecutive Losses",
      type: "number",
      placeholder: "3",
      helperText: "Max losses before pause",
      validation: z.number().min(1).max(20),
      colSpan: 6,
    },
    {
      name: "settings.maxPositionSizeUSD",
      label: "Max Position Size (USD)",
      type: "number",
      placeholder: "10000",
      helperText: "Maximum size for single position",
      validation: z.number().min(0),
      colSpan: 6,
    },
    {
      name: "settings.maxTotalExposureUSD",
      label: "Max Total Exposure (USD)",
      type: "number",
      placeholder: "50000",
      helperText: "Maximum total portfolio exposure",
      validation: z.number().min(0),
      colSpan: 6,
    },
    {
      name: "settings.minPositionSizeUSD",
      label: "Min Position Size (USD)",
      type: "number",
      placeholder: "100",
      helperText: "Minimum size for single position",
      validation: z.number().min(0),
      colSpan: 6,
    },
    {
      name: "settings.maxLeverageMultiple",
      label: "Max Leverage",
      type: "number",
      placeholder: "3",
      helperText: "Maximum leverage multiplier",
      validation: z.number().min(1).max(100),
      colSpan: 6,
    },
    {
      name: "settings.notificationsOn",
      label: "Enable Notifications",
      type: "checkbox",
      helperText: "Receive trading notifications",
      defaultValue: true,
      colSpan: 6,
    },
    {
      name: "settings.circuitBreakerOn",
      label: "Enable Circuit Breaker",
      type: "checkbox",
      helperText: "Automatic trading pause on limits breach",
      defaultValue: true,
      colSpan: 6,
    },
    {
      name: "settings.defaultAIProvider",
      label: "AI Provider",
      type: "select",
      options: [
        { label: "OpenAI", value: "openai" },
        { label: "Anthropic", value: "anthropic" },
        { label: "Google", value: "google" },
      ],
      validation: z.string(),
      colSpan: 6,
    },
    {
      name: "settings.defaultAIModel",
      label: "AI Model",
      type: "text",
      placeholder: "gpt-4",
      validation: z.string(),
      colSpan: 6,
    },
    {
      name: "settings.timezone",
      label: "Timezone",
      type: "text",
      placeholder: "UTC",
      validation: z.string(),
      colSpan: 6,
    },
    {
      name: "settings.dailyReportTime",
      label: "Daily Report Time",
      type: "text",
      placeholder: "18:00",
      validation: z.string(),
      colSpan: 6,
    },
  ],

  // Custom actions
  actions: [
    {
      key: "activate",
      label: "Activate",
      onClick: async (user) => {
        console.log("Activate user:", user.id);
        // Will be implemented via mutation
      },
      hidden: (user) => user.isActive,
    },
    {
      key: "deactivate",
      label: "Deactivate",
      onClick: async (user) => {
        console.log("Deactivate user:", user.id);
        // Will be implemented via mutation
      },
      hidden: (user) => !user.isActive,
      destructive: true,
    },
  ],

  // Feature flags
  enableSelection: false, // No bulk operations for users
  enableSearch: true,
  defaultPageSize: 20,

  // Custom messages
  emptyStateMessage: "No users found in the system.",
  errorMessage: "Failed to load users. Please try again.",

  // Data transformations
  transformBeforeEdit: (user) => ({
    settings: {
      riskLevel: user.settings.riskLevel,
      maxPositions: user.settings.maxPositions,
      maxPortfolioRisk: user.settings.maxPortfolioRisk,
      maxDailyDrawdown: user.settings.maxDailyDrawdown,
      maxConsecutiveLoss: user.settings.maxConsecutiveLoss,
      maxPositionSizeUSD: user.settings.maxPositionSizeUSD,
      maxTotalExposureUSD: user.settings.maxTotalExposureUSD,
      minPositionSizeUSD: user.settings.minPositionSizeUSD,
      maxLeverageMultiple: user.settings.maxLeverageMultiple,
      notificationsOn: user.settings.notificationsOn,
      circuitBreakerOn: user.settings.circuitBreakerOn,
      defaultAIProvider: user.settings.defaultAIProvider,
      defaultAIModel: user.settings.defaultAIModel,
      timezone: user.settings.timezone,
      dailyReportTime: user.settings.dailyReportTime,
    },
  }),

  transformBeforeUpdate: (data) => ({
    // Extract settings from nested structure
    ...(data.settings as Record<string, unknown>),
  }),
};
