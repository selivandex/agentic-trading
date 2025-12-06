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
  BATCH_DELETE_STRATEGIES_MUTATION,
  BATCH_PAUSE_STRATEGIES_MUTATION,
  BATCH_RESUME_STRATEGIES_MUTATION,
  BATCH_CLOSE_STRATEGIES_MUTATION,
  type Strategy,
  type StrategyStatus,
  type CreateStrategyInput,
  type UpdateStrategyInput,
} from "@/entities/strategy";

/**
 * Hook to get strategy by ID
 */
export const useStrategy = (id: string) => {
  return useQuery<{ strategy: Strategy | null }>(GET_STRATEGY_QUERY, {
    variables: { id },
    skip: !id,
  });
};

/**
 * Hook to get user strategies
 */
export const useUserStrategies = (
  userID: string,
  first?: number,
  after?: string,
  status?: StrategyStatus,
  search?: string
) => {
  return useQuery<{
    userStrategies: {
      edges: Array<{ node: Strategy; cursor: string }>;
      pageInfo: {
        hasNextPage: boolean;
        hasPreviousPage: boolean;
        startCursor?: string;
        endCursor?: string;
      };
      totalCount: number;
    };
  }>(GET_USER_STRATEGIES_QUERY, {
    variables: { userID, first, after, status, search },
    skip: !userID,
  });
};

/**
 * Hook to get all strategies
 */
export const useAllStrategies = (
  first?: number,
  after?: string,
  status?: StrategyStatus,
  search?: string
) => {
  return useQuery<{
    strategies: {
      edges: Array<{ node: Strategy; cursor: string }>;
      pageInfo: {
        hasNextPage: boolean;
        hasPreviousPage: boolean;
        startCursor?: string;
        endCursor?: string;
      };
      totalCount: number;
    };
  }>(GET_ALL_STRATEGIES_QUERY, {
    variables: { first, after, status, search },
  });
};

/**
 * Hook to create strategy
 */
export const useCreateStrategy = () => {
  return useMutation<
    { createStrategy: Strategy },
    { userID: string; input: CreateStrategyInput }
  >(CREATE_STRATEGY_MUTATION);
};

/**
 * Hook to update strategy
 */
export const useUpdateStrategy = () => {
  return useMutation<
    { updateStrategy: Strategy },
    { id: string; input: UpdateStrategyInput }
  >(UPDATE_STRATEGY_MUTATION);
};

/**
 * Hook to pause strategy
 */
export const usePauseStrategy = () => {
  return useMutation<{ pauseStrategy: Strategy }, { id: string }>(
    PAUSE_STRATEGY_MUTATION
  );
};

/**
 * Hook to resume strategy
 */
export const useResumeStrategy = () => {
  return useMutation<{ resumeStrategy: Strategy }, { id: string }>(
    RESUME_STRATEGY_MUTATION
  );
};

/**
 * Hook to close strategy
 */
export const useCloseStrategy = () => {
  return useMutation<{ closeStrategy: Strategy }, { id: string }>(
    CLOSE_STRATEGY_MUTATION
  );
};

/**
 * Hook to batch delete strategies (hard delete, only for closed strategies)
 */
export const useBatchDeleteStrategies = () => {
  return useMutation<{ batchDeleteStrategies: number }, { ids: string[] }>(
    BATCH_DELETE_STRATEGIES_MUTATION
  );
};

/**
 * Hook to batch pause strategies
 */
export const useBatchPauseStrategies = () => {
  return useMutation<{ batchPauseStrategies: number }, { ids: string[] }>(
    BATCH_PAUSE_STRATEGIES_MUTATION
  );
};

/**
 * Hook to batch resume strategies
 */
export const useBatchResumeStrategies = () => {
  return useMutation<{ batchResumeStrategies: number }, { ids: string[] }>(
    BATCH_RESUME_STRATEGIES_MUTATION
  );
};

/**
 * Hook to batch close strategies
 */
export const useBatchCloseStrategies = () => {
  return useMutation<{ batchCloseStrategies: number }, { ids: string[] }>(
    BATCH_CLOSE_STRATEGIES_MUTATION
  );
};
