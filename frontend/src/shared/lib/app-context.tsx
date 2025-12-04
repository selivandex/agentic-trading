/** @format */

"use client";

import { createContext, useContext, type ReactNode } from "react";
import type { User } from "@/entities/user";

/**
 * Application Context Data
 *
 * Simple app context for trading platform.
 * Contains current user and app-wide state.
 */
export interface AppContextData {
  user: User | null;
}

const AppContext = createContext<AppContextData | null>(null);

interface AppContextProviderProps {
  children: ReactNode;
  user?: User | null;
}

/**
 * Application Context Provider
 *
 * Provides app-wide data to all components.
 * Currently only user data is needed.
 *
 * @example
 * ```tsx
 * // In layout.tsx
 * <AppContextProvider user={session.user}>
 *   {children}
 * </AppContextProvider>
 * ```
 */
export const AppContextProvider = ({
  children,
  user = null,
}: AppContextProviderProps) => {
  return (
    <AppContext.Provider
      value={{
        user,
      }}
    >
      {children}
    </AppContext.Provider>
  );
};

/**
 * Hook to get application context
 *
 * Returns current user and app state.
 * For user-specific logic, prefer using `useMe()` from @/entities/user.
 *
 * @example
 * ```tsx
 * const { user } = useAppContext();
 * ```
 */
export const useAppContext = (): AppContextData => {
  const context = useContext(AppContext);

  if (!context) {
    throw new Error("useAppContext must be used within AppContextProvider");
  }

  return context;
};
