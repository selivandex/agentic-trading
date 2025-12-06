/** @format */

"use client";

import type { ReactNode } from "react";
import { createContext, useContext, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
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
  const [page] = useState(1); // Keep for state interface compatibility
  const [pageSize] = useState(config.defaultPageSize ?? 20);
  const [sort, setSort] = useState<CrudState<TEntity>["sort"]>();
  const [searchQuery, setSearchQuery] = useState<string>();
  const [cursors, setCursors] = useState<CrudState<TEntity>["cursors"]>({});
  const [totalCount, setTotalCount] = useState<number>();
  const [pageInfo, setPageInfo] = useState<CrudState<TEntity>["pageInfo"]>();

  // Next.js router for navigation
  const router = useRouter();

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
        if (config.basePath) {
          router.push(config.basePath);
        } else {
          setMode("index");
          setCurrentEntity(null);
        }
      },
      goToShow: (id: string) => {
        if (config.basePath) {
          router.push(`${config.basePath}/${id}`);
        } else {
          setMode("show");
        }
      },
      goToNew: () => {
        if (config.basePath) {
          router.push(`${config.basePath}/new`);
        } else {
          setMode("new");
          setCurrentEntity(null);
        }
      },
      goToEdit: (id: string) => {
        if (config.basePath) {
          router.push(`${config.basePath}/${id}/edit`);
        } else {
          setMode("edit");
        }
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
      setPage: (_newPage: number) => {
        // Not used in Relay pagination - use goToNextPage/goToPrevPage instead
      },
      setPageSize: (_newPageSize: number) => {
        // Not implemented yet
      },
      goToNextPage: () => {
        // This will be overridden in CrudList with fetchMore
      },
      goToPrevPage: () => {
        // Not implemented for Load More pattern
      },
      goToFirstPage: () => {
        setCursors({});
      },
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
        if (newPageInfo) {
          setPageInfo(newPageInfo);
          // Update cursors for next/prev navigation
          setCursors({
            after: newPageInfo.endCursor,
            before: newPageInfo.startCursor,
          });
        }
        if (newTotalCount !== undefined) setTotalCount(newTotalCount);
      },
    }),
    [
      config.basePath,
      router,
      setMode,
      setCurrentEntity,
      setSelectedEntities,
      setFilters,
      setCursors,
      setSort,
      setSearchQuery,
      setPageInfo,
      setTotalCount,
    ]
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
