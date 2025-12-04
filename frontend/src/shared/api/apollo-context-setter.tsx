/**
 * Apollo Context Setter (Simplified)
 *
 * Simple pass-through component for trading platform.
 * No organization/project headers needed.
 *
 * @format
 */

"use client";

import { type ReactNode } from "react";

interface ApolloContextSetterProps {
  children: ReactNode;
}

/**
 * Apollo Context Setter Component
 *
 * Simplified version for trading platform.
 * Just passes children through, no special headers needed.
 *
 * @example
 * ```tsx
 * <ApolloContextSetter>
 *   {children}
 * </ApolloContextSetter>
 * ```
 */
export const ApolloContextSetter = ({ children }: ApolloContextSetterProps) => {
  return <>{children}</>;
};
