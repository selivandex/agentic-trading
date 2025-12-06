/** @format */

"use client";

import { Crud } from "@/shared/ui/crud";
import { agentCrudConfig } from "@/entities/agent";
import { use } from "react";

/**
 * Edit Agent Page
 */
export default function EditAgentPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);

  return (
    <div className="h-full">
      <Crud config={agentCrudConfig} mode="edit" entityId={id} />
    </div>
  );
}
