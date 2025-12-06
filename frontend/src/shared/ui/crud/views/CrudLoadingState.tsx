/** @format */

"use client";

import React from "react";
import { Skeleton } from "@/shared/ui/skeleton/skeleton";

/**
 * Loading state presentation component
 */
export function CrudLoadingState() {
  return (
    <div className="space-y-3 p-6">
      <div className="flex items-center justify-between">
        <Skeleton className="h-5 w-32" />
        <Skeleton className="h-5 w-24" />
      </div>
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="flex items-center gap-4 py-4">
          <Skeleton className="h-10 flex-1" />
          <Skeleton className="h-10 w-24" />
          <Skeleton className="h-10 w-32" />
          <Skeleton className="h-10 w-20" />
        </div>
      ))}
    </div>
  );
}

