/** @format */

"use client";

import type { ReactNode } from "react";
import { createContext, useContext, useMemo, useState, useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";
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
  const router = useRouter();
  const searchParams = useSearchParams();

  // Initialize state from URL params
  const [mode, setMode] = useState<CrudState<TEntity>["mode"]>(initialMode);
  const [currentEntity, setCurrentEntity] = useState<TEntity | null>(null);
  const [selectedEntities, setSelectedEntities] = useState<TEntity[]>([]);
  const [filters, setFiltersState] = useState<Record<string, unknown>>(() => {
    // Parse filters from URL on mount
    const filtersParam = searchParams.get("filters");
    if (filtersParam) {
      try {
        return JSON.parse(decodeURIComponent(filtersParam));
      } catch {
        return {};
      }
    }
    return {};
  });
  const [page] = useState(1);
  const [pageSize] = useState(config.defaultPageSize ?? 20);
  const [sort, setSort] = useState<CrudState<TEntity>["sort"]>(() => {
    const sortBy = searchParams.get("sortBy");
    const sortDirection = searchParams.get("sortDirection");
    if (sortBy && sortDirection) {
      return { column: sortBy, direction: sortDirection as "asc" | "desc" };
    }
    return undefined;
  });
  const [searchQuery, setSearchQuery] = useState<string | undefined>(
    searchParams.get("search") || undefined
  );
  const [cursors, setCursors] = useState<CrudState<TEntity>["cursors"]>({});
  const [totalCount, setTotalCount] = useState<number>();
  const [pageInfo, setPageInfo] = useState<CrudState<TEntity>["pageInfo"]>();
  const [activeTab, setActiveTabState] = useState<string | undefined>(
    searchParams.get("tab") || undefined
  );

  // Update URL when filters/search/sort/tab change
  useEffect(() => {
    if (mode !== "index") return; // Only sync URL for list view

    const params = new URLSearchParams();

    // Add filters to URL
    if (Object.keys(filters).length > 0) {
      params.set("filters", encodeURIComponent(JSON.stringify(filters)));
    }

    // Add search to URL
    if (searchQuery) {
      params.set("search", searchQuery);
    }

    // Add sort to URL
    if (sort) {
      params.set("sortBy", sort.column);
      params.set("sortDirection", sort.direction);
    }

    // Add active tab to URL
    if (activeTab) {
      params.set("tab", activeTab);
    }

    // Update URL without causing a full page reload
    const newUrl = params.toString()
      ? `${window.location.pathname}?${params.toString()}`
      : window.location.pathname;

    router.replace(newUrl, { scroll: false });
  }, [filters, searchQuery, sort, activeTab, mode, router]);

  // Wrapper for setFilters that updates state
  const setFilters = (newFilters: Record<string, unknown>) => {
    setFiltersState(newFilters);
  };

  // Wrapper for setActiveTab
  const setActiveTab = (tabKey: string) => {
    setActiveTabState(tabKey);
    // Reset pagination when changing tabs
    setCursors({});
    setSelectedEntities([]);
  };

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
      activeTab,
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
      activeTab,
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
      setActiveTab,
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
      setActiveTab,
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
