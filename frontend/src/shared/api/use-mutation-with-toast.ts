/**
 * Mutation Hook with Toast Notifications
 *
 * Wraps Apollo mutations with automatic success/error toast notifications.
 * Provides consistent error handling across the application.
 *
 * @format
 */

import { useCallback } from "react";
import type { ApolloError, MutationResult } from "@apollo/client";
import { toast } from "@/shared/ui/toast";

interface MutationMessages {
  loading?: string;
  success: string;
  error?: string | ((error: ApolloError) => string);
}

interface UseMutationWithToastOptions<TData, TVariables> {
  mutation: (variables: TVariables) => Promise<{ data?: TData }>;
  messages: MutationMessages;
  onSuccess?: (data: TData) => void;
  onError?: (error: ApolloError) => void;
}

/**
 * Hook that wraps mutation with toast notifications
 *
 * @example
 * ```typescript
 * const { deleteScenario } = useDeleteScenario();
 *
 * const deleteWithToast = useMutationWithToast({
 *   mutation: deleteScenario,
 *   messages: {
 *     loading: "Deleting scenario...",
 *     success: "Scenario deleted successfully",
 *     error: "Failed to delete scenario"
 *   }
 * });
 *
 * await deleteWithToast({ variables: { scenarioId } });
 * ```
 */
export const useMutationWithToast = <TData = unknown, TVariables = unknown>({
  mutation,
  messages,
  onSuccess,
  onError,
}: UseMutationWithToastOptions<TData, TVariables>) => {
  const execute = useCallback(
    async (variables: TVariables) => {
      let toastId: string | number | undefined;

      try {
        // Show loading toast if message provided
        if (messages.loading) {
          toastId = toast.loading(messages.loading);
        }

        // Execute mutation
        const result = await mutation(variables);

        if (!result.data) {
          throw new Error("Mutation returned no data");
        }

        // Dismiss loading toast
        if (toastId) {
          toast.dismiss(toastId);
        }

        // Show success toast
        toast.success(messages.success);

        // Call success callback
        if (onSuccess) {
          onSuccess(result.data);
        }

        return result;
      } catch (error) {
        // Dismiss loading toast
        if (toastId) {
          toast.dismiss(toastId);
        }

        // Get error message
        const errorMessage =
          typeof messages.error === "function"
            ? messages.error(error as ApolloError)
            : messages.error ?? getDefaultErrorMessage(error);

        // Show error toast
        toast.error(errorMessage);

        // Call error callback
        if (onError && error instanceof Error) {
          onError(error as ApolloError);
        }

        throw error;
      }
    },
    [mutation, messages, onSuccess, onError]
  );

  return execute;
};

/**
 * Extract user-friendly error message from Apollo error
 */
function getDefaultErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    // Check for GraphQL errors
    const apolloError = error as ApolloError;
    if (apolloError.graphQLErrors?.length > 0) {
      return apolloError.graphQLErrors[0].message;
    }

    // Check for network errors
    if (apolloError.networkError) {
      return "Network error. Please check your connection.";
    }

    // Return generic error message
    return error.message;
  }

  return "An unexpected error occurred";
}

/**
 * Type helper for mutation hook result
 */
export type MutationWithToast<TData, TVariables> = (
  variables: TVariables
) => Promise<{ data?: TData }>;

/**
 * Helper to create mutation with toast for common patterns
 */
export const createMutationWithToast = <TData, TVariables>(
  useMutationHook: () => [
    (options: { variables: TVariables }) => Promise<{ data?: TData }>,
    MutationResult<TData>
  ],
  messages: MutationMessages
) => {
  return () => {
    const [mutationFn, mutationResult] = useMutationHook();

    const execute = useMutationWithToast({
      mutation: async (variables: TVariables) => {
        return await mutationFn({ variables });
      },
      messages,
    });

    return {
      execute,
      ...mutationResult,
    };
  };
};

/**
 * Standard error message templates
 */
export const ErrorMessages = {
  create: (entityName: string) => `Failed to create ${entityName}`,
  update: (entityName: string) => `Failed to update ${entityName}`,
  delete: (entityName: string) => `Failed to delete ${entityName}`,
  network: "Network error. Please check your connection.",
  unauthorized: "You don't have permission to perform this action.",
  notFound: (entityName: string) => `${entityName} not found`,
  validation: "Please check your input and try again.",
};

/**
 * Standard success message templates
 */
export const SuccessMessages = {
  create: (entityName: string) => `${entityName} created successfully`,
  update: (entityName: string) => `${entityName} updated successfully`,
  delete: (entityName: string) => `${entityName} deleted successfully`,
  activate: (entityName: string) => `${entityName} activated`,
  pause: (entityName: string) => `${entityName} paused`,
  archive: (entityName: string) => `${entityName} archived`,
  duplicate: (entityName: string) => `${entityName} duplicated`,
};

