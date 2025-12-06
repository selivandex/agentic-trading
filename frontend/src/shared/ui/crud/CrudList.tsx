/** @format */

"use client";

import { useCallback, useEffect, useState } from "react";
import { TableCard } from "@/components/application/table/table";
import { Button } from "@/shared/base/buttons/button";
import { useCrudContext } from "@/shared/lib/crud/context";
import { useCrudList } from "@/shared/lib/crud/use-crud-list";
import { useCrudSelection } from "@/shared/lib/crud/use-crud-selection";
import { useCrudBatchActions } from "@/shared/lib/crud/use-crud-batch-actions";
import { useCrudHandlers } from "@/shared/lib/crud/use-crud-handlers";
import { useCrudBreadcrumbs } from "@/shared/lib/crud/use-crud-breadcrumbs";
import type { CrudEntity } from "@/shared/lib/crud/types";
import { PageHeader } from "@/shared/ui/page-header";
import { CrudTableView } from "./views/CrudTableView";
import { CrudBatchActionsToolbar } from "./views/CrudBatchActionsToolbar";
import { CrudLoadingState } from "./views/CrudLoadingState";
import { CrudEmptyState } from "./views/CrudEmptyState";
import { CrudErrorState } from "./views/CrudErrorState";
import { CrudPagination } from "./views/CrudPagination";
import { CrudTabs } from "./views/CrudTabs";
import { CrudDynamicFilters } from "./views/CrudDynamicFilters";

/**
 * CRUD List Component
 * Container that orchestrates hooks and presentation components
 */
interface CrudListProps<_TEntity extends CrudEntity = CrudEntity> {
  /** View style: table, grid, cards, etc. */
  style?: "table" | "grid" | "cards";
}

