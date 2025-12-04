/**
 * Optimistic Updates Helpers
 *
 * Helpers for creating optimistic responses for mutations.
 * Optimistic updates make UI feel instant by updating cache before server responds.
 *
 * @format
 */

import type {
  Scenario,
  ScenarioStatus,
  Project,
  Organization,
} from "./generated/graphql";

/**
 * Create optimistic response for scenario update
 */
export const optimisticUpdateScenario = (
  scenarioId: string,
  updates: Partial<Scenario>
): { updateScenario: Partial<Scenario> } => {
  return {
    updateScenario: {
      __typename: "Scenario",
      id: scenarioId,
      ...updates,
    },
  };
};

/**
 * Create optimistic response for scenario status change
 */
export const optimisticScenarioStatusChange = (
  scenarioId: string,
  status: ScenarioStatus
): { activateScenario: Partial<Scenario> } | { pauseScenario: Partial<Scenario> } | { archiveScenario: Partial<Scenario> } => {
  const baseScenario: Partial<Scenario> = {
    __typename: "Scenario",
    id: scenarioId,
    status,
    updatedAt: new Date().toISOString(),
  };

  if (status === "ACTIVE") {
    return { activateScenario: baseScenario };
  } else if (status === "PAUSED") {
    return { pauseScenario: baseScenario };
  } else {
    return { archiveScenario: baseScenario };
  }
};

/**
 * Create optimistic response for scenario deletion
 */
export const optimisticDeleteScenario = (
  scenarioId: string
): { deleteScenario: { id: string; __typename: string } } => {
  return {
    deleteScenario: {
      __typename: "Scenario",
      id: scenarioId,
    },
  };
};

/**
 * Create optimistic response for project update
 */
export const optimisticUpdateProject = (
  projectId: string,
  updates: Partial<Project>
): { updateProject: Partial<Project> } => {
  return {
    updateProject: {
      __typename: "Project",
      id: projectId,
      ...updates,
      updatedAt: new Date().toISOString(),
    },
  };
};

/**
 * Create optimistic response for organization update
 */
export const optimisticUpdateOrganization = (
  organizationId: string,
  updates: Partial<Organization>
): { updateOrganization: Partial<Organization> } => {
  return {
    updateOrganization: {
      __typename: "Organization",
      id: organizationId,
      ...updates,
      updatedAt: new Date().toISOString(),
    },
  };
};

/**
 * Generic helper for optimistic updates
 */
export const createOptimisticResponse = <T extends { __typename: string; id: string }>(
  mutationName: string,
  typename: string,
  id: string,
  updates: Partial<T>
): Record<string, Partial<T>> => {
  return {
    [mutationName]: {
      __typename: typename,
      id,
      ...updates,
    },
  };
};

