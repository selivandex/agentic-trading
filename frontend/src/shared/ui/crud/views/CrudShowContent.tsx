/** @format */

"use client";

import React from "react";
import type { CrudConfig, CrudEntity, CrudFormField } from "@/shared/lib/crud/types";

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
  return (
    <div className="rounded-xl bg-primary p-6 shadow-sm ring-1 ring-secondary">
      <div className="space-y-6">
        {config.formFields.map((field: CrudFormField) => {
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

