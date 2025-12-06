/** @format */

"use client";

import React from "react";
import { ArrowLeft, Edit01, Trash01 } from "@untitledui/icons";
import { Button } from "@/components/base/buttons/button";
import type { CrudConfig, CrudEntity } from "@/shared/lib/crud/types";

/**
 * Show page header presentation component
 */
interface CrudShowHeaderProps<TEntity extends CrudEntity> {
  config: CrudConfig<TEntity>;
  entity: TEntity;
  entityId: string;
  destroyLoading: boolean;
  onBack: () => void;
  onEdit: () => void;
  onDelete: () => void;
}

export function CrudShowHeader<TEntity extends CrudEntity>({
  config,
  entity,
  entityId,
  destroyLoading,
  onBack,
  onEdit,
  onDelete,
}: CrudShowHeaderProps<TEntity>) {
  return (
    <div className="mb-6 flex items-center justify-between">
      <div className="flex items-center gap-4">
        <Button
          color="secondary"
          size="md"
          iconLeading={ArrowLeft}
          onClick={onBack}
        >
          Back
        </Button>
        <h1 className="text-2xl font-semibold text-primary">
          {config.resourceName} Details
        </h1>
      </div>

      <div className="flex items-center gap-3">
        <Button
          color="secondary"
          size="md"
          iconLeading={Edit01}
          onClick={onEdit}
        >
          Edit
        </Button>

        {/* Custom actions for show page */}
        {config.showActions &&
          config.showActions
            .filter((action) => !action.hidden || !action.hidden(entity))
            .map((action) => (
              <Button
                key={action.key}
                color={
                  action.destructive ? "secondary-destructive" : "secondary"
                }
                size="md"
                iconLeading={action.icon}
                onClick={() => action.onClick(entity)}
                isDisabled={action.disabled ? action.disabled(entity) : false}
              >
                {action.label}
              </Button>
            ))}

        {/* Delete button (only if destroy is configured) */}
        {config.graphql.destroy && (
          <Button
            color="secondary-destructive"
            size="md"
            iconLeading={Trash01}
            onClick={onDelete}
            isDisabled={destroyLoading}
          >
            {destroyLoading ? "Deleting..." : "Delete"}
          </Button>
        )}
      </div>
    </div>
  );
}

