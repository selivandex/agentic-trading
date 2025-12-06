/** @format */

"use client";

import React from "react";
import { X } from "@untitledui/icons";
import { Button } from "@/components/base/buttons/button";
import type { CrudConfig, CrudEntity } from "@/shared/lib/crud/types";

/**
 * Batch actions toolbar presentation component
 * Shows when entities are selected
 */
interface CrudBatchActionsToolbarProps<TEntity extends CrudEntity> {
  config: CrudConfig<TEntity>;
  selectedCount: number;
  selectedEntities: TEntity[];
  onClearSelection: () => void;
  onExecuteAction: (actionKey: string) => void;
}

export function CrudBatchActionsToolbar<TEntity extends CrudEntity>({
  config,
  selectedCount,
  selectedEntities,
  onClearSelection,
  onExecuteAction,
}: CrudBatchActionsToolbarProps<TEntity>) {
  if (!config.bulkActions || config.bulkActions.length === 0) {
    return null;
  }

  return (
    <div className="flex items-center justify-between gap-4 border-b border-border-secondary bg-utility-brand-50 px-6 py-4 dark:bg-utility-brand-900/20">
      <div className="flex items-center gap-3">
        <span className="text-sm font-medium text-primary">
          {selectedCount} selected
        </span>
        <Button
          size="sm"
          color="secondary"
          iconLeading={X}
          onClick={onClearSelection}
        >
          Clear
        </Button>
      </div>

      <div className="flex items-center gap-2">
        {config.bulkActions
          .filter((action) => {
            // Check if action should be hidden for batch
            if (action.hiddenForBatch) {
              return !action.hiddenForBatch(selectedEntities);
            }
            return true;
          })
          .map((action) => (
            <Button
              key={action.key}
              size="sm"
              color={action.destructive ? "secondary-destructive" : "secondary"}
              iconLeading={action.icon}
              onClick={() => onExecuteAction(action.key)}
              isDisabled={
                action.disabled
                  ? selectedEntities.some((entity) => action.disabled?.(entity))
                  : false
              }
            >
              {action.label}
            </Button>
          ))}
      </div>
    </div>
  );
}
