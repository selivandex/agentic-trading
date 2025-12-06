/** @format */

"use client";

import { useCallback } from "react";
import { toast } from "sonner";
import type { CrudConfig, CrudEntity, CrudActions } from "./types";

/**
 * Hook for CRUD batch actions logic
 * Handles batch operations on selected entities
 */
export function useCrudBatchActions<TEntity extends CrudEntity>(
  config: CrudConfig<TEntity>,
  selectedEntities: TEntity[],
  actions: CrudActions<TEntity>,
  refetch: () => Promise<void>
) {
  // Handle batch action
  const executeBatchAction = useCallback(
    async (actionKey: string) => {
      const action = config.bulkActions?.find((a) => a.key === actionKey);
      if (!action) return;

      try {
        const count = selectedEntities.length;
        const entityName =
          count === 1 ? config.resourceName : config.resourceNamePlural;

        // Use batch handler if available, otherwise fallback to individual calls
        if (action.onBatchClick) {
          // Execute batch action once for all entities
          await action.onBatchClick(selectedEntities);
        } else {
          // Fallback: execute action for each entity individually
          for (const entity of selectedEntities) {
            await action.onClick(entity);
          }
        }

        // Success notification
        toast.success(
          `${action.label} completed for ${count} ${entityName.toLowerCase()}`
        );

        // Clear selection and refetch
        actions.setSelectedEntities([]);
        await refetch();
      } catch (error) {
        const message =
          error instanceof Error
            ? error.message
            : `Failed to execute ${action.label}`;
        toast.error(message);
        throw error;
      }
    },
    [config, selectedEntities, actions, refetch]
  );

  return {
    executeBatchAction,
  };
}
