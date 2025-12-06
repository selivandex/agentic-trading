/** @format */

"use client";

import React from "react";
import { Skeleton } from "@/shared/ui/skeleton/skeleton";

/**
 * Show page loading state presentation component
 */
export function CrudShowLoadingState() {
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

