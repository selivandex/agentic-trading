/**
 * Apollo Context Setter
 *
 * @deprecated This module is no longer needed. Organization and project IDs are now
 * automatically extracted from URL pathname by Apollo Client's authLink.
 *
 * Kept for backward compatibility with existing code, but new code should not use this.
 * Headers (X-Flowly-Organization, X-Flowly-Project) are automatically added based on URL.
 *
 * @format
 */

"use client";

import { type ReactNode, useEffect } from "react";

interface ApolloContextSetterProps {
  children: ReactNode;
  organizationId: string;
  projectId: string;
}

/**
 * Global context values for Apollo Client
 * These are updated by ApolloContextSetter and read by authLink
 */
let currentOrganizationId: string | null = null;
let currentProjectId: string | null = null;

/**
 * Get current organization ID (used by authLink)
 */
export const getCurrentOrganizationId = (): string | null =>
  currentOrganizationId;

/**
 * Get current project ID (used by authLink)
 */
export const getCurrentProjectId = (): string | null => currentProjectId;

/**
 * Set Apollo context programmatically
 *
 * Use this to set organization/project context before making queries.
 * This is needed when context must be set before components render.
 *
 * @example
 * ```tsx
 * // In layout before useQuery
 * useEffect(() => {
 *   setApolloContext(organizationId, projectId);
 * }, [organizationId, projectId]);
 * ```
 */
export const setApolloContext = (
  organizationId: string | null,
  projectId: string | null
): void => {
  currentOrganizationId = organizationId;
  currentProjectId = projectId;
};

/**
 * Apollo Context Setter Component
 *
 * Sets organization and project IDs for all GraphQL requests.
 * Headers are automatically added by authLink in apollo-client-base.ts
 *
 * @example
 * ```tsx
 * <ApolloContextSetter
 *   organizationId="acme-corp"
 *   projectId="mobile-app"
 * >
 *   {children}
 * </ApolloContextSetter>
 * ```
 */
export const ApolloContextSetter = ({
  children,
  organizationId,
  projectId,
}: ApolloContextSetterProps) => {
  // Update global context values when props change
  useEffect(() => {
    currentOrganizationId = organizationId;
    currentProjectId = projectId;

    // Clear when unmounting
    return () => {
      currentOrganizationId = null;
      currentProjectId = null;
    };
  }, [organizationId, projectId]);

  return <>{children}</>;
};
