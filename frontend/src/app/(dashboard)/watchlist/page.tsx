/** @format */

"use client";

import { FundWatchlistManager } from "@/entities/fund-watchlist";

/**
 * Fund Watchlist Management Page
 *
 * Full CRUD interface for managing fund watchlist:
 * - List view with search, sort, pagination
 * - Create new watchlist item
 * - Edit existing watchlist item
 * - View watchlist item details
 * - Delete watchlist item
 * - Pause/Resume actions
 * - Scopes: All, Active, Paused, Inactive
 * - Filters: Market Type, Category, Tier
 */
export default function WatchlistPage() {
  return <FundWatchlistManager />;
}
