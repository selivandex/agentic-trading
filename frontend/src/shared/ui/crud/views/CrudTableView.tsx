/** @format */

"use client";

import type { Key } from "react";
import React from "react";
import { Edit01, Trash01, Eye } from "@untitledui/icons";
import { Table } from "@/components/application/table/table";
import { Dropdown } from "@/components/base/dropdown/dropdown";
import type { CrudEntity, CrudConfig, CrudState } from "@/shared/lib/crud/types";

/**
 * Table presentation component for CRUD
 * Pure presentational - no business logic
 */
interface CrudTableViewProps<TEntity extends CrudEntity> {
  config: CrudConfig<TEntity>;
  state: CrudState<TEntity>;
  rows: TEntity[];
  columns: Array<{ key: string; name: string; allowsSorting?: boolean }>;
  onSelectionChange: (keys: "all" | Set<Key>) => void;
  onSortChange: (descriptor: {
    column: Key;
    direction: "ascending" | "descending";
  }) => void;
  onEdit: (id: string) => void;
  onShow: (id: string) => void;
  onDelete: (entity: TEntity) => void;
  destroyLoading: boolean;
}

export function CrudTableView<TEntity extends CrudEntity>({
  config,
  state,
  rows,
  columns,
  onSelectionChange,
  onSortChange,
  onEdit,
  onShow,
  onDelete,
  destroyLoading,
}: CrudTableViewProps<TEntity>) {
  return (
    <Table
      aria-label={`${config.resourceNamePlural} table`}
      selectionMode={config.enableSelection ? "multiple" : "none"}
      selectionBehavior={config.enableSelection ? "toggle" : "replace"}
      selectedKeys={
        state.selectedEntities.length > 0
          ? new Set(state.selectedEntities.map((e) => e.id))
          : new Set()
      }
      onSelectionChange={onSelectionChange}
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
      onSortChange={onSortChange}
    >
      <Table.Header columns={columns}>
        {(column) => (
          <Table.Head
            key={column.key}
            label={column.name}
            allowsSorting={column.allowsSorting}
            isRowHeader={column.key === columns[0]?.key}
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
                    <div className="flex items-center justify-end">
                      <Dropdown.Root>
                        <Dropdown.DotsButton />
                        <Dropdown.Popover className="w-min">
                          <Dropdown.Menu>
                            {/* Show action */}
                            <Dropdown.Item icon={Eye} onAction={() => onShow(row.id)}>
                              View
                            </Dropdown.Item>

                            {/* Edit action */}
                            <Dropdown.Item icon={Edit01} onAction={() => onEdit(row.id)}>
                              Edit
                            </Dropdown.Item>

                            {/* Custom actions */}
                            {config.actions &&
                              config.actions.length > 0 &&
                              config.actions
                                .filter(
                                  (action) =>
                                    !action.hidden || !action.hidden(row as TEntity)
                                )
                                .map((action) => (
                                  <Dropdown.Item
                                    key={action.key}
                                    icon={action.icon}
                                    onAction={() => action.onClick(row as TEntity)}
                                    isDisabled={
                                      action.disabled
                                        ? action.disabled(row as TEntity)
                                        : false
                                    }
                                    className={
                                      action.destructive ? "text-error" : undefined
                                    }
                                  >
                                    {action.label}
                                  </Dropdown.Item>
                                ))}

                            {/* Delete action (only if destroy mutation is configured) */}
                            {config.graphql.destroy && (
                              <Dropdown.Item
                                icon={Trash01}
                                onAction={() => onDelete(row as TEntity)}
                                isDisabled={destroyLoading}
                                className="text-error"
                              >
                                Delete
                              </Dropdown.Item>
                            )}
                          </Dropdown.Menu>
                        </Dropdown.Popover>
                      </Dropdown.Root>
                    </div>
                  )}
                </Table.Cell>
              );
            }}
          </Table.Row>
        )}
      </Table.Body>
    </Table>
  );
}

