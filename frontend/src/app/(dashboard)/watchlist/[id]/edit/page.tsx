/** @format */

"use client";

import { Crud } from "@/shared/ui/crud";
import { fundWatchlistCrudConfig } from "@/entities/fund-watchlist";
import { use } from "react";

/**
 * Edit Fund Watchlist Page
 *
 * Edit an existing watchlist item (category, tier, etc.)
 */
export default function EditWatchlistPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);

  return (
    <div className="h-full">
      <Crud config={fundWatchlistCrudConfig} mode="edit" entityId={id} />
    </div>
  );
}
