/** @format */

"use client";

import React from "react";
import { Plus } from "@untitledui/icons";
import { Button } from "@/components/base/buttons/button";
import { EmptyState } from "@/shared/application";
import type { CrudConfig, CrudEntity } from "@/shared/lib/crud/types";

/**
 * Empty state presentation component using shared EmptyState
 */
interface CrudEmptyStateProps<TEntity extends CrudEntity> {
  config: CrudConfig<TEntity>;
  onNew: () => void;
}

export function CrudEmptyState<TEntity extends CrudEntity>({
  config,
  onNew,
}: CrudEmptyStateProps<TEntity>) {
  return (
    <div className="py-12">
      <EmptyState size="md">
        <EmptyState.Header>
          <EmptyState.FeaturedIcon color="gray" />
        </EmptyState.Header>

        <EmptyState.Content>
          <EmptyState.Title>
            No {config.resourceNamePlural.toLowerCase()}
          </EmptyState.Title>
          <EmptyState.Description>
            {config.emptyStateMessage ??
              `Get started by creating a new ${config.resourceName.toLowerCase()}`}
          </EmptyState.Description>
        </EmptyState.Content>

        <EmptyState.Footer>
          <Button size="md" iconLeading={Plus} onClick={onNew}>
            New {config.resourceName}
          </Button>
        </EmptyState.Footer>
      </EmptyState>
    </div>
  );
}
