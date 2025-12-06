/** @format */

/**
 * Strategy Entity Types
 *
 * Trading strategy entity with configuration and performance metrics
 */

import type { User } from "@/entities/user";
import type { CrudEntity } from "@/shared/lib/crud";

export enum StrategyStatus {
  ACTIVE = "active",
  PAUSED = "paused",
  CLOSED = "closed",
}

export enum MarketType {
  SPOT = "spot",
  FUTURES = "futures",
}

export enum RiskTolerance {
  CONSERVATIVE = "conservative",
  MODERATE = "moderate",
  AGGRESSIVE = "aggressive",
}

export enum RebalanceFrequency {
  DAILY = "daily",
  WEEKLY = "weekly",
  MONTHLY = "monthly",
  NEVER = "never",
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
