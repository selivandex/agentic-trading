/** @format */

"use client";

import { ArrowLeft, Edit01, Trash01 } from "@untitledui/icons";
import { Button } from "@/components/base/buttons/button";
import { Skeleton } from "@/shared/ui/skeleton/skeleton";
import { useCrudContext } from "@/shared/lib/crud/context";
import { useCrudShowQuery } from "@/shared/lib/crud/use-crud-query";
import { useCrudMutations } from "@/shared/lib/crud/use-crud-mutations";
import type { CrudEntity } from "@/shared/lib/crud/types";

/**
 * CRUD Show Component
 * Displays single entity details
 */
export function CrudShow<TEntity extends CrudEntity = CrudEntity>({
  entityId,
}: {
  entityId: string;
}) {
  const { config, actions } = useCrudContext<TEntity>();

  // Fetch entity
  const { entity, loading, error, refetch } = useCrudShowQuery<TEntity>(
    config,
    entityId
  );

  // Mutations
  const { destroy, destroyLoading } = useCrudMutations<TEntity>(config);

  // Handle delete
  const handleDelete = async () => {
    if (
      confirm(`Are you sure you want to delete this ${config.resourceName}?`)
    ) {
      await destroy(entityId);
      actions.goToIndex();
    }
  };

  // Loading state
  if (loading) {
    return (
      <div className="mx-auto max-w-4xl">
        <div className="mb-6 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Skeleton className="h-10 w-20" />
            <Skeleton className="h-8 w-48" />
          </div>
          <div className="flex items-center gap-3">
            <Skeleton className="h-10 w-20" />
            <Skeleton className="h-10 w-24" />
          </div>
        </div>

        <div className="rounded-xl bg-primary p-6 shadow-sm ring-1 ring-secondary">
          <div className="space-y-6">
            {Array.from({ length: 6 }).map((_, i) => (
              <div
                key={i}
                className="border-b border-secondary pb-4 last:border-0"
              >
                <Skeleton className="mb-2 h-4 w-32" />
                <Skeleton className="h-6 w-full" />
              </div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  // Error state
  if (error || !entity) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <h2 className="text-lg font-semibold text-primary">
          Error loading data
        </h2>
        <p className="mt-2 text-sm text-secondary">
          {config.errorMessage ?? error?.message ?? "Entity not found"}
        </p>
        <Button onClick={() => refetch()} size="lg" className="mt-4">
          Try Again
        </Button>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-4xl">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button
            color="secondary"
            size="md"
            iconLeading={ArrowLeft}
            onClick={() => actions.goToIndex()}
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
            onClick={() => actions.goToEdit(entityId)}
          >
            Edit
          </Button>
          <Button
            color="secondary-destructive"
            size="md"
            iconLeading={Trash01}
            onClick={handleDelete}
            isDisabled={destroyLoading}
          >
            {destroyLoading ? "Deleting..." : "Delete"}
          </Button>
        </div>
      </div>

      {/* Content */}
      <div className="rounded-xl bg-primary p-6 shadow-sm ring-1 ring-secondary">
        <div className="space-y-6">
          {config.formFields.map((field) => {
            const value = entity[field.name];

            return (
              <div
                key={field.name}
                className="border-b border-secondary pb-4 last:border-0"
              >
                <dt className="mb-2 text-sm font-medium text-secondary">
                  {field.label}
                </dt>
                <dd className="text-base text-primary">
                  {formatFieldValue(value, field.type)}
                </dd>
              </div>
            );
          })}

          {/* Custom actions */}
          {config.actions && config.actions.length > 0 && (
            <div className="mt-6 flex gap-3 border-t border-secondary pt-6">
              {config.actions
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
                    isDisabled={
                      action.disabled ? action.disabled(entity) : false
                    }
                  >
                    {action.label}
                  </Button>
                ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

/**
 * Format field value for display
 */
function formatFieldValue(value: unknown, fieldType: string): string {
  if (value == null) {
    return "—";
  }

  switch (fieldType) {
    case "checkbox":
      return value ? "Yes" : "No";

    case "date":
      if (typeof value === "string" || value instanceof Date) {
        return new Date(value).toLocaleDateString();
      }
      return String(value);

    case "datetime":
      if (typeof value === "string" || value instanceof Date) {
        return new Date(value).toLocaleString();
      }
      return String(value);

    case "number":
      if (typeof value === "number") {
        return value.toLocaleString();
      }
      return String(value);

    default:
      if (typeof value === "object") {
        return JSON.stringify(value, null, 2);
      }
      return String(value);
  }
}
