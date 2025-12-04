/** @format */

"use client";

import { useCurrentUser } from "@/features/auth/api";
import { signOut } from "next-auth/react";
import { clearCache } from "@/shared/api";

/**
 * User menu component with logout button
 * Shows user email and provides logout functionality
 */
export const UserMenu = () => {
  const { user, loading } = useCurrentUser();

  if (loading) {
    return (
      <div className="flex items-center gap-2">
        <div className="h-8 w-8 bg-gray-200 rounded-full animate-pulse" />
        <div className="h-4 w-24 bg-gray-200 rounded animate-pulse" />
      </div>
    );
  }

  if (!user) {
    return null;
  }

  const displayName =
    user.firstName && user.lastName
      ? `${user.firstName} ${user.lastName}`
      : user.email;

  const initials =
    user.firstName && user.lastName
      ? `${user.firstName[0]}${user.lastName[0]}`.toUpperCase()
      : user.email.substring(0, 2).toUpperCase();

  return (
    <div className="flex items-center gap-3">
      {/* User Avatar */}
      <div className="flex items-center gap-2">
        <div className="h-8 w-8 bg-blue-600 text-white rounded-full flex items-center justify-center text-sm font-medium">
          {initials}
        </div>
        <span className="text-sm font-medium text-gray-700">{displayName}</span>
      </div>

      {/* Logout Button */}
      <button
        onClick={async () => {
          // Clear Apollo cache before logout
          await clearCache();
          // Sign out and redirect to login
          signOut({ callbackUrl: "/login" });
        }}
        className="px-3 py-1.5 text-sm text-gray-700 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
      >
        Sign out
      </button>
    </div>
  );
};
