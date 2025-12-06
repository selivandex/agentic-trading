/** @format */

import { z } from "zod";
import type { CrudConfig } from "@/shared/lib/crud";
import { Badge } from "@/components/base/badges/badges";
import type { FundWatchlist } from "@/entities/fund-watchlist";
import {
  GET_FUND_WATCHLISTS_CONNECTION_QUERY,
  GET_FUND_WATCHLIST_QUERY,
  CREATE_FUND_WATCHLIST_MUTATION,
  UPDATE_FUND_WATCHLIST_MUTATION,
} from "@/entities/fund-watchlist";

/**
 * CRUD Configuration for FundWatchlist entity
 *
 * @example
 * ```tsx
 * import { Crud } from '@/shared/ui/crud';
 * import { fundWatchlistCrudConfig } from '@/entities/fund-watchlist';
 *
 * function FundWatchlistPage() {
 *   return <Crud config={fundWatchlistCrudConfig} />;
 * }
 * ```
 */
export const fundWatchlistCrudConfig: CrudConfig<FundWatchlist> = {
  // Resource names
  resourceName: "Watchlist Item",
  resourceNamePlural: "Watchlist Items",

  // Base path for navigation
  basePath: "/watchlist",

  // Display name for breadcrumbs and titles
  getDisplayName: (watchlist) => watchlist.symbol,

  // GraphQL operations
  graphql: {
    list: {
      query: GET_FUND_WATCHLISTS_CONNECTION_QUERY,
      dataPath: "fundWatchlistsConnection",
      variables: {},
      useConnection: true, // Relay pagination
    },
    show: {
      query: GET_FUND_WATCHLIST_QUERY,
      variables: { id: "" },
      dataPath: "fundWatchlist",
    },
    create: {
      mutation: CREATE_FUND_WATCHLIST_MUTATION,
      variables: {},
      dataPath: "createFundWatchlist",
    },
    update: {
      mutation: UPDATE_FUND_WATCHLIST_MUTATION,
      variables: {},
      dataPath: "updateFundWatchlist",
    },
    destroy: undefined, // Delete handled via custom actions
  },

  // Table columns
  columns: [
    {
      key: "symbol",
      label: "Symbol",
      sortable: true,
      width: "20%",
    },
    {
      key: "marketType",
      label: "Market Type",
      sortable: true,
      render: (item) => {
        if (!item.marketType) return "—";
        const marketType = item.marketType.toUpperCase();
        const colorMap: Record<string, "blue" | "purple" | "gray"> = {
          SPOT: "blue",
          FUTURES: "purple",
        };
        return (
          <Badge color={colorMap[marketType] ?? "gray"} size="sm">
            {marketType}
          </Badge>
        );
      },
    },
    {
      key: "category",
      label: "Category",
      sortable: true,
      render: (item) => {
        if (!item.category) return "—";
        const category = item.category.toUpperCase();
        const categoryMap: Record<string, string> = {
          LARGE_CAP: "Large Cap",
          MID_CAP: "Mid Cap",
          SMALL_CAP: "Small Cap",
          DEFI: "DeFi",
          GAMING: "Gaming",
          AI: "AI",
          MEME: "Meme",
        };
        return categoryMap[category] || item.category;
      },
    },
    {
      key: "tier",
      label: "Tier",
      sortable: true,
      render: (item) => {
        if (item.tier === undefined || item.tier === null) return "—";
        const colorMap: Record<
          number,
          "success" | "blue" | "warning" | "gray"
        > = {
          1: "success",
          2: "blue",
          3: "warning",
          4: "gray",
        };
        return (
          <Badge color={colorMap[item.tier] ?? "gray"} size="sm">
            Tier {item.tier}
          </Badge>
        );
      },
    },
    {
      key: "status",
      label: "Status",
      sortable: true,
      render: (item) => {
        if (!item) return "—";
        if (item.isPaused === true) {
          return (
            <Badge color="warning" size="sm">
              Paused
            </Badge>
          );
        }
        if (item.isActive === false) {
          return (
            <Badge color="gray" size="sm">
              Inactive
            </Badge>
          );
        }
        return (
          <Badge color="success" size="sm">
            Active
          </Badge>
        );
      },
    },
    {
      key: "lastAnalyzedAt",
      label: "Last Analyzed",
      sortable: true,
      render: (item) =>
        item.lastAnalyzedAt
          ? new Date(item.lastAnalyzedAt).toLocaleDateString()
          : "Never",
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
      name: "symbol",
      label: "Symbol",
      type: "text",
      placeholder: "e.g., BTCUSDT",
      helperText: "Trading pair symbol",
      validation: z
        .string()
        .min(1, "Symbol is required")
        .max(20, "Symbol too long")
        .toUpperCase(),
      disabled: (mode) => mode === "edit", // Read-only in edit mode
      colSpan: 6,
    },
    {
      name: "marketType",
      label: "Market Type",
      type: "select",
      options: [
        { label: "Spot", value: "SPOT" },
        { label: "Futures", value: "FUTURES" },
      ],
      validation: z.enum(["SPOT", "FUTURES"]),
      //disabled: (mode) => mode === "edit", // Read-only in edit mode
      colSpan: 6,
    },
    {
      name: "category",
      label: "Category",
      type: "select",
      options: [
        { label: "Large Cap", value: "LARGE_CAP" },
        { label: "Mid Cap", value: "MID_CAP" },
        { label: "Small Cap", value: "SMALL_CAP" },
        { label: "DeFi", value: "DEFI" },
        { label: "Gaming", value: "GAMING" },
        { label: "AI", value: "AI" },
        { label: "Meme", value: "MEME" },
      ],
      validation: z.enum([
        "LARGE_CAP",
        "MID_CAP",
        "SMALL_CAP",
        "DEFI",
        "GAMING",
        "AI",
        "MEME",
      ]),
      colSpan: 6,
    },
    {
      name: "tier",
      label: "Tier",
      type: "select",
      options: [
        { label: "Tier 1", value: "1" },
        { label: "Tier 2", value: "2" },
        { label: "Tier 3", value: "3" },
        { label: "Tier 4", value: "4" },
      ],
      validation: z.number().min(1).max(4),
      colSpan: 6,
    },
  ],

  // Custom actions (overridden in FundWatchlistManager component with real mutations)
  actions: [],

  // Bulk actions (overridden in FundWatchlistManager component with real mutations)
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

  // Feature flags
  enableSelection: true,
  enableSearch: true,
  defaultPageSize: 20,

  // Custom messages
  emptyStateMessage: "No watchlist items yet. Add symbols to start monitoring.",
  errorMessage: "Failed to load watchlist items. Please try again.",

  // Data transformations
  transformBeforeEdit: (item) => ({
    symbol: item.symbol,
    marketType: item.marketType,
    category: item.category,
    tier: item.tier,
  }),

  transformBeforeCreate: (data) => ({
    symbol: (data.symbol as string).toUpperCase(),
    marketType: (data.marketType as string).toUpperCase(),
    category: (data.category as string).toUpperCase(),
    tier: Number(data.tier),
  }),

  transformBeforeUpdate: (data) => {
    const updateInput: Record<string, unknown> = {};

    if (data.category) {
      updateInput.category = (data.category as string).toUpperCase();
    }

    if (data.tier !== undefined) {
      updateInput.tier = Number(data.tier);
    }

    return updateInput;
  },
};
