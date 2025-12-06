/** @format */

"use client";

import { Crud } from "@/shared/ui/crud";
import { agentCrudConfig } from "@/entities/agent";

/**
 * Create New Agent Page
 */
export default function NewAgentPage() {
  return (
    <div className="h-full">
      <Crud config={agentCrudConfig} mode="new" />
    </div>
  );
}
