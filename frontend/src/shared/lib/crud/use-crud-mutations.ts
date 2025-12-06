/** @format */

"use client";

import { useMutation } from "@apollo/client";
import { useCallback } from "react";
import { toast } from "sonner";
import type { CrudConfig, CrudEntity } from "./types";
import { get } from "./utils";

/**
 * Hook for CRUD mutations (create, update, delete)
 */
export function useCrudMutations<TEntity extends CrudEntity = CrudEntity>(
  config: CrudConfig<TEntity>,
) {
  // Create mutation
  const [createMutation, createState] = useMutation(
    config.graphql.create.mutation,
  );

  const create = useCallback(
    async (data: Record<string, unknown>) => {
      try {
        // Transform data if needed
        const transformedData = config.transformBeforeCreate
          ? config.transformBeforeCreate(data)
          : data;

        const variables = {
          ...config.graphql.create.variables,
          input: transformedData,
        };

        const result = await createMutation({
          variables,
          // Refetch list query to update cache
          refetchQueries: [config.graphql.list.query],
        });

        const entity = get(
          result.data,
          config.graphql.create.dataPath,
        ) as TEntity;

        toast.success(`${config.resourceName} created successfully`);
        return entity;
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Failed to create";
        toast.error(message);
        throw error;
      }
    },
    [config, createMutation],
  );

  // Update mutation
  const [updateMutation, updateState] = useMutation(
    config.graphql.update.mutation,
  );

  const update = useCallback(
    async (id: string, data: Record<string, unknown>) => {
      try {
        // Transform data if needed
        const transformedData = config.transformBeforeUpdate
          ? config.transformBeforeUpdate(data)
          : data;

        const variables = {
          ...config.graphql.update.variables,
          id,
          input: transformedData,
        };

        const result = await updateMutation({
          variables,
          // Refetch queries to update cache
          refetchQueries: [config.graphql.list.query, config.graphql.show.query],
        });

        const entity = get(
          result.data,
          config.graphql.update.dataPath,
        ) as TEntity;

        toast.success(`${config.resourceName} updated successfully`);
        return entity;
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Failed to update";
        toast.error(message);
        throw error;
      }
    },
    [config, updateMutation],
  );

  // Delete mutation
  const [destroyMutation, destroyState] = useMutation(
    config.graphql.destroy.mutation,
  );

  const destroy = useCallback(
    async (id: string) => {
      try {
        const variables = {
          ...config.graphql.destroy.variables,
          id,
        };

        await destroyMutation({
          variables,
          // Refetch list query to update cache
          refetchQueries: [config.graphql.list.query],
        });

        toast.success(`${config.resourceName} deleted successfully`);
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Failed to delete";
        toast.error(message);
        throw error;
      }
    },
    [config, destroyMutation],
  );

  return {
    create,
    createLoading: createState.loading,
    createError: createState.error,

    update,
    updateLoading: updateState.loading,
    updateError: updateState.error,

    destroy,
    destroyLoading: destroyState.loading,
    destroyError: destroyState.error,

    isLoading:
      createState.loading || updateState.loading || destroyState.loading,
  };
}
