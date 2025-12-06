/** @format */

"use client";

import { Crud } from "@/shared/ui/crud";
import { strategyCrudConfig } from "@/entities/strategy";

/**
 * Create New Strategy Page
 */
export default function NewStrategyPage() {
  return (
    <div className="h-full">
      <Crud config={strategyCrudConfig} mode="new" />
    </div>
  );
}
