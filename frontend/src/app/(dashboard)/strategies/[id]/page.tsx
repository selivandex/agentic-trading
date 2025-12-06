/** @format */

"use client";

import { Crud } from "@/shared/ui/crud";
import { strategyCrudConfig } from "@/entities/strategy";
import { use } from "react";

/**
 * Strategy Detail Page
 */
export default function StrategyDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);

  return (
    <div className="container mx-auto py-8">
      <Crud config={strategyCrudConfig} mode="show" entityId={id} />
    </div>
  );
}
