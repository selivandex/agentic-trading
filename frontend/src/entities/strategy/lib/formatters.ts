/** @format */

import type {
  Strategy,
  StrategyStatus,
  MarketType,
  RiskTolerance,
  RebalanceFrequency,
} from "@/entities/strategy/model/types";

/**
 * Format strategy status for display
 */
export function formatStrategyStatus(status: StrategyStatus): string {
  const map: Record<StrategyStatus, string> = {
    ACTIVE: "Active",
    PAUSED: "Paused",
    CLOSED: "Closed",
  };

  return map[status] || status;
}

/**
 * Get strategy status color for badges
 */
export function getStrategyStatusColor(
  status: StrategyStatus
): "success" | "warning" | "gray" {
  const colorMap: Record<StrategyStatus, "success" | "warning" | "gray"> = {
    ACTIVE: "success",
    PAUSED: "warning",
    CLOSED: "gray",
  };

  return colorMap[status] || "gray";
}

/**
 * Format market type for display
 */
export function formatMarketType(marketType: MarketType): string {
  const map: Record<MarketType, string> = {
    SPOT: "Spot",
    FUTURES: "Futures",
  };

  return map[marketType] || marketType;
}

/**
 * Format risk tolerance for display
 */
export function formatRiskTolerance(riskTolerance: RiskTolerance): string {
  const map: Record<RiskTolerance, string> = {
    CONSERVATIVE: "Conservative",
    MODERATE: "Moderate",
    AGGRESSIVE: "Aggressive",
  };

  return map[riskTolerance] || riskTolerance;
}

/**
 * Format rebalance frequency for display
 */
export function formatRebalanceFrequency(
  frequency: RebalanceFrequency
): string {
  const map: Record<RebalanceFrequency, string> = {
    DAILY: "Daily",
    WEEKLY: "Weekly",
    MONTHLY: "Monthly",
    NEVER: "Never",
  };

  return map[frequency] || frequency;
}

/**
 * Format decimal value to currency
 */
export function formatCurrency(value: string | number, decimals = 2): string {
  const num = typeof value === "string" ? parseFloat(value) : value;

  if (isNaN(num)) return "$0.00";

  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  }).format(num);
}

/**
 * Format percentage value
 */
export function formatPercentage(
  value: string | number,
  decimals = 2
): string {
  const num = typeof value === "string" ? parseFloat(value) : value;

  if (isNaN(num)) return "0.00%";

  const sign = num >= 0 ? "+" : "";
  return `${sign}${num.toFixed(decimals)}%`;
}

/**
 * Get PnL color class based on value
 */
export function getPnLColorClass(value: string | number): string {
  const num = typeof value === "string" ? parseFloat(value) : value;

  if (isNaN(num) || num === 0) return "text-gray-600";

  return num > 0 ? "text-green-600" : "text-red-600";
}

/**
 * Calculate strategy ROI
 */
export function calculateROI(strategy: Strategy): number {
  const allocated = parseFloat(strategy.allocatedCapital);
  const current = parseFloat(strategy.currentEquity);

  if (isNaN(allocated) || isNaN(current) || allocated === 0) return 0;

  return ((current - allocated) / allocated) * 100;
}
