/** @format */

"use client";

import { Crud } from "@/shared/ui/crud";
import { fundWatchlistCrudConfig } from "@/entities/fund-watchlist";
import { use } from "react";

/**
 * Fund Watchlist Detail Page
 *
 * Shows detailed information about a specific watchlist item
 */
export default function WatchlistDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);

  return (
    <div className="h-full">
      <Crud config={fundWatchlistCrudConfig} mode="show" entityId={id} />
    </div>
  );
}
