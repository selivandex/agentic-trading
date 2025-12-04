/**
 * Authentication Provider
 * 
 * Provides authentication context using Next-Auth.
 * Session is managed by Next-Auth with JWT strategy.
 */

"use client";

import { SessionProvider } from "next-auth/react";
import type { ReactNode } from "react";

/**
 * Auth Provider component
 * Wraps the application with Next-Auth SessionProvider
 */
export const AuthProvider = ({ children }: { children: ReactNode }) => {
  return <SessionProvider>{children}</SessionProvider>;
};

