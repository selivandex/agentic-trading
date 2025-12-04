/** @format */

"use client";

import { createContext, useContext, type ReactNode } from "react";

/**
 * Generic Organization Type
 *
 * Infrastructure DTO for organization data.
 * Real domain types are defined in entities layer.
 */
export interface OrganizationDTO {
  id: string;
  name: string;
  slug: string;
  timezone?: string | null;
  createdAt: string;
  updatedAt: string;
}

/**
 * Generic Project Type
 *
 * Infrastructure DTO for project data.
 * Real domain types are defined in entities layer.
 */
export interface ProjectDTO {
  id: string;
  name: string;
  slug: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * Application Context Data
 *
 * Generic context for providing organization and project to the app.
 * This is infrastructure layer - contains no business logic, only data transport.
 *
 * Values can be null during initial loading - components should handle this state.
 */
export interface AppContextData {
  organization: OrganizationDTO | null;
  project: ProjectDTO | null;
  projects: ProjectDTO[];
}

const AppContext = createContext<AppContextData | null>(null);

interface AppContextProviderProps {
  children: ReactNode;
  organization: OrganizationDTO | null;
  project: ProjectDTO | null;
  projects?: ProjectDTO[];
}

/**
 * Application Context Provider
 *
 * Provides organization and project data to all components.
 * Data is loaded client-side using GraphQL query.
 *
 * Accepts null values during initial loading - components can check for null and show loading state.
 *
 * @example
 * ```tsx
 * // In layout.tsx
 * const { organization, project } = useProjectContext(organizationId, projectId);
 *
 * return (
 *   <AppContextProvider organization={organization} project={project}>
 *     {children}
 *   </AppContextProvider>
 * );
 * ```
 */
export const AppContextProvider = ({
  children,
  organization,
  project,
  projects = [],
}: AppContextProviderProps) => {
  return (
    <AppContext.Provider
      value={{
        organization,
        project,
        projects,
      }}
    >
      {children}
    </AppContext.Provider>
  );
};

/**
 * Hook to get application context (organization and project)
 *
 * Low-level hook that returns full app context.
 * For most cases, use `useOrganization()` or `useProject()` from entities instead.
 *
 * @example
 * ```tsx
 * // ✅ Preferred - use entity hooks
 * import { useOrganization } from "@/entities/organization";
 * import { useProject } from "@/entities/project";
 *
 * // ⚠️ Only use useAppContext if you need both at once
 * const { organization, project } = useAppContext();
 * const basePath = `/${organization.slug}/${project.slug}`;
 * ```
 */
export const useAppContext = (): AppContextData => {
  const context = useContext(AppContext);

  if (!context) {
    throw new Error("useAppContext must be used within AppContextProvider");
  }

  return context;
};
