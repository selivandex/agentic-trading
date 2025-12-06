/** @format */

"use client";

import React from "react";
import { Button } from "@/shared/base/buttons/button";
import { useCrudContext } from "@/shared/lib/crud/context";
import { useCrudShow } from "@/shared/lib/crud/use-crud-show";
import { useCrudFieldFormatter } from "@/shared/lib/crud/use-crud-field-formatter";
import { useCrudBreadcrumbs } from "@/shared/lib/crud/use-crud-breadcrumbs";
import type { CrudEntity } from "@/shared/lib/crud/types";
import { PageHeader, PageHeaderSkeleton } from "@/shared/ui/page-header";
import { CrudShowContent } from "./views/CrudShowContent";
import { CrudShowErrorState } from "./views/CrudShowErrorState";

/**
 * CRUD Show Component
 * Container that orchestrates show page hooks and presentation components
 */
export function CrudShow<TEntity extends CrudEntity = CrudEntity>({
  entityId,
}: {
  entityId: string;
}) {
  const { config, actions } = useCrudContext<TEntity>();

  // Logic hooks
  const { entity, loading, error, refetch, handleDelete, destroyLoading } =
    useCrudShow<TEntity>(config, entityId, () => actions.goToIndex());

  const { formatFieldValue } = useCrudFieldFormatter();

  // Breadcrumbs
  const breadcrumbs = useCrudBreadcrumbs<TEntity>(config, "show", entity ?? undefined);

  const paddingClasses = "px-4 lg:px-8";

  // Get entity title for display
  const entityTitle = entity
    ? (entity as { name?: string }).name ??
      (entity as { title?: string }).title ??
      `${config.resourceName} #${entity.id}`
    : config.resourceName;

  // Loading state
  if (loading) {
    return (
      <div className="mx-auto mb-8 flex flex-col gap-5">
        <PageHeaderSkeleton
          background="transparent"
          showBreadcrumbs={!!breadcrumbs && breadcrumbs.length > 0}
          breadcrumbCount={breadcrumbs?.length ?? 0}
          showTitle
          actionCount={2}
        />
        <div className={paddingClasses}>
          <div className="rounded-xl border border-gray-200 bg-white p-8 shadow-sm">
            <div className="space-y-6">
              {Array.from({ length: 6 }).map((_, i) => (
                <div key={i} className="space-y-2">
                  <div className="h-4 w-32 bg-gray-200 rounded animate-pulse" />
                  <div className="h-6 w-full bg-gray-100 rounded animate-pulse" />
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Error state
  if (error || !entity) {
    return (
      <CrudShowErrorState
        config={config}
        error={error}
        onRetry={() => refetch()}
      />
    );
  }

  // Build actions for PageHeader
  const showActions = config.showActions
    ? config.showActions
        .filter((action) => !action.hidden || !action.hidden(entity))
        .map((action) => (
          <Button
            key={action.key}
            color={action.destructive ? "secondary-destructive" : "secondary"}
            size="md"
            onClick={() => action.onClick(entity)}
            isDisabled={
              destroyLoading || (action.disabled && action.disabled(entity))
            }
          >
            {action.label}
          </Button>
        ))
    : [];

  return (
    <div className="mx-auto mb-8 flex flex-col gap-5">
      {/* Page Header with Breadcrumbs, Title, and Actions */}
      <PageHeader
        background="transparent"
        breadcrumbs={breadcrumbs}
        showBreadcrumbs={!!breadcrumbs && breadcrumbs.length > 0}
        title={entityTitle}
        backHref={config.basePath}
        onBackClick={() => actions.goToIndex()}
      >
        <PageHeader.Actions>
          {showActions}
          <Button
            color="secondary"
            size="md"
            onClick={() => actions.goToEdit(entityId)}
            isDisabled={destroyLoading}
          >
            Edit
          </Button>
          {config.graphql.destroy && (
            <Button
              color="secondary-destructive"
              size="md"
              onClick={handleDelete}
              isDisabled={destroyLoading}
            >
              {destroyLoading ? "Deleting..." : "Delete"}
            </Button>
          )}
        </PageHeader.Actions>
      </PageHeader>

      <div className={paddingClasses}>
        {/* Content */}
        <CrudShowContent
          config={config}
          entity={entity}
          formatFieldValue={formatFieldValue}
        />
      </div>
    </div>
  );
}
