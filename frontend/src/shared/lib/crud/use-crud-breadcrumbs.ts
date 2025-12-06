/** @format */

import { useMemo } from "react";
import type { BreadcrumbItemData } from "@/shared/ui/page-header";
import type {
  CrudEntity,
  CrudConfig,
  CrudResourceGroup,
  CrudBreadcrumbsConfig,
} from "./types";

/**
 * Build resource group hierarchy (flattened from parent chain)
 */
function buildResourceGroupHierarchy(
  group: CrudResourceGroup
): CrudResourceGroup[] {
  const hierarchy: CrudResourceGroup[] = [];
  let current: CrudResourceGroup | undefined = group;

  while (current) {
    hierarchy.unshift(current);
    current = current.parent;
  }

  return hierarchy;
}

/**
 * Generate breadcrumbs from CRUD config and current mode
 */
function generateBreadcrumbs<TEntity extends CrudEntity>(
  config: CrudConfig<TEntity>,
  mode: "index" | "show" | "new" | "edit",
  entity?: TEntity
): BreadcrumbItemData[] {
  const items: BreadcrumbItemData[] = [];

  // Add resource group hierarchy
  if (config.resourceGroup) {
    const groupHierarchy = buildResourceGroupHierarchy(config.resourceGroup);
    for (const group of groupHierarchy) {
      items.push({
        label: group.name,
        href: group.path,
        icon: group.icon,
      });
    }
  }

  // Add resource list item
  items.push({
    label: config.resourceNamePlural,
    href: config.basePath ?? "#",
  });

  // Add mode-specific items
  switch (mode) {
    case "show":
      if (entity) {
        const entityLabel =
          (entity as { name?: string }).name ??
          (entity as { title?: string }).title ??
          `${config.resourceName} #${entity.id}`;
        items.push({
          label: entityLabel,
          href: `${config.basePath}/${entity.id}`,
        });
      }
      break;
    case "new":
      items.push({
        label: `New ${config.resourceName}`,
      });
      break;
    case "edit":
      if (entity) {
        const entityLabel =
          (entity as { name?: string }).name ??
          (entity as { title?: string }).title ??
          `${config.resourceName} #${entity.id}`;
        items.push({
          label: entityLabel,
          href: `${config.basePath}/${entity.id}`,
        });
        items.push({
          label: "Edit",
        });
      }
      break;
    case "index":
    default:
      // For index, remove the last item since we're already there
      items.pop();
      break;
  }

  return items;
}

/**
 * Hook to generate breadcrumbs for CRUD pages
 *
 * Auto-generates breadcrumbs from CRUD config, resource groups, and current mode.
 * Can be overridden with custom breadcrumbs in config.
 */
export function useCrudBreadcrumbs<TEntity extends CrudEntity>(
  config: CrudConfig<TEntity>,
  mode: "index" | "show" | "new" | "edit",
  entity?: TEntity
): BreadcrumbItemData[] | undefined {
  return useMemo(() => {
    const breadcrumbsConfig: CrudBreadcrumbsConfig<TEntity> | undefined =
      config.breadcrumbs;

    // If breadcrumbs explicitly disabled, return undefined
    if (breadcrumbsConfig?.enabled === false) {
      return undefined;
    }

    // Build final breadcrumb items
    const items: BreadcrumbItemData[] = [];

    // Add root items (e.g., Home)
    if (breadcrumbsConfig?.rootItems) {
      items.push(
        ...breadcrumbsConfig.rootItems.map((item) => ({
          label:
            typeof item.label === "function" ? item.label(entity) : item.label,
          href:
            typeof item.href === "function" ? item.href(entity) : item.href,
          icon: item.icon,
        }))
      );
    }

    // Get mode-specific breadcrumbs (manual or auto-generated)
    let modeItems: BreadcrumbItemData[] = [];
    const shouldAutoGenerate = breadcrumbsConfig?.autoGenerate !== false;

    // Check for manual breadcrumbs based on mode
    let manualItems = breadcrumbsConfig?.[mode];

    if (manualItems && manualItems.length > 0) {
      // Use manual breadcrumbs
      modeItems = manualItems.map((item) => ({
        label:
          typeof item.label === "function" ? item.label(entity) : item.label,
        href: typeof item.href === "function" ? item.href(entity) : item.href,
        icon: item.icon,
      }));
    } else if (shouldAutoGenerate) {
      // Auto-generate breadcrumbs
      modeItems = generateBreadcrumbs(config, mode, entity);
    }

    items.push(...modeItems);

    // If no items at all, return undefined
    return items.length > 0 ? items : undefined;
  }, [config, mode, entity]);
}

