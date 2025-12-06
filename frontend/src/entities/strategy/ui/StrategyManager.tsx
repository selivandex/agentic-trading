/** @format */

"use client";

import { useMemo } from "react";
import { toast } from "sonner";
import { PlayCircle, XClose } from "@untitledui/icons";
import { Crud } from "@/shared/ui/crud";
import { strategyCrudConfig } from "../lib/crud-config";
import {
  usePauseStrategy,
  useResumeStrategy,
  useCloseStrategy,
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
      onClick: async (strategy: Strategy) => {
        try {
          await pauseStrategy({
            variables: { id: strategy.id },
          });
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to pause";
          throw new Error(message);
        }
      },
      disabled: () => pauseLoading,
    };

    const resumeAllAction = {
      key: "resume-all",
      label: "Resume Selected",
      icon: PlayCircle,
      onClick: async (strategy: Strategy) => {
        try {
          await resumeStrategy({
            variables: { id: strategy.id },
          });
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to resume";
          throw new Error(message);
        }
      },
      disabled: () => resumeLoading,
    };

    const closeAllAction = {
      key: "close-all",
      label: "Close Selected",
      icon: XClose,
      onClick: async (strategy: Strategy) => {
        try {
          await closeStrategy({
            variables: { id: strategy.id },
          });
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to close";
          throw new Error(message);
        }
      },
      disabled: () => closeLoading,
      destructive: true,
    };

    return {
      ...strategyCrudConfig,
      // Actions for table rows (dropdown menu)
      actions: [pauseAction, resumeAction, closeAction],
      // Actions for show page (header buttons) - only most important
      showActions: [resumeAction, closeAction],
      // Batch actions for selected rows
      bulkActions: [pauseAllAction, resumeAllAction, closeAllAction],
    };
  }, [
    pauseStrategy,
    pauseLoading,
    resumeStrategy,
    resumeLoading,
    closeStrategy,
    closeLoading,
  ]);

  return <Crud config={enhancedConfig} />;
}