export function CrudList<TEntity extends CrudEntity = CrudEntity>({
  style = "table",
}: CrudListProps<TEntity>) {
  const { config, state, actions } = useCrudContext<TEntity>();

  // Data fetching logic
  const {
    entities,
    rows,
    columns,
    pageInfo,
    totalCount,
    scopes,
    filters,
    loading,
    error,
    refetch,
    fetchMore,
  } = useCrudList<TEntity>(config, state);

  // Selection logic
  const { handleSelectionChange, clearSelection } = useCrudSelection<TEntity>(
    entities,
    actions
  );

  // Wrap refetch to match expected signature
  const wrappedRefetch = useCallback(async () => {
    await refetch();
  }, [refetch]);

  // Batch actions logic
  const { executeBatchAction } = useCrudBatchActions<TEntity>(
    config,
    state.selectedEntities,
    actions,
    wrappedRefetch
  );

  // Action handlers
  const { handleSortChange, handleDelete, handleSearchChange, destroyLoading } =
    useCrudHandlers<TEntity>(config, actions, wrappedRefetch);

  // Breadcrumbs
  const breadcrumbs = useCrudBreadcrumbs<TEntity>(config, "index");

  // Auto-select default scope when scopes are loaded
  useEffect(() => {
    if (
      config.tabs?.enabled &&
      scopes &&
      scopes.length > 0 &&
      !state.activeTab
    ) {
      // Find configured default scope or use first scope
      const defaultScopeId = config.tabs.defaultScope;
      const defaultScope = defaultScopeId
        ? scopes.find((s) => s.id === defaultScopeId)
        : scopes[0];

      if (defaultScope) {
        actions.setActiveTab(defaultScope.id);
      }
    }
  }, [scopes, state.activeTab, config.tabs?.enabled, config.tabs?.defaultScope, actions]);

  // Load more handler
  const handleLoadMore = useCallback(async () => {
    if (pageInfo?.endCursor && fetchMore) {
      await fetchMore({
        variables: {
          after: pageInfo.endCursor,
          first: state.pageSize,
        },
        updateQuery: (prev, { fetchMoreResult }) => {
          if (!fetchMoreResult) return prev;

          const prevConnection = prev[config.graphql.list.dataPath];
          const newConnection = fetchMoreResult[config.graphql.list.dataPath];

          return {
            ...prev,
            [config.graphql.list.dataPath]: {
              ...newConnection,
              edges: [...prevConnection.edges, ...newConnection.edges],
            },
          };
        },
      });
    }
  }, [pageInfo, fetchMore, state.pageSize, config.graphql.list.dataPath]);

  // Error state
  if (error) {
    return (
      <CrudErrorState config={config} error={error} onRetry={() => refetch()} />
    );
  }

  const hasSelection = state.selectedEntities.length > 0;
  const paddingClasses = "px-4 lg:px-8";
  const hasFilters =
    config.dynamicFilters?.enabled && filters && filters.length > 0;

  // Draft filters (not applied yet)
  const [draftFilters, setDraftFilters] = useState<Record<string, unknown>>(
    state.filters
  );

  // Sync draft filters when state.filters change externally (e.g., from URL)
  useEffect(() => {
    setDraftFilters(state.filters);
  }, [state.filters]);

  // Apply filters handler
  const handleApplyFilters = useCallback(() => {
    actions.setFilters(draftFilters);
  }, [draftFilters, actions]);

  // Clear filters handler
  const handleClearFilters = useCallback(() => {
    setDraftFilters({});
    actions.setFilters({});
  }, [actions]);

  // Check if there are pending filter changes
  const hasFilterChanges =
    JSON.stringify(draftFilters) !== JSON.stringify(state.filters);

  return (
    <div className="mx-auto mb-8 flex flex-col gap-5">
      {/* Page Header with Breadcrumbs, Title, and Actions */}
      <PageHeader
        background="transparent"
        breadcrumbs={breadcrumbs}
        showBreadcrumbs={!!breadcrumbs && breadcrumbs.length > 0}
        title={config.resourceNamePlural}
        description={`Manage your ${config.resourceNamePlural.toLowerCase()}`}
        search={
          config.enableSearch !== false
            ? {
                placeholder: `Search ${config.resourceNamePlural.toLowerCase()}...`,
                value: state.searchQuery ?? "",
                onChange: handleSearchChange,
              }
            : undefined
        }
      >
        <PageHeader.Actions>
          <Button color="primary" size="md" onClick={() => actions.goToNew()}>
            New {config.resourceName}
          </Button>
        </PageHeader.Actions>
      </PageHeader>

      {/* Tabs for filtering (rendered if backend provides scopes) */}
      {config.tabs?.enabled && scopes && scopes.length > 0 && (
        <div className={paddingClasses}>
          <CrudTabs
            config={config.tabs}
            activeScope={state.activeTab}
            scopes={scopes}
            onScopeChange={actions.setActiveTab}
          />
        </div>
      )}

      {/* Main content with optional sidebar */}
      <div className={paddingClasses}>
        <div className={hasFilters ? "flex gap-6" : ""}>
          {/* Main content area */}
          <div className={hasFilters ? "flex-1 min-w-0" : "w-full"}>
            <TableCard.Root>
              {/* Batch Actions Toolbar */}
              {hasSelection && (
                <CrudBatchActionsToolbar
                  config={config}
                  selectedCount={state.selectedEntities.length}
                  selectedEntities={state.selectedEntities}
                  onClearSelection={clearSelection}
                  onExecuteAction={executeBatchAction}
                />
              )}

              {/* Loading State */}
              {loading && !entities.length ? (
                <CrudLoadingState />
              ) : entities.length === 0 ? (
                /* Empty State */
                <CrudEmptyState config={config} onNew={() => actions.goToNew()} />
              ) : (
                /* Data View */
                <>
                  {style === "table" && (
                    <CrudTableView
                      config={config}
                      state={state}
                      rows={rows}
                      columns={columns}
                      onSelectionChange={handleSelectionChange}
                      onSortChange={handleSortChange}
                      onEdit={(id) => actions.goToEdit(id)}
                      onShow={(id) => actions.goToShow(id)}
                      onDelete={handleDelete}
                      destroyLoading={destroyLoading}
                    />
                  )}
                  {/* Future: Add grid view (style="grid"), cards view (style="cards"), etc. */}
                </>
              )}
            </TableCard.Root>

            {/* Pagination */}
            {entities.length > 0 && (
              <CrudPagination
                pageInfo={pageInfo}
                totalCount={totalCount}
                currentCount={entities.length}
                pageSize={state.pageSize}
                loading={loading}
                onLoadMore={handleLoadMore}
              />
            )}
          </div>

          {/* Right Sidebar - Dynamic Filters */}
          {hasFilters && (
            <aside className="w-80 flex-shrink-0">
              <div className="sticky top-4">
                <div className="rounded-lg border border-border-secondary bg-primary p-6">
                  <h3 className="mb-4 text-lg font-semibold text-secondary">
                    Filters
                  </h3>
                  <div className="space-y-4">
                    <CrudDynamicFilters
                      config={config.dynamicFilters!}
                      filters={filters}
                      values={draftFilters}
                      onChange={(filterId, value) => {
                        setDraftFilters({
                          ...draftFilters,
                          [filterId]: value,
                        });
                      }}
                    />
                    <div className="flex gap-2 border-t border-border-secondary pt-4">
                      <Button
                        color="secondary"
                        size="sm"
                        onClick={handleClearFilters}
                        className="flex-1"
                        disabled={Object.keys(draftFilters).length === 0}
                      >
                        Clear
                      </Button>
                      <Button
                        color="primary"
                        size="sm"
                        onClick={handleApplyFilters}
                        className="flex-1"
                        disabled={!hasFilterChanges}
                      >
                        Apply
                      </Button>
                    </div>
                  </div>
                </div>
              </div>
            </aside>
          )}
        </div>
      </div>
    </div>
  );
}
