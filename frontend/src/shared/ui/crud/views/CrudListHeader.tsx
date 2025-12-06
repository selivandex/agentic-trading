/** @format */

"use client";

import React from "react";
import { Plus } from "@untitledui/icons";
import { Button } from "@/components/base/buttons/button";
import { Input } from "@/components/base/input/input";
import { TableCard } from "@/components/application/table/table";
import type { CrudConfig, CrudEntity } from "@/shared/lib/crud/types";

/**
 * CRUD list header presentation component
 */
interface CrudListHeaderProps<TEntity extends CrudEntity> {
  config: CrudConfig<TEntity>;
  entitiesCount: number;
  searchQuery?: string;
  onSearchChange: (value: string | React.ChangeEvent<HTMLInputElement>) => void;
  onNew: () => void;
}

export function CrudListHeader<TEntity extends CrudEntity>({
  config,
  entitiesCount,
  searchQuery,
  onSearchChange,
  onNew,
}: CrudListHeaderProps<TEntity>) {
  return (
    <TableCard.Header
      title={config.resourceNamePlural}
      badge={entitiesCount}
      contentTrailing={
        <div className="flex items-center gap-3">
          {config.enableSearch && (
            <Input
              type="search"
              placeholder="Search..."
              value={searchQuery ?? ""}
              onChange={onSearchChange}
              className="w-64"
            />
          )}
          <Button size="md" iconLeading={Plus} onClick={onNew}>
            New {config.resourceName}
          </Button>
        </div>
      }
    />
  );
}

