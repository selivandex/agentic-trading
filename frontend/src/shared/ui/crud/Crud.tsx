/** @format */

"use client";

import { CrudProvider } from "@/shared/lib/crud/context";
import type { CrudConfig, CrudEntity } from "@/shared/lib/crud/types";
import { CrudTable } from "./CrudTable";
import { CrudForm } from "./CrudForm";
import { CrudShow } from "./CrudShow";

/**
 * Main CRUD Component
 * Orchestrates all CRUD views based on current state
 */
export interface CrudProps<TEntity extends CrudEntity = CrudEntity> {
  config: CrudConfig<TEntity>;
  /** Initial view mode */
  mode?: "index" | "show" | "new" | "edit";
  /** Initial entity ID (for show/edit) */
  entityId?: string;
}

export function Crud<TEntity extends CrudEntity = CrudEntity>({
  config,
  mode = "index",
  entityId,
}: CrudProps<TEntity>) {
  return (
    <CrudProvider
      config={config}
      initialMode={mode}
      _initialEntityId={entityId}
    >
      <CrudRouter entityId={entityId} />
    </CrudProvider>
  );
}

/**
 * Internal router component that renders the appropriate view
 */
function CrudRouter<TEntity extends CrudEntity = CrudEntity>({
  entityId,
}: {
  entityId?: string;
}) {
  const { state } = useCrudContext<TEntity>();

  switch (state.mode) {
    case "index":
      return <CrudTable<TEntity> />;

    case "show":
      return entityId ? (
        <CrudShow<TEntity> entityId={entityId} />
      ) : (
        <div>No entity ID provided</div>
      );

    case "new":
      return <CrudForm<TEntity> mode="new" />;

    case "edit":
      return entityId ? (
        <CrudForm<TEntity> mode="edit" entityId={entityId} />
      ) : (
        <div>No entity ID provided</div>
      );

    default:
      return <div>Unknown mode</div>;
  }
}

// Import useCrudContext for CrudRouter
import { useCrudContext } from "@/shared/lib/crud/context";
