/** @format */

"use client";

import React from "react";
import { Button } from "@/components/base/buttons/button";
import type { CrudConfig, CrudEntity } from "@/shared/lib/crud/types";

/**
 * Error state presentation component
 */
interface CrudErrorStateProps<TEntity extends CrudEntity> {
  config: CrudConfig<TEntity>;
  error: Error;
  onRetry: () => void;
}

export function CrudErrorState<TEntity extends CrudEntity>({
  config,
  error,
  onRetry,
}: CrudErrorStateProps<TEntity>) {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <h2 className="text-lg font-semibold text-primary">Error loading data</h2>
      <p className="mt-2 text-sm text-secondary">
        {config.errorMessage ?? error?.message ?? "Something went wrong"}
      </p>
      <Button onClick={onRetry} size="lg" className="mt-4">
        Try Again
      </Button>
    </div>
  );
}

