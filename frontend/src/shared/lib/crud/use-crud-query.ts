/** @format */

"use client";

import { useQuery } from "@apollo/client";
import { useMemo } from "react";
import type { CrudConfig, CrudEntity, CrudState, Connection, PageInfo } from "./types";
import { get } from "./utils";

/**
 * Hook for fetching CRUD list data with Relay pagination
 */
export function useCrudListQuery<TEntity extends CrudEntity = CrudEntity>(
  config: CrudConfig<TEntity>,
  state: CrudState<TEntity>,
  options?: {
    skip?: boolean;
  },
) {
  const useConnection = config.graphql.list.useConnection !== false;

  const variables = useMemo(() => {
    const baseVariables = config.graphql.list.variables ?? {};

    if (useConnection) {
      // Relay cursor-based pagination
      const paginationVariables: Record<string, unknown> = {
        first: state.pageSize,
      };

      // Add cursor if not on first page
      if (state.cursors?.after) {
        paginationVariables.after = state.cursors.after;
      }

      // Add sort
      const sortVariables = state.sort
        ? {
            sortBy: state.sort.column,
            sortDirection: state.sort.direction,
          }
        : {};

      // Add search
      const searchVariables = state.searchQuery
        ? {
            search: state.searchQuery,
          }
        : {};

      // Add filters
      const filterVariables = state.filters;

      return {
        ...baseVariables,
        ...paginationVariables,
        ...sortVariables,
        ...searchVariables,
        ...filterVariables,
      };
    } else {
      // Legacy offset-based pagination
      const paginationVariables = {
        limit: state.pageSize,
        offset: (state.page - 1) * state.pageSize,
      };

      const sortVariables = state.sort
        ? {
            sortBy: state.sort.column,
            sortDirection: state.sort.direction,
          }
        : {};

      const searchVariables = state.searchQuery
        ? {
            search: state.searchQuery,
          }
        : {};

      const filterVariables = state.filters;

      return {
        ...baseVariables,
        ...paginationVariables,
        ...sortVariables,
        ...searchVariables,
        ...filterVariables,
      };
    }
  }, [config, state, useConnection]);

  const { data, loading, error, refetch } = useQuery(
    config.graphql.list.query,
    {
      variables,
      skip: options?.skip,
      fetchPolicy: "cache-and-network",
    },
  );

  const result = useMemo(() => {
    if (!data) {
      return {
        entities: [] as TEntity[],
        pageInfo: undefined,
        totalCount: undefined,
      };
    }

    if (useConnection) {
      // Extract Relay connection
      const connection = get(
        data,
        config.graphql.list.dataPath,
        null,
      ) as Connection<TEntity> | null;

      if (!connection) {
        return {
          entities: [] as TEntity[],
          pageInfo: undefined,
          totalCount: undefined,
        };
      }

      return {
        entities: connection.edges.map((edge) => edge.node),
        pageInfo: connection.pageInfo,
        totalCount: connection.totalCount,
      };
    } else {
      // Legacy array response
      return {
        entities: get(data, config.graphql.list.dataPath, []) as TEntity[],
        pageInfo: undefined,
        totalCount: undefined,
      };
    }
  }, [data, config.graphql.list.dataPath, useConnection]);

  return {
    ...result,
    loading,
    error,
    refetch,
  };
}

/**
 * Hook for fetching single CRUD entity
 */
export function useCrudShowQuery<TEntity extends CrudEntity = CrudEntity>(
  config: CrudConfig<TEntity>,
  id: string,
  options?: {
    skip?: boolean;
  },
) {
  const variables = useMemo(
    () => ({
      ...config.graphql.show.variables,
      id,
    }),
    [config, id],
  );

  const { data, loading, error, refetch } = useQuery(
    config.graphql.show.query,
    {
      variables,
      skip: options?.skip || !id,
      fetchPolicy: "cache-and-network",
    },
  );

  const entity = useMemo<TEntity | null>(() => {
    if (!data) return null;
    return get(data, config.graphql.show.dataPath, null) as TEntity | null;
  }, [data, config.graphql.show.dataPath]);

  return {
    entity,
    loading,
    error,
    refetch,
  };
}
