/** @format */

"use client";

import { Crud } from "@/shared/ui/crud";
import { strategyCrudConfig } from "@/entities/strategy";
import { use } from "react";

/**
 * Edit Strategy Page
 */
export default function EditStrategyPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);

  return (
    <div className="h-full">
      <Crud config={strategyCrudConfig} mode="edit" entityId={id} />
    </div>
  );
}
