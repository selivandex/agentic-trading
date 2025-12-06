/** @format */

"use client";

import { Crud } from "@/shared/ui/crud";
import { fundWatchlistCrudConfig } from "@/entities/fund-watchlist";

/**
 * Create New Fund Watchlist Item Page
 *
 * Add a new symbol to the fund's watchlist
 */
export default function NewWatchlistPage() {
  return (
    <div className="h-full">
      <Crud config={fundWatchlistCrudConfig} mode="new" />
    </div>
  );
}
