/**
 * Route Parameters Context for Apollo Client
 *
 * Provides organization_id and project_id from URL route params to Apollo Client.
 * Used by apollo-client-base.ts for route parameter context.
 *
 * This is part of shared/api layer as it's specifically for Apollo Client header injection.
 *
 * @format
 */

"use client";

import { createContext, useContext, useEffect, type ReactNode } from "react";

interface RouteParamsContextValue {
  organizationId: string | null;
  projectId: string | null;
}

const RouteParamsContext = createContext<RouteParamsContextValue>({
  organizationId: null,
  projectId: null,
});

interface RouteParamsProviderProps {
  children: ReactNode;
  organizationId?: string;
  projectId?: string;
}

/**
 * Provider for route parameters (organization_id, project_id)
 *
 * Used in project layout to expose route params to Apollo Client.
 * Apollo's authLink reads these values to add headers to GraphQL requests.
 *
 * @example
 * ```tsx
 * // In project layout
 * <RouteParamsProvider
 *   organizationId={params.organization_id}
 *   projectId={params.project_id}
 * >
 *   {children}
 * </RouteParamsProvider>
 * ```
 */
export const RouteParamsProvider = ({
  children,
  organizationId,
  projectId,
}: RouteParamsProviderProps) => {
  // Update global variable when props change
  // This ensures Apollo authLink has access to current route params
  useEffect(() => {
    currentRouteParams = {
      organizationId: organizationId ?? null,
      projectId: projectId ?? null,
    };

    // Cleanup on unmount
    return () => {
      currentRouteParams = {
        organizationId: null,
        projectId: null,
      };
    };
  }, [organizationId, projectId]);

  return (
    <RouteParamsContext.Provider
      value={{
        organizationId: organizationId ?? null,
        projectId: projectId ?? null,
      }}
    >
      {children}
    </RouteParamsContext.Provider>
  );
};

/**
 * Hook to access current route parameters
 *
 * Returns null if used outside of RouteParamsProvider (e.g., auth routes)
 */
export const useRouteParams = (): RouteParamsContextValue => {
  return useContext(RouteParamsContext);
};

/**
 * Get current route parameters synchronously
 *
 * Used by Apollo authLink to add headers to GraphQL requests.
 * Returns null if not in app context (e.g., auth routes).
 */
let currentRouteParams: RouteParamsContextValue = {
  organizationId: null,
  projectId: null,
};

export const getCurrentRouteParams = (): RouteParamsContextValue => {
  return currentRouteParams;
};

export const setCurrentRouteParams = (
  params: RouteParamsContextValue
): void => {
  currentRouteParams = params;
};
