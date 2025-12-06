/** @format */

"use client";

import React from "react";
import type {
  CrudConfig,
  CrudEntity,
  CrudFormField,
} from "@/shared/lib/crud/types";

/**
 * Show page content presentation component
 */
interface CrudShowContentProps<TEntity extends CrudEntity> {
  config: CrudConfig<TEntity>;
  entity: TEntity;
  formatFieldValue: (value: unknown, fieldType: string) => string;
}

export function CrudShowContent<TEntity extends CrudEntity>({
  config,
  entity,
  formatFieldValue,
}: CrudShowContentProps<TEntity>) {
  const showConfig = config.show;
  const layout = showConfig?.layout ?? "single-column";

  // Custom layout - full control
  if (layout === "custom" && showConfig?.customContent) {
    return <>{showConfig.customContent(entity)}</>;
  }

  // Two-column layout
  if (layout === "two-column") {
    return (
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main content (left side, 2/3 width) */}
        <div className="lg:col-span-2">
          {showConfig?.beforeContent && showConfig.beforeContent(entity)}
          {showConfig?.mainContent ? (
            <>{showConfig.mainContent(entity)}</>
          ) : (
            <DefaultFieldsList
              formFields={config.formFields}
              entity={entity}
              formatFieldValue={formatFieldValue}
            />
          )}
          {showConfig?.afterContent && showConfig.afterContent(entity)}
        </div>

        {/* Sidebar (right side, 1/3 width) */}
        <div className="lg:col-span-1">
          {showConfig?.sidebar && showConfig.sidebar(entity)}
        </div>
      </div>
    );
  }

  // Single-column layout (default)
  return (
    <div className="max-w-4xl">
      {showConfig?.beforeContent && showConfig.beforeContent(entity)}
      {showConfig?.mainContent ? (
        <>{showConfig.mainContent(entity)}</>
      ) : (
        <DefaultFieldsList
          formFields={config.formFields}
          entity={entity}
          formatFieldValue={formatFieldValue}
        />
      )}
      {showConfig?.afterContent && showConfig.afterContent(entity)}
    </div>
  );
}

/**
 * Default fields list presentation
 */
function DefaultFieldsList<TEntity extends CrudEntity>({
  formFields,
  entity,
  formatFieldValue,
}: {
  formFields: CrudFormField<TEntity>[];
  entity: TEntity;
  formatFieldValue: (value: unknown, fieldType: string) => string;
}) {
  return (
    <div className="rounded-xl bg-primary p-6 shadow-sm ring-1 ring-secondary">
      <div className="space-y-6">
        {formFields.map((field: CrudFormField) => {
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
      </div>
    </div>
  );
}

