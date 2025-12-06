/** @format */

"use client";

import { useMemo } from "react";
import { Crud } from "@/shared/ui/crud";
import { agentCrudConfig } from "@/entities/agent";

/**
 * AgentManager Component
 *
 * Enhanced Crud wrapper for Agents
 * Auto-generated from table: agents
 */
export function AgentManager() {
  const enhancedConfig = useMemo(() => {
    return {
      ...agentCrudConfig,
      enableSelection: true,
    };
  }, []);

  return <Crud config={enhancedConfig} />;
}
