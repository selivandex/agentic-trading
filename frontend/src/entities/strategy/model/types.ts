/** @format */

/**
 * Strategy Entity Types
 *
 * Trading strategy entity with configuration and performance metrics
 */

import type { User } from "@/entities/user";
import type { CrudEntity } from "@/shared/lib/crud";

export enum StrategyStatus {
  ACTIVE = "ACTIVE",
  PAUSED = "PAUSED",
  CLOSED = "CLOSED",
}

export enum MarketType {
  SPOT = "SPOT",
  FUTURES = "FUTURES",
}

export enum RiskTolerance {
  CONSERVATIVE = "CONSERVATIVE",
  MODERATE = "MODERATE",
  AGGRESSIVE = "AGGRESSIVE",
}

export enum RebalanceFrequency {
  DAILY = "DAILY",
  WEEKLY = "WEEKLY",
  MONTHLY = "MONTHLY",
  NEVER = "NEVER",
}

export interface Strategy extends CrudEntity {
  id: string;
  userID: string;
  user?: User | null;

  // Strategy metadata
  name: string;
  description: string;
  status: StrategyStatus;

  // Capital allocation
  allocatedCapital: string;
  currentEquity: string;
  cashReserve: string;

  // Configuration
  marketType: MarketType;
  riskTolerance: RiskTolerance;
  rebalanceFrequency: RebalanceFrequency;
  targetAllocations?: Record<string, unknown> | null;

  // Performance metrics
  totalPnL: string;
  totalPnLPercent: string;
  sharpeRatio?: string | null;
  maxDrawdown?: string | null;
  winRate?: string | null;

  // Timestamps
  createdAt: string;
  updatedAt: string;
  closedAt?: string | null;
  lastRebalancedAt?: string | null;

  // Reasoning log
  reasoningLog?: Record<string, unknown> | null;
}

export interface CreateStrategyInput {
  name: string;
  description: string;
  allocatedCapital: string;
  marketType: MarketType;
  riskTolerance: RiskTolerance;
  rebalanceFrequency: RebalanceFrequency;
  targetAllocations?: Record<string, unknown>;
}

export interface UpdateStrategyInput {
  name?: string;
  description?: string;
  riskTolerance?: RiskTolerance;
  rebalanceFrequency?: RebalanceFrequency;
  targetAllocations?: Record<string, unknown>;
}
