/** @format */

"use client";

import { useMemo } from "react";
import { toast } from "sonner";
import { PlayCircle, XClose, Trash01 } from "@untitledui/icons";
import { Crud } from "@/shared/ui/crud";
import { strategyCrudConfig } from "../lib/crud-config";
import {
  usePauseStrategy,
  useResumeStrategy,
  useCloseStrategy,
  useBatchPauseStrategies,
  useBatchResumeStrategies,
  useBatchCloseStrategies,
  useBatchDeleteStrategies,
} from "@/entities/strategy";
import type { Strategy } from "@/entities/strategy";

/**
 * StrategyManager Component
 *
 * Wrapper around Crud that adds real mutation implementations for strategy actions
 */
export function StrategyManager() {
  const [pauseStrategy, { loading: pauseLoading }] = usePauseStrategy();
  const [resumeStrategy, { loading: resumeLoading }] = useResumeStrategy();
  const [closeStrategy, { loading: closeLoading }] = useCloseStrategy();
  const [batchPauseStrategies, { loading: batchPauseLoading }] =
    useBatchPauseStrategies();
  const [batchResumeStrategies, { loading: batchResumeLoading }] =
    useBatchResumeStrategies();
  const [batchCloseStrategies, { loading: batchCloseLoading }] =
    useBatchCloseStrategies();
  const [batchDeleteStrategies, { loading: batchDeleteLoading }] =
    useBatchDeleteStrategies();

  // Enhanced config with real mutation implementations
  const enhancedConfig = useMemo(() => {
    // Common actions used in multiple places
    const pauseAction = {
      key: "pause",
      label: "Pause",
      onClick: async (strategy: Strategy) => {
        try {
          await pauseStrategy({
            variables: { id: strategy.id },
          });
          toast.success("Strategy paused successfully");
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to pause";
          toast.error(message);
        }
      },
      disabled: () => pauseLoading,
      hidden: (strategy: Strategy) => strategy.status !== "active",
    };

    const resumeAction = {
      key: "resume",
      label: "Resume",
      icon: PlayCircle,
      onClick: async (strategy: Strategy) => {
        try {
          await resumeStrategy({
            variables: { id: strategy.id },
          });
          toast.success("Strategy resumed successfully");
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to resume";
          toast.error(message);
        }
      },
      disabled: () => resumeLoading,
      hidden: (strategy: Strategy) => strategy.status !== "paused",
    };

    const closeAction = {
      key: "close",
      label: "Close Strategy",
      icon: XClose,
      onClick: async (strategy: Strategy) => {
        if (
          !confirm(
            "Are you sure you want to close this strategy? This will liquidate all positions."
          )
        ) {
          return;
        }

        try {
          await closeStrategy({
            variables: { id: strategy.id },
          });
          toast.success("Strategy closed successfully");
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to close";
          toast.error(message);
        }
      },
      disabled: () => closeLoading,
      hidden: (strategy: Strategy) => strategy.status === "closed",
      destructive: true,
    };

    // Batch actions for selected rows
    const pauseAllAction = {
      key: "pause-all",
      label: "Pause Selected",
      // Fallback for single entity (not used in batch context)
      onClick: async (_strategy: Strategy) => {
        try {
          await pauseStrategy({
            variables: { id: _strategy.id },
          });
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to pause";
          throw new Error(message);
        }
      },
      // Batch handler - more efficient
      onBatchClick: async (strategies: Strategy[]) => {
        try {
          const ids = strategies.map((s) => s.id);
          const result = await batchPauseStrategies({
            variables: { ids },
          });
          const count = result.data?.batchPauseStrategies ?? 0;
          toast.success(`Paused ${count} of ${ids.length} strategies`);
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to pause";
          toast.error(message);
          throw error;
        }
      },
      disabled: () => pauseLoading || batchPauseLoading,
      // Hide if all selected strategies are not active
      hiddenForBatch: (strategies: Strategy[]) =>
        !strategies.some((s) => s.status === "active"),
    };

    const resumeAllAction = {
      key: "resume-all",
      label: "Resume Selected",
      icon: PlayCircle,
      // Fallback for single entity
      onClick: async (_strategy: Strategy) => {
        try {
          await resumeStrategy({
            variables: { id: _strategy.id },
          });
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to resume";
          throw new Error(message);
        }
      },
      // Batch handler - more efficient
      onBatchClick: async (strategies: Strategy[]) => {
        try {
          const ids = strategies.map((s) => s.id);
          const result = await batchResumeStrategies({
            variables: { ids },
          });
          const count = result.data?.batchResumeStrategies ?? 0;
          toast.success(`Resumed ${count} of ${ids.length} strategies`);
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to resume";
          toast.error(message);
          throw error;
        }
      },
      disabled: () => resumeLoading || batchResumeLoading,
      // Hide if all selected strategies are not paused
      hiddenForBatch: (strategies: Strategy[]) =>
        !strategies.some((s) => s.status === "paused"),
    };

    const closeAllAction = {
      key: "close-all",
      label: "Close Selected",
      icon: XClose,
      // Fallback for single entity
      onClick: async (_strategy: Strategy) => {
        try {
          await closeStrategy({
            variables: { id: _strategy.id },
          });
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to close";
          throw new Error(message);
        }
      },
      // Batch handler - more efficient
      onBatchClick: async (strategies: Strategy[]) => {
        if (
          !confirm(
            `Are you sure you want to close ${strategies.length} strategies? This will liquidate all positions.`
          )
        ) {
          return;
        }

        try {
          const ids = strategies.map((s) => s.id);
          const result = await batchCloseStrategies({
            variables: { ids },
          });
          const count = result.data?.batchCloseStrategies ?? 0;
          toast.success(`Closed ${count} of ${ids.length} strategies`);
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to close";
          toast.error(message);
          throw error;
        }
      },
      disabled: () => closeLoading || batchCloseLoading,
      destructive: true,
      // Hide if all selected strategies are already closed
      hiddenForBatch: (strategies: Strategy[]) =>
        !strategies.some((s) => s.status !== "closed"),
    };

    const deleteAllAction = {
      key: "delete-all",
      label: "Delete Selected",
      icon: Trash01,
      // Fallback for single entity
      onClick: async (_strategy: Strategy) => {
        // This would need a delete mutation - placeholder
        toast.error("Delete not implemented for single strategy");
      },
      // Batch handler - more efficient
      onBatchClick: async (strategies: Strategy[]) => {
        // Only allow deletion of closed strategies
        const closedStrategies = strategies.filter(
          (s) => s.status === "closed"
        );
        if (closedStrategies.length === 0) {
          toast.error("Can only delete closed strategies");
          return;
        }

        if (closedStrategies.length < strategies.length) {
          toast.warning(
            `Only ${closedStrategies.length} of ${strategies.length} selected strategies are closed and can be deleted`
          );
        }

        if (
          !confirm(
            `Are you sure you want to permanently delete ${closedStrategies.length} closed strategies? This action cannot be undone.`
          )
        ) {
          return;
        }

        try {
          const ids = closedStrategies.map((s) => s.id);
          const result = await batchDeleteStrategies({
            variables: { ids },
          });
          const count = result.data?.batchDeleteStrategies ?? 0;
          toast.success(`Deleted ${count} of ${ids.length} strategies`);
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to delete";
          toast.error(message);
          throw error;
        }
      },
      disabled: () => batchDeleteLoading,
      destructive: true,
      // Hide if no closed strategies are selected
      hiddenForBatch: (strategies: Strategy[]) =>
        !strategies.some((s) => s.status === "closed"),
    };

    return {
      ...strategyCrudConfig,
      // Actions for table rows (dropdown menu)
      actions: [pauseAction, resumeAction, closeAction],
      // Actions for show page (header buttons) - only most important
      showActions: [resumeAction, closeAction],
      // Batch actions for selected rows
      bulkActions: [
        pauseAllAction,
        resumeAllAction,
        closeAllAction,
        deleteAllAction,
      ],
    };
  }, [
    pauseStrategy,
    pauseLoading,
    resumeStrategy,
    resumeLoading,
    closeStrategy,
    closeLoading,
    batchPauseStrategies,
    batchPauseLoading,
    batchResumeStrategies,
    batchResumeLoading,
    batchCloseStrategies,
    batchCloseLoading,
    batchDeleteStrategies,
    batchDeleteLoading,
  ]);

  return <Crud config={enhancedConfig} />;
}
