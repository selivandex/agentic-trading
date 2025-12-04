/** @format */

import { UserAvatar } from "@/entities/user/ui/UserAvatar";
import { BadgeWithDot } from "@/shared/base";
import type { User } from "@/entities/user/model/types";
import { formatUserDisplayName } from "@/entities/user/lib/formatters";

interface UserCardProps {
  user: User;
  showEmail?: boolean;
  showStatus?: boolean;
  className?: string;
}

/**
 * UserCard component
 *
 * Displays user information in a card format
 */
export function UserCard({
  user,
  showEmail = true,
  showStatus = false,
  className = ""
}: UserCardProps) {
  const displayName = formatUserDisplayName(user);

  return (
    <div className={`flex items-center gap-3 ${className}`}>
      <UserAvatar user={user} size="md" />

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <p className="text-sm font-medium text-primary truncate">
            {displayName}
          </p>
          {user.isPremium && (
            <BadgeWithDot color="warning" size="sm">
              Premium
            </BadgeWithDot>
          )}
        </div>

        {showEmail && user.email && (
          <p className="text-sm text-secondary truncate">
            {user.email}
          </p>
        )}

        {showStatus && (
          <BadgeWithDot
            color={user.isActive ? "success" : "gray"}
            size="sm"
          >
            {user.isActive ? "Active" : "Inactive"}
          </BadgeWithDot>
        )}
      </div>
    </div>
  );
}
