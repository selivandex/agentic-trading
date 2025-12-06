/** @format */

"use client";

import { useCallback } from "react";
import type { CrudConfig, CrudEntity } from "./types";
import { useCrudShowQuery } from "./use-crud-query";
import { useCrudMutations } from "./use-crud-mutations";

/**
 * Hook for CRUD show logic
 * Handles entity fetching and delete operation
 */
export function useCrudShow<TEntity extends CrudEntity>(
  config: CrudConfig<TEntity>,
  entityId: string,
  onDelete?: () => void
) {
  // Fetch entity
  const { entity, loading, error, refetch } = useCrudShowQuery<TEntity>(
    config,
    entityId
  );

  // Mutations
  const { destroy, destroyLoading } = useCrudMutations<TEntity>(config);

  // Handle delete
  const handleDelete = useCallback(async () => {
    if (
      confirm(`Are you sure you want to delete this ${config.resourceName}?`)
    ) {
      await destroy(entityId);
      onDelete?.();
    }
  }, [config.resourceName, destroy, entityId, onDelete]);

  return {
    entity,
    loading,
    error,
    refetch,
    handleDelete,
    destroyLoading,
  };
}

