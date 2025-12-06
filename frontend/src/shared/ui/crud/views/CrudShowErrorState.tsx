/** @format */

"use client";

import React from "react";
import { Button } from "@/components/base/buttons/button";
import type { CrudConfig, CrudEntity } from "@/shared/lib/crud/types";

/**
 * Show page error state presentation component
 */
interface CrudShowErrorStateProps<TEntity extends CrudEntity> {
  config: CrudConfig<TEntity>;
  error?: Error;
  onRetry: () => void;
}

export function CrudShowErrorState<TEntity extends CrudEntity>({
  config,
  error,
  onRetry,
}: CrudShowErrorStateProps<TEntity>) {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <h2 className="text-lg font-semibold text-primary">Error loading data</h2>
      <p className="mt-2 text-sm text-secondary">
        {config.errorMessage ?? error?.message ?? "Entity not found"}
      </p>
      <Button onClick={onRetry} size="lg" className="mt-4">
        Try Again
      </Button>
    </div>
  );
}

