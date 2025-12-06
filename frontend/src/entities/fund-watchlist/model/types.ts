/** @format */

/**
 * Fund Watchlist Entity Types
 *
 * Watchlist of trading symbols being monitored by the fund
 */

import type { CrudEntity } from "@/shared/lib/crud";

export interface FundWatchlist extends CrudEntity {
  id: string;
  symbol: string;
  marketType: string;

  // Metadata
  category: string;
  tier: number;

  // State
  isActive: boolean;
  isPaused: boolean;
  pausedReason?: string | null;

  // Analytics
  lastAnalyzedAt?: string | null;

  // Timestamps
  createdAt: string;
  updatedAt: string;
}

export interface CreateFundWatchlistInput {
  symbol: string;
  marketType: string;
  category: string;
  tier: number;
}

export interface UpdateFundWatchlistInput {
  category?: string;
  tier?: number;
  isActive?: boolean;
  isPaused?: boolean;
  pausedReason?: string;
}
