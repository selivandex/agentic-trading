/** @format */

"use client";

import { type ReactNode } from "react";
import {
  AppContextProvider,
  type OrganizationDTO,
  type ProjectDTO,
} from "./app-context";
import { ApolloContextSetter } from "@/shared/api";

/**
 * Project Provider
 *
 * Provides organization and project context to all child components.
 * Organization and project data can be loaded client-side via useProjectContext().
 *
 * Sets Apollo Client context headers automatically for all GraphQL queries:
 * - X-Flowly-Organization: organization.slug
 * - X-Flowly-Project: project.slug
 *
 * All GraphQL queries/mutations automatically include these headers without manual passing.
 *
 * Accepts null values during initial loading - components can check for null and show loading state.
 *
 * @example
 * ```tsx
 * // In layout.tsx
 * const { organization, project } = useProjectContext(orgId, projectId);
 *
 * return (
 *   <ProjectProvider organization={organization} project={project}>
 *     {children}
 *   </ProjectProvider>
 * );
 * ```
 */

interface ProjectProviderProps {
  children: ReactNode;
  organization: OrganizationDTO | null;
  project: ProjectDTO | null;
  projects?: ProjectDTO[];
}

export const ProjectProvider = ({
  children,
  organization,
  project,
  projects = [],
}: ProjectProviderProps) => {
  return (
    <ApolloContextSetter
      organizationId={organization?.slug ?? ""}
      projectId={project?.slug ?? ""}
    >
      <AppContextProvider organization={organization} project={project} projects={projects}>
        {children}
      </AppContextProvider>
    </ApolloContextSetter>
  );
};
