/**
 * Authentication hooks for React components
 *
 * Uses Next-Auth session that includes full user data with memberships.
 * Session is automatically populated from GraphQL in auth callbacks.
 *
 * @format
 */

import { useSession, signOut as nextAuthSignOut } from "next-auth/react";
import { useCallback } from "react";
import { clearCache } from "@/shared/api";

/**
 * Hook to get current authenticated user from Next-Auth session
 *
 * This returns full user data including memberships from Next-Auth session.
 * Data is automatically fetched and cached in JWT callbacks.
 */
export const useCurrentUser = () => {
  const { data: session, status, update } = useSession();
  const isAuthenticated = status === "authenticated";

  return {
    user: session?.user ?? null,
    loading: status === "loading",
    error: null,
    refetch: update, // Trigger session refresh
    isAuthenticated,
  };
};

/**
 * Hook for sign out with Apollo cache cleanup and context clearing
 *
 * Wraps Next-Auth signOut to also clear Apollo cache and last context cookies.
 * This prevents stale data from showing after logout.
 *
 * @example
 * ```tsx
 * const { signOut } = useSignOut();
 *
 * await signOut({ callbackUrl: "/login" });
 * ```
 */
export const useSignOut = () => {
  const signOut = useCallback(
    async (options?: Parameters<typeof nextAuthSignOut>[0]) => {
      // Clear last context cookies
      try {
        await fetch("/api/clear-context", { method: "POST" });
      } catch (error) {
        console.error("Failed to clear context cookies:", error);
      }

      // Clear Apollo cache before signing out
      // This prevents stale data from previous session
      clearCache();

      // Sign out from Next-Auth
      await nextAuthSignOut(options);
    },
    []
  );

  return { signOut };
};

/**
 * Combined auth hook with all authentication functionality
 *
 * This is the main hook for authentication in the app.
 *
 * @example
 * ```tsx
 * import { useAuth } from "@/features/auth";
 *
 * const { user, isAuthenticated, signOut } = useAuth();
 *
 * // Sign out
 * await signOut({ callbackUrl: "/login" });
 * ```
 *
 * For sign in/sign up, use Next-Auth directly:
 * ```tsx
 * import { signIn } from "next-auth/react";
 *
 * await signIn("credentials", { email, password });
 * ```
 */
export const useAuth = () => {
  const { user, loading, error, refetch, isAuthenticated } = useCurrentUser();
  const { signOut } = useSignOut();

  return {
    user,
    loading,
    error,
    isAuthenticated,
    refetch,
    signOut,
  };
};



