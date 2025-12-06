/** @format */

"use client";

import { useCallback } from "react";
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

      <div className={paddingClasses}>
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
    </div>
  );
}
