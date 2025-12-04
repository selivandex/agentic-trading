/** @format */

import { BadgeWithDot } from "@/shared/base";
import type { FundWatchlist } from "@/entities/fund-watchlist/model/types";
import {
  formatTier,
  getTierColor,
  getWatchlistStatusColor,
  getWatchlistStatusText,
  formatMarketType,
  formatCategory,
} from "@/entities/fund-watchlist/lib/formatters";

interface WatchlistItemCardProps {
  item: FundWatchlist;
  onClick?: () => void;
  className?: string;
}

/**
 * WatchlistItemCard component
 *
 * Displays fund watchlist item in a card format
 */
export function WatchlistItemCard({
  item,
  onClick,
  className = "",
}: WatchlistItemCardProps) {
  const statusColor = getWatchlistStatusColor(item);
  const statusText = getWatchlistStatusText(item);
  const tierColor = getTierColor(item.tier);

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
            {item.symbol}
          </h3>
          <p className="text-sm text-secondary mt-1">
            {formatMarketType(item.marketType)}
          </p>
        </div>

        <BadgeWithDot color={statusColor} size="sm">
          {statusText}
        </BadgeWithDot>
      </div>

      <div className="flex items-center gap-2 flex-wrap">
        <BadgeWithDot color={tierColor} size="sm">
          {formatTier(item.tier)}
        </BadgeWithDot>

        <BadgeWithDot color="gray" size="sm">
          {formatCategory(item.category)}
        </BadgeWithDot>
      </div>

      {item.isPaused && item.pausedReason && (
        <p className="text-xs text-tertiary mt-3 pt-3 border-t border-primary">
          {item.pausedReason}
        </p>
      )}
    </div>
  );
}
