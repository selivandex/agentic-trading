/** @format */

import { z } from "zod";
import type { CrudConfig } from "@/shared/lib/crud";
import { Badge } from "@/components/base/badges/badges";
import { Panel } from "@/shared/ui/panel";
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

  // Base path for navigation
  basePath: "/users",

  // Display name for breadcrumbs and titles
  getDisplayName: (user) => {
    const name = `${user.firstName} ${user.lastName}`.trim();
    return name || user.email || user.telegramUsername || "Unknown User";
  },

  // GraphQL operations
  graphql: {
    list: {
      query: GET_USERS_QUERY,
      dataPath: "users",
      variables: {},
      useConnection: true, // Relay pagination
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
          <Badge color={colorMap[user.settings.riskLevel] ?? "gray"} size="sm">
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

  // Custom actions (will be overridden in UserManager component with real mutations)
  actions: [],

  // Bulk actions (will be overridden in UserManager component with real mutations)
  bulkActions: [],

  // Tabs configuration (scopes from backend)
  tabs: {
    enabled: true,
    type: "underline",
    size: "md",
    filterVariable: "scope",
    // defaultScope not set - will use first scope from backend
  },

  // Dynamic filters configuration (filters from backend)
  dynamicFilters: {
    enabled: true,
  },

  // Show page customization
  show: {
    layout: "two-column",
    mainContent: (user) => (
      <>
        {/* User Profile Info */}
        <Panel>
          <Panel.Header title="Profile Information" />
          <Panel.Divider />
          <Panel.Content>
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <dt className="text-sm font-medium text-secondary">
                    First Name
                  </dt>
                  <dd className="mt-1 text-base text-primary">
                    {user.firstName || "—"}
                  </dd>
                </div>
                <div>
                  <dt className="text-sm font-medium text-secondary">
                    Last Name
                  </dt>
                  <dd className="mt-1 text-base text-primary">
                    {user.lastName || "—"}
                  </dd>
                </div>
              </div>

              <div>
                <dt className="text-sm font-medium text-secondary">Email</dt>
                <dd className="mt-1 text-base text-primary">
                  {user.email || "—"}
                </dd>
              </div>

              <div>
                <dt className="text-sm font-medium text-secondary">
                  Telegram Username
                </dt>
                <dd className="mt-1 text-base text-primary">
                  {user.telegramUsername ? `@${user.telegramUsername}` : "—"}
                </dd>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <dt className="text-sm font-medium text-secondary">Status</dt>
                  <dd className="mt-1">
                    <Badge color={user.isActive ? "success" : "gray"} size="sm">
                      {user.isActive ? "Active" : "Inactive"}
                    </Badge>
                  </dd>
                </div>
                <div>
                  <dt className="text-sm font-medium text-secondary">
                    Premium
                  </dt>
                  <dd className="mt-1">
                    <Badge color={user.isPremium ? "brand" : "gray"} size="sm">
                      {user.isPremium ? "Premium" : "Free"}
                    </Badge>
                  </dd>
                </div>
              </div>

              <div>
                <dt className="text-sm font-medium text-secondary">
                  Member Since
                </dt>
                <dd className="mt-1 text-base text-primary">
                  {new Date(user.createdAt).toLocaleDateString("en-US", {
                    year: "numeric",
                    month: "long",
                    day: "numeric",
                  })}
                </dd>
              </div>
            </div>
          </Panel.Content>
        </Panel>

        {/* Risk Settings */}
        <Panel className="mt-6">
          <Panel.Header
            title="Risk Settings"
            subtitle="Trading risk parameters"
          />
          <Panel.Divider />
          <Panel.Content>
            <div className="space-y-4">
              <div>
                <dt className="text-sm font-medium text-secondary">
                  Risk Level
                </dt>
                <dd className="mt-1">
                  <Badge
                    color={
                      user.settings.riskLevel === "CONSERVATIVE"
                        ? "success"
                        : user.settings.riskLevel === "MODERATE"
                        ? "warning"
                        : "error"
                    }
                    size="sm"
                  >
                    {user.settings.riskLevel}
                  </Badge>
                </dd>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <dt className="text-sm font-medium text-secondary">
                    Max Positions
                  </dt>
                  <dd className="mt-1 text-base text-primary">
                    {user.settings.maxPositions}
                  </dd>
                </div>
                <div>
                  <dt className="text-sm font-medium text-secondary">
                    Max Portfolio Risk
                  </dt>
                  <dd className="mt-1 text-base text-primary">
                    {user.settings.maxPortfolioRisk}%
                  </dd>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <dt className="text-sm font-medium text-secondary">
                    Max Daily Drawdown
                  </dt>
                  <dd className="mt-1 text-base text-primary">
                    {user.settings.maxDailyDrawdown}%
                  </dd>
                </div>
                <div>
                  <dt className="text-sm font-medium text-secondary">
                    Max Leverage
                  </dt>
                  <dd className="mt-1 text-base text-primary">
                    {user.settings.maxLeverageMultiple}x
                  </dd>
                </div>
              </div>
            </div>
          </Panel.Content>
        </Panel>
      </>
    ),
    sidebar: (user) => (
      <>
        {/* Exchange Accounts Panel - TODO: Replace with real GraphQL query for user.id */}
        {user.id && (
          <>
            <Panel>
              <Panel.Header title="Exchange Accounts" />
              <Panel.Divider />
              <Panel.Content noPadding>
                <div className="divide-y divide-secondary">
                  {/* Mock data - в реальности тут будет GraphQL запрос */}
                  <div className="px-6 py-4">
                    <div className="flex items-center justify-between">
                      <div>
                        <div className="font-medium text-primary">Binance</div>
                        <div className="text-sm text-secondary">
                          Connected 2 days ago
                        </div>
                      </div>
                      <Badge color="success" size="sm">
                        Active
                      </Badge>
                    </div>
                  </div>
                  <div className="px-6 py-4">
                    <div className="flex items-center justify-between">
                      <div>
                        <div className="font-medium text-primary">Bybit</div>
                        <div className="text-sm text-secondary">
                          Connected 5 days ago
                        </div>
                      </div>
                      <Badge color="success" size="sm">
                        Active
                      </Badge>
                    </div>
                  </div>
                  <div className="px-6 py-4">
                    <div className="flex items-center justify-between">
                      <div>
                        <div className="font-medium text-primary">OKX</div>
                        <div className="text-sm text-secondary">
                          Connected 1 month ago
                        </div>
                      </div>
                      <Badge color="gray" size="sm">
                        Inactive
                      </Badge>
                    </div>
                  </div>
                </div>
              </Panel.Content>
              <Panel.Footer>
                <div className="text-sm text-secondary">
                  Total: 3 accounts connected
                </div>
              </Panel.Footer>
            </Panel>

            {/* Recent Activity Panel */}
            <Panel className="mt-6">
              <Panel.Header title="Recent Activity" />
              <Panel.Divider />
              <Panel.Content noPadding>
                <div className="divide-y divide-secondary">
                  <div className="px-6 py-3">
                    <div className="flex items-start gap-3">
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium text-primary truncate">
                          Opened BTC/USDT position
                        </p>
                        <p className="text-xs text-secondary">2 hours ago</p>
                      </div>
                      <Badge color="success" size="sm">
                        Trade
                      </Badge>
                    </div>
                  </div>
                  <div className="px-6 py-3">
                    <div className="flex items-start gap-3">
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium text-primary truncate">
                          Updated risk settings
                        </p>
                        <p className="text-xs text-secondary">1 day ago</p>
                      </div>
                      <Badge color="gray" size="sm">
                        Settings
                      </Badge>
                    </div>
                  </div>
                  <div className="px-6 py-3">
                    <div className="flex items-start gap-3">
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium text-primary truncate">
                          Connected Binance account
                        </p>
                        <p className="text-xs text-secondary">2 days ago</p>
                      </div>
                      <Badge color="brand" size="sm">
                        Account
                      </Badge>
                    </div>
                  </div>
                </div>
              </Panel.Content>
            </Panel>

            {/* Statistics Panel */}
            <Panel className="mt-6">
              <Panel.Header title="Statistics" />
              <Panel.Divider />
              <Panel.Content>
                <div className="space-y-3">
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-secondary">Total Trades</span>
                    <span className="text-sm font-semibold text-primary">
                      247
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-secondary">Win Rate</span>
                    <span className="text-sm font-semibold text-success">
                      64.5%
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-secondary">Total P&L</span>
                    <span className="text-sm font-semibold text-success">
                      +$12,450
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-secondary">
                      Active Positions
                    </span>
                    <span className="text-sm font-semibold text-primary">
                      5
                    </span>
                  </div>
                </div>
              </Panel.Content>
            </Panel>
          </>
        )}
      </>
    ),
  },

  // Feature flags
  enableSelection: true,
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
