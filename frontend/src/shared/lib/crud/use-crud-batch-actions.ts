/** @format */

"use client";

import { useCallback } from "react";
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

      // Execute action for all selected entities
      for (const entity of selectedEntities) {
        await action.onClick(entity);
      }

      // Clear selection and refetch
      actions.setSelectedEntities([]);
      await refetch();
    },
    [config.bulkActions, selectedEntities, actions, refetch]
  );

  return {
    executeBatchAction,
  };
}

