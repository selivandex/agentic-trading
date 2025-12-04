/** @format */

import { Avatar } from "@/shared/base";
import type { User } from "@/entities/user/model/types";
import { getUserInitials, formatUserDisplayName } from "@/entities/user/lib/formatters";

interface UserAvatarProps {
  user: User;
  size?: "xxs" | "xs" | "sm" | "md" | "lg" | "xl" | "2xl";
  className?: string;
}

/**
 * UserAvatar component
 *
 * Displays user avatar with initials fallback
 */
export function UserAvatar({ user, size = "md", className }: UserAvatarProps) {
  const initials = getUserInitials(user);
  const displayName = formatUserDisplayName(user);

  return (
    <Avatar
      size={size}
      className={className}
      initials={initials}
      alt={displayName}
    />
  );
}
