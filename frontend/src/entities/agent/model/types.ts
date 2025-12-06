/** @format */

/**
 * Agent Entity Types
 *
 * Auto-generated from table: agents
 */

import type { CrudEntity } from "@/shared/lib/crud";

export interface Agent extends CrudEntity {
  id: string;
  identifier: string;
  name: string;
  description?: string | null;
  category?: string | null;
  systemPrompt: string;
  instructions?: string | null;
  modelProvider?: string | null;
  modelName?: string | null;
  temperature?: number | null;
  maxTokens?: number | null;
  availableTools?: Record<string, any> | null;
  maxCostPerRun?: number | null;
  timeoutSeconds?: number | null;
  isActive?: boolean | null;
  version?: number | null;
  createdAt: string;
  updatedAt: string;
}

export interface CreateAgentInput {
  identifier?: string;
  name?: string;
  description?: string;
  category?: string;
  systemPrompt?: string;
  instructions?: string;
  modelProvider?: string;
  modelName?: string;
  temperature?: number;
  maxTokens?: number;
  availableTools?: Record<string, any>;
  maxCostPerRun?: number;
  timeoutSeconds?: number;
  isActive?: boolean;
  version?: number;
}

export interface UpdateAgentInput {
  identifier?: string;
  name?: string;
  description?: string;
  category?: string;
  systemPrompt?: string;
  instructions?: string;
  modelProvider?: string;
  modelName?: string;
  temperature?: number;
  maxTokens?: number;
  availableTools?: Record<string, any>;
  maxCostPerRun?: number;
  timeoutSeconds?: number;
  isActive?: boolean;
  version?: number;
}
