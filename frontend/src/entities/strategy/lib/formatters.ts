/** @format */

import type {
  Strategy,
  StrategyStatus,
  MarketType,
  RiskTolerance,
  RebalanceFrequency,
} from "@/entities/strategy";

/**
 * Format strategy status for display
 */
export function formatStrategyStatus(status: StrategyStatus): string {
  const map: Record<StrategyStatus, string> = {
    active: "Active",
    paused: "Paused",
    closed: "Closed",
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
    active: "success",
    paused: "warning",
    closed: "gray",
  };

  return colorMap[status] || "gray";
}

/**
 * Format market type for display
 */
export function formatMarketType(marketType: MarketType): string {
  const map: Record<MarketType, string> = {
    spot: "Spot",
    futures: "Futures",
  };

  return map[marketType] || marketType;
}

/**
 * Format risk tolerance for display
 */
export function formatRiskTolerance(riskTolerance: RiskTolerance): string {
  const map: Record<RiskTolerance, string> = {
    conservative: "Conservative",
    moderate: "Moderate",
    aggressive: "Aggressive",
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
    daily: "Daily",
    weekly: "Weekly",
    monthly: "Monthly",
    never: "Never",
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
