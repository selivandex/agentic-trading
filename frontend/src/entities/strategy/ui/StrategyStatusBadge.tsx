/** @format */

import { BadgeWithDot } from "@/shared/base";
import type { StrategyStatus } from "@/entities/strategy";
import {
  formatStrategyStatus,
  getStrategyStatusColor,
} from "@/entities/strategy";

interface StrategyStatusBadgeProps {
  status: StrategyStatus;
  size?: "sm" | "md" | "lg";
  className?: string;
}

/**
 * StrategyStatusBadge component
 *
 * Displays strategy status as a colored badge
 */
export function StrategyStatusBadge({
  status,
  size = "sm",
  className = "",
}: StrategyStatusBadgeProps) {
  const color = getStrategyStatusColor(status);

  return (
    <BadgeWithDot color={color} size={size} className={className}>
      {formatStrategyStatus(status)}
    </BadgeWithDot>
  );
}
