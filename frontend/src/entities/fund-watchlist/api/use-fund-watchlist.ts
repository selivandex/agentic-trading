/** @format */

import { useQuery, useMutation } from "@apollo/client";
import {
  GET_FUND_WATCHLIST_QUERY,
  GET_FUND_WATCHLIST_BY_SYMBOL_QUERY,
  GET_FUND_WATCHLISTS_QUERY,
  GET_MONITORED_SYMBOLS_QUERY,
  CREATE_FUND_WATCHLIST_MUTATION,
  UPDATE_FUND_WATCHLIST_MUTATION,
  DELETE_FUND_WATCHLIST_MUTATION,
  TOGGLE_FUND_WATCHLIST_PAUSE_MUTATION,
} from "@/entities/fund-watchlist/api/fund-watchlist.graphql";
import type {
  FundWatchlist,
  CreateFundWatchlistInput,
  UpdateFundWatchlistInput,
} from "@/entities/fund-watchlist/model/types";

/**
 * Hook to get watchlist item by ID
 */
export function useFundWatchlist(id: string) {
  return useQuery<{ fundWatchlist: FundWatchlist | null }>(
    GET_FUND_WATCHLIST_QUERY,
    {
      variables: { id },
      skip: !id,
    }
  );
}

/**
 * Hook to get watchlist item by symbol
 */
export function useFundWatchlistBySymbol(symbol: string, marketType: string) {
  return useQuery<{ fundWatchlistBySymbol: FundWatchlist | null }>(
    GET_FUND_WATCHLIST_BY_SYMBOL_QUERY,
    {
      variables: { symbol, marketType },
      skip: !symbol || !marketType,
    }
  );
}

/**
 * Hook to get all watchlist items
 */
export function useFundWatchlists(filters?: {
  limit?: number;
  offset?: number;
  isActive?: boolean;
  category?: string;
  tier?: number;
}) {
  return useQuery<{ fundWatchlists: FundWatchlist[] }>(
    GET_FUND_WATCHLISTS_QUERY,
    {
      variables: filters,
    }
  );
}

/**
 * Hook to get monitored symbols
 */
export function useMonitoredSymbols(marketType?: string) {
  return useQuery<{ monitoredSymbols: FundWatchlist[] }>(
    GET_MONITORED_SYMBOLS_QUERY,
    {
      variables: { marketType },
    }
  );
}

/**
 * Hook to create watchlist item
 */
export function useCreateFundWatchlist() {
  return useMutation<
    { createFundWatchlist: FundWatchlist },
    { input: CreateFundWatchlistInput }
  >(CREATE_FUND_WATCHLIST_MUTATION, {
    refetchQueries: ["GetFundWatchlists", "GetMonitoredSymbols"],
  });
}

/**
 * Hook to update watchlist item
 */
export function useUpdateFundWatchlist() {
  return useMutation<
    { updateFundWatchlist: FundWatchlist },
    { id: string; input: UpdateFundWatchlistInput }
  >(UPDATE_FUND_WATCHLIST_MUTATION);
}

/**
 * Hook to delete watchlist item
 */
export function useDeleteFundWatchlist() {
  return useMutation<{ deleteFundWatchlist: boolean }, { id: string }>(
    DELETE_FUND_WATCHLIST_MUTATION,
    {
      refetchQueries: ["GetFundWatchlists", "GetMonitoredSymbols"],
    }
  );
}

/**
 * Hook to toggle pause
 */
export function useToggleFundWatchlistPause() {
  return useMutation<
    { toggleFundWatchlistPause: FundWatchlist },
    { id: string; isPaused: boolean; reason?: string }
  >(TOGGLE_FUND_WATCHLIST_PAUSE_MUTATION);
}
