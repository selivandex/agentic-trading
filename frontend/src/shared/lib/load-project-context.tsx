/**
 * Load Project Context Hook
 *
 * Client-side hook to load organization and project data using Apollo Client.
 * Replaces server-side fetch with proper Apollo cache integration.
 *
 * @format
 */

"use client";

import { useMemo } from "react";
import { useQuery } from "@apollo/client";
import { GET_CURRENT_CONTEXT_QUERY } from "@/shared/api";
import type { GetCurrentContextQuery } from "@/shared/api";
import type { OrganizationDTO, ProjectDTO } from "./app-context";
import { isContextError } from "@/shared/api/graphql-error-codes";

interface UseProjectContextOptions {
  organizationId: string;
  projectId: string;
}

interface UseProjectContextResult {
  organization: OrganizationDTO | null;
  project: ProjectDTO | null;
  projects: ProjectDTO[];
  loading: boolean;
  error: Error | null;
}

/**
 * Hook to load project context from GraphQL
 *
 * Uses Apollo Client for proper caching and real-time updates.
 * Automatically sets organization/project headers via ApolloContextSetter.
 *
 * @example
 * ```tsx
 * const { organization, project, loading, error } = useProjectContext({
 *   organizationId: params.organization_id,
 *   projectId: params.project_id,
 * });
 *
 * if (loading) return <LoadingSpinner />;
 * if (error) return <ErrorMessage error={error} />;
 *
 * return <ProjectLayout organization={organization} project={project} />;
 * ```
 */
export const useProjectContext = ({
  organizationId,
  projectId,
}: UseProjectContextOptions): UseProjectContextResult => {
  // Query with Apollo Client
  const {
    data,
    loading,
    error: queryError,
  } = useQuery<GetCurrentContextQuery>(GET_CURRENT_CONTEXT_QUERY, {
    variables: {
      first: 100, // Load first 100 projects for sidebar
      // Add org/project as variables to make them part of cache key
      // These are not used by GraphQL query but ensure proper cache separation
      __cacheKey_organizationId: organizationId,
      __cacheKey_projectId: projectId,
    },
    // Use cache-first for instant loading from localStorage
    fetchPolicy: "cache-first",
    // Skip query if IDs are not provided
    skip: !organizationId || !projectId,
  });

  // Extract and validate organization and project
  const result = useMemo(() => {
    const organization = data?.organization ?? null;

    // Find current project from organization's projects list
    const projects =
      data?.organization?.projects?.edges
        ?.map((edge) => edge?.node)
        .filter((node): node is NonNullable<typeof node> => node != null) ?? [];

    const project =
      projects.find((p) => p.slug === projectId || p.id === projectId) ?? null;

    // Validate and create error if needed
    let error: Error | null = null;

    // Handle GraphQL errors from query
    if (queryError) {
      // Check if it's a context error (organization/project not found)
      const isContextErr = queryError.graphQLErrors?.some((err) =>
        isContextError(err.extensions?.code as string)
      );

      if (isContextErr) {
        // Use original error message from backend
        const contextError = queryError.graphQLErrors.find((err) =>
          isContextError(err.extensions?.code as string)
        );

        error = new Error(contextError?.message || "Context not found");
      } else {
        // For non-GraphQL errors (network errors, 404, etc.)
        // Try to determine the context from error message or use generic message
        const errorMessage = queryError.message.toLowerCase();

        // If it's a 404 or similar error, assume it's organization not found
        // since that's the first context that gets checked
        if (
          errorMessage.includes("404") ||
          errorMessage.includes("not found")
        ) {
          error = new Error("Organization not found");
        } else {
          error = new Error(`Failed to load context: ${queryError.message}`);
        }
      }
    }

    // Validate data only if no GraphQL error occurred
    if (!loading && data && !error) {
      if (!organization) {
        error = new Error(
          `Organization not found. Please provide valid X-Flowly-Organization header.`
        );
      } else if (!project) {
        error = new Error(
          `Project not found. Please provide valid X-Flowly-Project header.`
        );
      }
    }

    return {
      organization,
      project,
      projects,
      error,
    };
  }, [data, loading, queryError, projectId]);

  return {
    ...result,
    loading,
  };
};
