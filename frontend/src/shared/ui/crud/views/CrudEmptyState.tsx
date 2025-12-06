/** @format */

"use client";

import React from "react";
import { Plus } from "@untitledui/icons";
import { Button } from "@/components/base/buttons/button";
import type { CrudConfig, CrudEntity } from "@/shared/lib/crud/types";

/**
 * Empty state presentation component
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
    <div className="flex flex-col items-center justify-center py-12">
      <h2 className="text-lg font-semibold text-primary">
        No {config.resourceNamePlural.toLowerCase()}
      </h2>
      <p className="mt-2 text-sm text-secondary">
        {config.emptyStateMessage ??
          `Get started by creating a new ${config.resourceName.toLowerCase()}`}
      </p>
      <Button size="lg" iconLeading={Plus} onClick={onNew} className="mt-4">
        New {config.resourceName}
      </Button>
    </div>
  );
}

