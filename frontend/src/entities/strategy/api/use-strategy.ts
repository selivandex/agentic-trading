/** @format */

import { useQuery, useMutation } from "@apollo/client";
import {
  GET_STRATEGY_QUERY,
  GET_USER_STRATEGIES_QUERY,
  GET_ALL_STRATEGIES_QUERY,
  CREATE_STRATEGY_MUTATION,
  UPDATE_STRATEGY_MUTATION,
  PAUSE_STRATEGY_MUTATION,
  RESUME_STRATEGY_MUTATION,
  CLOSE_STRATEGY_MUTATION,
} from "@/entities/strategy/api/strategy.graphql";
import type {
  Strategy,
  StrategyStatus,
  CreateStrategyInput,
  UpdateStrategyInput,
} from "@/entities/strategy/model/types";

/**
 * Hook to get strategy by ID
 */
export function useStrategy(id: string) {
  return useQuery<{ strategy: Strategy | null }>(GET_STRATEGY_QUERY, {
    variables: { id },
    skip: !id,
  });
}

/**
 * Hook to get user strategies
 */
export function useUserStrategies(userID: string, status?: StrategyStatus) {
  return useQuery<{ userStrategies: Strategy[] }>(GET_USER_STRATEGIES_QUERY, {
    variables: { userID, status },
    skip: !userID,
  });
}

/**
 * Hook to get all strategies (admin)
 */
export function useAllStrategies(
  limit?: number,
  offset?: number,
  status?: StrategyStatus
) {
  return useQuery<{ strategies: Strategy[] }>(GET_ALL_STRATEGIES_QUERY, {
    variables: { limit, offset, status },
  });
}

/**
 * Hook to create strategy
 */
export function useCreateStrategy() {
  return useMutation<
    { createStrategy: Strategy },
    { userID: string; input: CreateStrategyInput }
  >(CREATE_STRATEGY_MUTATION, {
    refetchQueries: ["GetUserStrategies"],
  });
}

/**
 * Hook to update strategy
 */
export function useUpdateStrategy() {
  return useMutation<
    { updateStrategy: Strategy },
    { id: string; input: UpdateStrategyInput }
  >(UPDATE_STRATEGY_MUTATION);
}

/**
 * Hook to pause strategy
 */
export function usePauseStrategy() {
  return useMutation<{ pauseStrategy: Strategy }, { id: string }>(
    PAUSE_STRATEGY_MUTATION
  );
}

/**
 * Hook to resume strategy
 */
export function useResumeStrategy() {
  return useMutation<{ resumeStrategy: Strategy }, { id: string }>(
    RESUME_STRATEGY_MUTATION
  );
}

/**
 * Hook to close strategy
 */
export function useCloseStrategy() {
  return useMutation<{ closeStrategy: Strategy }, { id: string }>(
    CLOSE_STRATEGY_MUTATION,
    {
      refetchQueries: ["GetUserStrategies"],
    }
  );
}
