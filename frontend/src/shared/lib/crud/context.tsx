/** @format */

"use client";

import type { ReactNode } from "react";
import { createContext, useContext, useMemo, useState } from "react";
import type { CrudActions, CrudConfig, CrudEntity, CrudState } from "./types";

/**
 * CRUD Context
 */
interface CrudContextValue<TEntity extends CrudEntity = CrudEntity> {
  config: CrudConfig<TEntity>;
  state: CrudState<TEntity>;
  actions: CrudActions<TEntity>;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const CrudContext = createContext<CrudContextValue<any> | null>(null);

/**
 * Hook to access CRUD context
 */
export function useCrudContext<TEntity extends CrudEntity = CrudEntity>() {
  const context = useContext(CrudContext);
  if (!context) {
    throw new Error("useCrudContext must be used within CrudProvider");
  }
  return context as unknown as CrudContextValue<TEntity>;
}

/**
 * CRUD Provider props
 */
interface CrudProviderProps<TEntity extends CrudEntity = CrudEntity> {
  config: CrudConfig<TEntity>;
  children: ReactNode;
  /** Initial mode */
  initialMode?: CrudState<TEntity>["mode"];
  /** Initial entity ID (for show/edit modes) */
  _initialEntityId?: string;
}

/**
 * CRUD Provider component
 * Manages CRUD state and provides actions
 */
export function CrudProvider<TEntity extends CrudEntity = CrudEntity>({
  config,
  children,
  initialMode = "index",
  _initialEntityId,
}: CrudProviderProps<TEntity>) {
  // State
  const [mode, setMode] = useState<CrudState<TEntity>["mode"]>(initialMode);
  const [currentEntity, setCurrentEntity] = useState<TEntity | null>(null);
  const [selectedEntities, setSelectedEntities] = useState<TEntity[]>([]);
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(config.defaultPageSize ?? 20);
  const [sort, setSort] = useState<CrudState<TEntity>["sort"]>();
  const [searchQuery, setSearchQuery] = useState<string>();
  const [cursors, setCursors] = useState<CrudState<TEntity>["cursors"]>({});
  const [totalCount, setTotalCount] = useState<number>();
  const [pageInfo, setPageInfo] = useState<CrudState<TEntity>["pageInfo"]>();

  // State object
  const state = useMemo<CrudState<TEntity>>(
    () => ({
      mode,
      currentEntity,
      selectedEntities,
      filters,
      page,
      pageSize,
      sort,
      searchQuery,
      cursors,
      totalCount,
      pageInfo,
    }),
    [
      mode,
      currentEntity,
      selectedEntities,
      filters,
      page,
      pageSize,
      sort,
      searchQuery,
      cursors,
      totalCount,
      pageInfo,
    ]
  );

  // Actions
  const actions = useMemo<CrudActions<TEntity>>(
    () => ({
      goToIndex: () => {
        setMode("index");
        setCurrentEntity(null);
      },
      goToShow: (_id: string) => {
        setMode("show");
        // Entity will be loaded by the Show component
      },
      goToNew: () => {
        setMode("new");
        setCurrentEntity(null);
      },
      goToEdit: (_id: string) => {
        setMode("edit");
        // Entity will be loaded by the Edit component
      },
      create: async (_data: Record<string, unknown>) => {
        // Implemented by useCrudMutations hook
        throw new Error("Create not implemented");
      },
      update: async (_id: string, _data: Record<string, unknown>) => {
        // Implemented by useCrudMutations hook
        throw new Error("Update not implemented");
      },
      destroy: async (_id: string) => {
        // Implemented by useCrudMutations hook
        throw new Error("Destroy not implemented");
      },
      setSelectedEntities,
      setFilters,
      setPage,
      setPageSize,
      setSort: (column: string, direction: "asc" | "desc") => {
        setSort({ column, direction });
      },
      setSearchQuery,
      refresh: async () => {
        // Trigger refetch - implemented by components
      },
      updatePaginationState: (
        newPageInfo?: typeof pageInfo,
        newTotalCount?: number
      ) => {
        if (newPageInfo) setPageInfo(newPageInfo);
        if (newTotalCount !== undefined) setTotalCount(newTotalCount);
      },
    }),
    []
  );

  const value = useMemo<CrudContextValue<TEntity>>(
    () => ({
      config,
      state,
      actions,
    }),
    [config, state, actions]
  );

  return <CrudContext.Provider value={value}>{children}</CrudContext.Provider>;
}
