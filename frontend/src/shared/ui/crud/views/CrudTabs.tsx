/** @format */

"use client";

import { Tabs } from "@/shared/ui/tabs/tabs";
import type {
  CrudEntity,
  CrudTabsConfig,
  Scope,
} from "@/shared/lib/crud/types";

interface CrudTabsProps<_TEntity extends CrudEntity = CrudEntity> {
  /** Tabs configuration (display settings) */
  config: CrudTabsConfig;
  /** Active scope ID */
  activeScope?: string;
  /** Scopes from backend */
  scopes?: Scope[];
  /** Callback when scope changes */
  onScopeChange: (scopeId: string) => void;
}

/**
 * CRUD Tabs Component
 * Displays tabs for filtering data based on scopes from backend
 */
export function CrudTabs<TEntity extends CrudEntity = CrudEntity>({
  config,
  activeScope,
  scopes,
  onScopeChange,
}: CrudTabsProps<TEntity>) {
  const { type = "underline", size = "md", fullWidth = false } = config;

  // If no scopes from backend, don't render
  if (!scopes || scopes.length === 0) {
    return null;
  }

  // Prepare items for TabList
  const tabItems = scopes.map((scope) => {
    // Map icon if mapper function provided
    const icon = config.iconMapper ? config.iconMapper(scope.id) : undefined;

    return {
      id: scope.id,
      label: scope.name,
      badge: scope.count,
      icon,
    };
  });

  return (
    <Tabs
      selectedKey={activeScope}
      onSelectionChange={(key) => onScopeChange(key as string)}
    >
      <Tabs.List
        type={type}
        size={size}
        fullWidth={fullWidth}
        items={tabItems}
        aria-label="Filter tabs"
      />
    </Tabs>
  );
}
