/** @format */

"use client";

import { type ReactNode } from "react";
import { AppContextProvider } from "./app-context";
import { ApolloContextSetter } from "@/shared/api";
import type { User } from "@/entities/user";

/**
 * App Provider (simplified)
 *
 * Provides app context to all child components.
 * Simplified for trading platform - no organizations/projects.
 *
 * @example
 * ```tsx
 * // In layout.tsx
 * <AppProvider user={session.user}>
 *   {children}
 * </AppProvider>
 * ```
 */

interface AppProviderProps {
  children: ReactNode;
  user?: User | null;
}

export const AppProvider = ({
  children,
  user = null,
}: AppProviderProps) => {
  return (
    <ApolloContextSetter>
      <AppContextProvider user={user}>
        {children}
      </AppContextProvider>
    </ApolloContextSetter>
  );
};

// Export for backward compatibility
export { AppProvider as ProjectProvider };
