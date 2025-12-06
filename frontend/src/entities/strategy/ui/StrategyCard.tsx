/** @format */

import { BadgeWithDot } from "@/shared/base";
import type { Strategy } from "@/entities/strategy";
import {
  formatStrategyStatus,
  getStrategyStatusColor,
  formatMarketType,
  formatRiskTolerance,
  formatCurrency,
  formatPercentage,
} from "@/entities/strategy";

interface StrategyCardProps {
  strategy: Strategy;
  onClick?: () => void;
  className?: string;
}

/**
 * StrategyCard component
 *
 * Displays strategy information in a card format
 */
export function StrategyCard({
  strategy,
  onClick,
  className = "",
}: StrategyCardProps) {
  const statusColor = getStrategyStatusColor(strategy.status);
  const pnlValue = parseFloat(strategy.totalPnLPercent);
  const pnlColor: "success" | "error" | "gray" =
    pnlValue > 0 ? "success" : pnlValue < 0 ? "error" : "gray";

  return (
    <div
      className={`border border-primary rounded-lg p-4 hover:shadow-md transition-shadow ${
        onClick ? "cursor-pointer" : ""
      } ${className}`}
      onClick={onClick}
    >
      <div className="flex items-start justify-between mb-3">
        <div className="flex-1">
          <h3 className="text-lg font-semibold text-primary">
            {strategy.name}
          </h3>
          <p className="text-sm text-secondary mt-1 line-clamp-2">
            {strategy.description}
          </p>
        </div>

        <BadgeWithDot color={statusColor} size="sm">
          {formatStrategyStatus(strategy.status)}
        </BadgeWithDot>
      </div>

      <div className="grid grid-cols-2 gap-4 mb-3">
        <div>
          <p className="text-xs text-tertiary">Allocated Capital</p>
          <p className="text-sm font-semibold text-primary">
            {formatCurrency(strategy.allocatedCapital)}
          </p>
        </div>

        <div>
          <p className="text-xs text-tertiary">Current Equity</p>
          <p className="text-sm font-semibold text-primary">
            {formatCurrency(strategy.currentEquity)}
          </p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4 mb-3">
        <div>
          <p className="text-xs text-tertiary">Total P&L</p>
          <BadgeWithDot color={pnlColor} size="sm">
            {formatCurrency(strategy.totalPnL)}
          </BadgeWithDot>
        </div>

        <div>
          <p className="text-xs text-tertiary">P&L %</p>
          <BadgeWithDot color={pnlColor} size="sm">
            {formatPercentage(strategy.totalPnLPercent)}
          </BadgeWithDot>
        </div>
      </div>

      <div className="flex items-center gap-2 pt-3 border-t border-primary flex-wrap">
        <BadgeWithDot color="gray" size="sm">
          {formatMarketType(strategy.marketType)}
        </BadgeWithDot>
        <BadgeWithDot color="gray" size="sm">
          {formatRiskTolerance(strategy.riskTolerance)}
        </BadgeWithDot>
      </div>
    </div>
  );
}
