/** @format */

import type { User } from "@/entities/user";

/**
 * Format user full name
 */
export function formatUserFullName(user: User): string {
  return `${user.firstName} ${user.lastName}`.trim();
}

/**
 * Format user display name (fallback to email or telegram username)
 */
export function formatUserDisplayName(user: User): string {
  const fullName = formatUserFullName(user);
  if (fullName) return fullName;

  if (user.email) return user.email;
  if (user.telegramUsername) return `@${user.telegramUsername}`;

  return "Unknown User";
}

/**
 * Get user initials for avatar
 */
export function getUserInitials(user: User): string {
  const firstInitial = user.firstName?.charAt(0).toUpperCase() || "";
  const lastInitial = user.lastName?.charAt(0).toUpperCase() || "";

  if (firstInitial && lastInitial) {
    return `${firstInitial}${lastInitial}`;
  }

  if (firstInitial) return firstInitial;
  if (user.email) return user.email.charAt(0).toUpperCase();

  return "?";
}

/**
 * Format risk level display name
 */
export function formatRiskLevel(riskLevel: string): string {
  const map: Record<string, string> = {
    CONSERVATIVE: "Conservative",
    MODERATE: "Moderate",
    AGGRESSIVE: "Aggressive",
  };

  return map[riskLevel] || riskLevel;
}
