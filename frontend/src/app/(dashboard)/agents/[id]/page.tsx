/** @format */

"use client";

import { Crud } from "@/shared/ui/crud";
import { agentCrudConfig } from "@/entities/agent";
import { use } from "react";

/**
 * Agent Detail Page
 */
export default function AgentDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);

  return (
    <div className="h-full">
      <Crud config={agentCrudConfig} mode="show" entityId={id} />
    </div>
  );
}
