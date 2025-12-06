/** @format */

"use client";

import { useMemo } from "react";
import type { CrudConfig, CrudEntity, CrudState } from "./types";
import { useCrudListQuery } from "./use-crud-query";

/**
 * Hook for CRUD list logic
 * Handles data fetching, pagination, sorting
 */
export function useCrudList<TEntity extends CrudEntity>(
  config: CrudConfig<TEntity>,
  state: CrudState<TEntity>
) {
  // Fetch data
  const { entities, pageInfo, totalCount, loading, error, refetch, fetchMore } =
    useCrudListQuery<TEntity>(config, state);

  // Table columns
  const columns = useMemo(() => {
    return config.columns.map((col) => ({
      key: col.key,
      name: col.label,
      allowsSorting: col.sortable,
    }));
  }, [config.columns]);

  // Table rows
  const rows = useMemo(() => {
    return entities;
  }, [entities]);

  return {
    entities,
    rows,
    columns,
    pageInfo,
    totalCount,
    loading,
    error,
    refetch,
    fetchMore,
  };
}

