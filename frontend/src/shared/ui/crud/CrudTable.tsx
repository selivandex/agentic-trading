/** @format */

"use client";

import type { Key } from "react";
import { useCallback, useMemo } from "react";
import { Edit01, Plus, Trash01 } from "@untitledui/icons";
import { Button } from "@/components/base/buttons/button";
import { Dropdown } from "@/components/base/dropdown/dropdown";
import { Input } from "@/components/base/input/input";
import { Table, TableCard } from "@/components/application/table/table";
import { Skeleton } from "@/shared/ui/skeleton/skeleton";
import { useCrudContext } from "@/shared/lib/crud/context";
import { useCrudListQuery } from "@/shared/lib/crud/use-crud-query";
import { useCrudMutations } from "@/shared/lib/crud/use-crud-mutations";
import type { CrudEntity } from "@/shared/lib/crud/types";

/**
 * CRUD Table Component
 * Displays list of entities with actions
 */
export function CrudTable<TEntity extends CrudEntity = CrudEntity>() {
  const { config, state, actions } = useCrudContext<TEntity>();

  // Fetch data
  const { entities, loading, error, refetch } = useCrudListQuery<TEntity>(
    config,
    state
  );

  // Mutations
  const { destroy, destroyLoading } = useCrudMutations<TEntity>(config);

  // Handle selection change
  const handleSelectionChange = useCallback(
    (keys: Set<Key>) => {
      const selectedIds = Array.from(keys) as string[];
      const selected = entities.filter((e) => selectedIds.includes(e.id));
      actions.setSelectedEntities(selected);
    },
    [entities, actions]
  );

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

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <h2 className="text-lg font-semibold text-primary">
          Error loading data
        </h2>
        <p className="mt-2 text-sm text-secondary">
          {config.errorMessage ?? error.message ?? "Something went wrong"}
        </p>
        <Button onClick={() => refetch()} size="lg" className="mt-4">
          Try Again
        </Button>
      </div>
    );
  }

  return (
    <TableCard.Root>
      <TableCard.Header
        title={config.resourceNamePlural}
        badge={entities.length}
        contentTrailing={
          <div className="flex items-center gap-3">
            {config.enableSearch && (
              <Input
                type="search"
                placeholder="Search..."
                value={state.searchQuery ?? ""}
                onChange={handleSearchChange}
                className="w-64"
              />
            )}
            <Button
              size="md"
              iconLeading={Plus}
              onClick={() => actions.goToNew()}
            >
              New {config.resourceName}
            </Button>
          </div>
        }
      />

      {loading && !entities.length ? (
        <div className="space-y-3 p-6">
          <div className="flex items-center justify-between">
            <Skeleton className="h-5 w-32" />
            <Skeleton className="h-5 w-24" />
          </div>
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="flex items-center gap-4 py-4">
              <Skeleton className="h-10 flex-1" />
              <Skeleton className="h-10 w-24" />
              <Skeleton className="h-10 w-32" />
              <Skeleton className="h-10 w-20" />
            </div>
          ))}
        </div>
      ) : entities.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-12">
          <h2 className="text-lg font-semibold text-primary">
            No {config.resourceNamePlural.toLowerCase()}
          </h2>
          <p className="mt-2 text-sm text-secondary">
            {config.emptyStateMessage ??
              `Get started by creating a new ${config.resourceName.toLowerCase()}`}
          </p>
          <Button
            size="lg"
            iconLeading={Plus}
            onClick={() => actions.goToNew()}
            className="mt-4"
          >
            New {config.resourceName}
          </Button>
        </div>
      ) : (
        <Table
          aria-label={`${config.resourceNamePlural} table`}
          selectionMode={config.enableSelection ? "multiple" : "none"}
          selectionBehavior={config.enableSelection ? "toggle" : undefined}
          selectedKeys={new Set(state.selectedEntities.map((e) => e.id))}
          onSelectionChange={(keys) => handleSelectionChange(keys as Set<Key>)}
          sortDescriptor={
            state.sort
              ? {
                  column: state.sort.column,
                  direction:
                    state.sort.direction === "asc"
                      ? ("ascending" as const)
                      : ("descending" as const),
                }
              : undefined
          }
          onSortChange={handleSortChange}
        >
          <Table.Header columns={columns}>
            {(column) => (
              <Table.Head
                key={column.key}
                label={column.name}
                allowsSorting={column.allowsSorting}
              />
            )}
          </Table.Header>

          <Table.Body items={rows}>
            {(row) => (
              <Table.Row key={row.id} columns={columns}>
                {(column) => {
                  const columnConfig = config.columns.find(
                    (c) => c.key === column.key
                  );

                  return (
                    <Table.Cell key={column.key}>
                      {columnConfig?.render
                        ? columnConfig.render(row as TEntity)
                        : String(row[column.key] ?? "")}

                      {/* Actions column */}
                      {column.key === "actions" && (
                        <div className="flex items-center gap-2">
                          <Button
                            size="sm"
                            color="secondary"
                            iconLeading={Edit01}
                            onClick={() => actions.goToEdit(row.id)}
                          >
                            Edit
                          </Button>

                          {config.actions && config.actions.length > 0 && (
                            <Dropdown.Root>
                              <Dropdown.DotsButton />
                              <Dropdown.Popover className="w-min">
                                <Dropdown.Menu>
                                  {config.actions
                                    .filter(
                                      (action) =>
                                        !action.hidden ||
                                        !action.hidden(row as TEntity)
                                    )
                                    .map((action) => (
                                      <Dropdown.Item
                                        key={action.key}
                                        icon={action.icon}
                                        onAction={() =>
                                          action.onClick(row as TEntity)
                                        }
                                        isDisabled={
                                          action.disabled
                                            ? action.disabled(row as TEntity)
                                            : false
                                        }
                                        className={
                                          action.destructive
                                            ? "text-error"
                                            : undefined
                                        }
                                      >
                                        {action.label}
                                      </Dropdown.Item>
                                    ))}

                                  <Dropdown.Item
                                    icon={Trash01}
                                    onAction={() =>
                                      handleDelete(row as TEntity)
                                    }
                                    isDisabled={destroyLoading}
                                    className="text-error"
                                  >
                                    Delete
                                  </Dropdown.Item>
                                </Dropdown.Menu>
                              </Dropdown.Popover>
                            </Dropdown.Root>
                          )}
                        </div>
                      )}
                    </Table.Cell>
                  );
                }}
              </Table.Row>
            )}
          </Table.Body>
        </Table>
      )}
    </TableCard.Root>
  );
}
