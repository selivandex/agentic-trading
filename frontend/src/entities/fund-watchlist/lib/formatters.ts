/** @format */

import type { FundWatchlist } from "@/entities/fund-watchlist/model/types";

/**
 * Format tier for display
 */
export function formatTier(tier: number): string {
  return `Tier ${tier}`;
}

/**
 * Get tier color based on tier number
 */
export function getTierColor(tier: number): "gray" | "brand" | "success" {
  if (tier === 1) return "success";
  if (tier === 2) return "brand";
  return "gray";
}

/**
 * Get status color based on active/paused state
 */
export function getWatchlistStatusColor(
  item: FundWatchlist
): "success" | "warning" | "gray" {
  if (!item.isActive) return "gray";
  if (item.isPaused) return "warning";
  return "success";
}

/**
 * Get status text
 */
export function getWatchlistStatusText(item: FundWatchlist): string {
  if (!item.isActive) return "Inactive";
  if (item.isPaused) return "Paused";
  return "Active";
}

/**
 * Format market type for display
 */
export function formatMarketType(marketType: string): string {
  return marketType.charAt(0).toUpperCase() + marketType.slice(1).toLowerCase();
}

/**
 * Format category for display
 */
export function formatCategory(category: string): string {
  return category
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join(" ");
}
