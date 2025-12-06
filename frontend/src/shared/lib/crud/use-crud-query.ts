/** @format */

"use client";

import { useQuery } from "@apollo/client";
import { useMemo } from "react";
import type { CrudConfig, CrudEntity, CrudState, Connection } from "./types";
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

    // Resolve global scope variables from config
    const configScopeVariables =
      typeof config.graphql.list.scope === "function"
        ? config.graphql.list.scope()
        : config.graphql.list.scope ?? {};

    // Add active tab filter variable if tabs are enabled
    const tabFilterVariables: Record<string, unknown> = {};
    if (config.tabs?.enabled && state.activeTab) {
      const filterVariable = config.tabs.filterVariable ?? "scope";
      tabFilterVariables[filterVariable] = state.activeTab;
    }

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

      // Add filters as a single JSONObject parameter
      const filterVariables =
        Object.keys(state.filters).length > 0
          ? { filters: state.filters }
          : {};

      return {
        ...baseVariables,
        ...configScopeVariables,
        ...tabFilterVariables, // Tab filter
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

      // Add filters as a single JSONObject parameter
      const filterVariables =
        Object.keys(state.filters).length > 0
          ? { filters: state.filters }
          : {};

      return {
        ...baseVariables,
        ...configScopeVariables,
        ...tabFilterVariables, // Tab filter
        ...paginationVariables,
        ...sortVariables,
        ...searchVariables,
        ...filterVariables,
      };
    }
  }, [config, state, useConnection]);

  const { data, loading, error, refetch, fetchMore } = useQuery(
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
        scopes: undefined,
        filters: undefined,
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
          scopes: undefined,
          filters: undefined,
        };
      }

      return {
        entities: connection.edges.map((edge) => edge.node),
        pageInfo: connection.pageInfo,
        totalCount: connection.totalCount,
        scopes: connection.scopes,
        filters: connection.filters,
      };
    } else {
      // Legacy array response
      return {
        entities: get(data, config.graphql.list.dataPath, []) as TEntity[],
        pageInfo: undefined,
        totalCount: undefined,
        scopes: undefined,
        filters: undefined,
      };
    }
  }, [data, config.graphql.list.dataPath, useConnection]);

  return {
    ...result,
    loading,
    error,
    refetch,
    fetchMore,
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
  const variables = useMemo(() => {
    // Resolve scope variables (can be object or function)
    const scopeVariables =
      typeof config.graphql.show.scope === "function"
        ? config.graphql.show.scope()
        : config.graphql.show.scope ?? {};

    return {
      ...config.graphql.show.variables,
      ...scopeVariables,
      id,
    };
  }, [config, id]);

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
