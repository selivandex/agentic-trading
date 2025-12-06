/** @format */

"use client";

import { useMemo } from "react";
import { toast } from "sonner";
import { PlayCircle, PauseCircle, Trash01 } from "@untitledui/icons";
import { Crud } from "@/shared/ui/crud";
import { fundWatchlistCrudConfig } from "../lib/crud-config";
import {
  usePauseFundWatchlist,
  useResumeFundWatchlist,
  useDeleteFundWatchlist,
  useBatchDeleteFundWatchlists,
} from "@/entities/fund-watchlist";
import type { FundWatchlist } from "@/entities/fund-watchlist";

/**
 * FundWatchlistManager Component
 *
 * Wrapper around Crud that adds real mutation implementations for fund watchlist actions
 */
export function FundWatchlistManager() {
  const [pauseItem, { loading: pauseLoading }] = usePauseFundWatchlist();
  const [resumeItem, { loading: resumeLoading }] = useResumeFundWatchlist();
  const [deleteItem, { loading: deleteLoading }] = useDeleteFundWatchlist();
  const [batchDeleteItems, { loading: batchDeleteLoading }] =
    useBatchDeleteFundWatchlists();

  // Enhanced config with real mutation implementations
  const enhancedConfig = useMemo(() => {
    // Common actions used in multiple places
    const pauseAction = {
      key: "pause",
      label: "Pause",
      icon: PauseCircle,
      onClick: async (item: FundWatchlist) => {
        try {
          await pauseItem({
            variables: { id: item.id },
          });
          toast.success("Watchlist item paused successfully");
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to pause";
          toast.error(message);
        }
      },
      disabled: () => pauseLoading,
      hidden: (item: FundWatchlist) => item.isPaused || !item.isActive,
    };

    const resumeAction = {
      key: "resume",
      label: "Resume",
      icon: PlayCircle,
      onClick: async (item: FundWatchlist) => {
        try {
          await resumeItem({
            variables: { id: item.id },
          });
          toast.success("Watchlist item resumed successfully");
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to resume";
          toast.error(message);
        }
      },
      disabled: () => resumeLoading,
      hidden: (item: FundWatchlist) => !item.isPaused,
    };

    const deleteAction = {
      key: "delete",
      label: "Delete",
      icon: Trash01,
      onClick: async (item: FundWatchlist) => {
        if (!confirm(`Are you sure you want to delete ${item.symbol}?`)) {
          return;
        }

        try {
          await deleteItem({
            variables: { id: item.id },
          });
          toast.success("Watchlist item deleted successfully");
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to delete";
          toast.error(message);
        }
      },
      disabled: () => deleteLoading,
      destructive: true,
    };

    // Batch actions for selected rows
    const deleteAllAction = {
      key: "delete-all",
      label: "Delete Selected",
      icon: Trash01,
      // Fallback for single entity
      onClick: async (_item: FundWatchlist) => {
        try {
          await deleteItem({
            variables: { id: _item.id },
          });
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to delete";
          throw new Error(message);
        }
      },
      // Batch handler - more efficient
      onBatchClick: async (items: FundWatchlist[]) => {
        if (
          !confirm(
            `Are you sure you want to delete ${items.length} watchlist items? This action cannot be undone.`
          )
        ) {
          return;
        }

        try {
          const ids = items.map((item) => item.id);
          const result = await batchDeleteItems({
            variables: { ids },
          });
          const count = result.data?.batchDeleteFundWatchlists ?? 0;
          toast.success(`Deleted ${count} of ${ids.length} items`);
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to delete";
          toast.error(message);
          throw error;
        }
      },
      disabled: () => batchDeleteLoading,
      destructive: true,
    };

    return {
      ...fundWatchlistCrudConfig,
      // Actions for table rows (dropdown menu)
      actions: [pauseAction, resumeAction, deleteAction],
      // Actions for show page (header buttons) - only most important
      showActions: [resumeAction, deleteAction],
      // Batch actions for selected rows
      bulkActions: [deleteAllAction],
    };
  }, [
    pauseItem,
    pauseLoading,
    resumeItem,
    resumeLoading,
    deleteItem,
    deleteLoading,
    batchDeleteItems,
    batchDeleteLoading,
  ]);

  return <Crud config={enhancedConfig} />;
}
