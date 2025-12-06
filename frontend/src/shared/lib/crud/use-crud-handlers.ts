/** @format */

"use client";

import { useCallback } from "react";
import type { Key } from "react";
import type { CrudConfig, CrudEntity, CrudActions } from "./types";
import { useCrudMutations } from "./use-crud-mutations";

/**
 * Hook for CRUD action handlers
 * Handles sort, search, delete operations
 */
export function useCrudHandlers<TEntity extends CrudEntity>(
  config: CrudConfig<TEntity>,
  actions: CrudActions<TEntity>,
  refetch: () => Promise<void>
) {
  // Mutations
  const { destroy, destroyLoading } = useCrudMutations<TEntity>(config);

  // Handle sort change
  const handleSortChange = useCallback(
    (descriptor: { column: Key; direction: "ascending" | "descending" }) => {
      actions.setSort(
        descriptor.column as string,
        descriptor.direction === "ascending" ? "asc" : "desc"
      );
    },
    [actions]
  );

  // Handle delete
  const handleDelete = useCallback(
    async (entity: TEntity) => {
      if (
        confirm(`Are you sure you want to delete this ${config.resourceName}?`)
      ) {
        await destroy(entity.id);
        await refetch();
      }
    },
    [config, destroy, refetch]
  );

  // Handle search
  const handleSearchChange = useCallback(
    (value: string | React.ChangeEvent<HTMLInputElement>) => {
      const searchValue =
        typeof value === "string" ? value : value.target.value;
      actions.setSearchQuery(searchValue);
    },
    [actions]
  );

  return {
    handleSortChange,
    handleDelete,
    handleSearchChange,
    destroyLoading,
  };
}

