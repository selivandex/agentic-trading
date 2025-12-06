/** @format */

"use client";

import { useCallback } from "react";
import type { Key } from "react";
import type { CrudEntity, CrudActions } from "./types";

/**
 * Hook for CRUD selection logic
 * Handles entity selection and batch operations
 */
export function useCrudSelection<TEntity extends CrudEntity>(
  entities: TEntity[],
  actions: CrudActions<TEntity>
) {
  // Handle selection change
  const handleSelectionChange = useCallback(
    (keys: "all" | Set<Key>) => {
      if (keys === "all") {
        // Select all visible entities
        actions.setSelectedEntities(entities);
      } else {
        const selectedIds = Array.from(keys) as string[];
        const selected = entities.filter((e) => selectedIds.includes(e.id));
        actions.setSelectedEntities(selected);
      }
    },
    [entities, actions]
  );

  // Clear selection
  const clearSelection = useCallback(() => {
    actions.setSelectedEntities([]);
  }, [actions]);

  return {
    handleSelectionChange,
    clearSelection,
  };
}

