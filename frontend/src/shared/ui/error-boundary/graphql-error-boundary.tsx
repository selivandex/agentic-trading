/**
 * GraphQL Error Boundary
 *
 * Catches GraphQL and network errors and displays user-friendly error UI.
 * Use this to wrap components that make GraphQL queries/mutations.
 *
 * @format
 */

"use client";

import { Component, type ReactNode } from "react";
import { ApolloError } from "@apollo/client";

interface GraphQLErrorBoundaryProps {
  children: ReactNode;
  fallback?: (error: Error, resetError: () => void) => ReactNode;
  onError?: (error: Error) => void;
}

interface GraphQLErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

/**
 * Error Boundary for GraphQL operations
 *
 * Catches errors from Apollo Client queries/mutations and displays fallback UI.
 *
 * @example
 * ```tsx
 * <GraphQLErrorBoundary>
 *   <ScenariosList />
 * </GraphQLErrorBoundary>
 * ```
 *
 * @example Custom fallback
 * ```tsx
 * <GraphQLErrorBoundary
 *   fallback={(error, resetError) => (
 *     <div>
 *       <p>Failed to load: {error.message}</p>
 *       <button onClick={resetError}>Retry</button>
 *     </div>
 *   )}
 * >
 *   <ScenariosList />
 * </GraphQLErrorBoundary>
 * ```
 */
export class GraphQLErrorBoundary extends Component<
  GraphQLErrorBoundaryProps,
  GraphQLErrorBoundaryState
> {
  constructor(props: GraphQLErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): GraphQLErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error) {
    // Log error to console in development
    if (process.env.NODE_ENV === "development") {
      console.error("[GraphQLErrorBoundary] Caught error:", error);
    }

    // Call onError callback if provided
    if (this.props.onError) {
      this.props.onError(error);
    }

    // TODO: Send to error tracking service (Sentry, etc.)
    // if (process.env.NODE_ENV === "production") {
    //   trackError(error);
    // }
  }

  resetError = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError && this.state.error) {
      // Use custom fallback if provided
      if (this.props.fallback) {
        return this.props.fallback(this.state.error, this.resetError);
      }

      // Default fallback UI
      return (
        <DefaultErrorFallback
          error={this.state.error}
          resetError={this.resetError}
        />
      );
    }

    return this.props.children;
  }
}

/**
 * Default error fallback component
 */
function DefaultErrorFallback({
  error,
  resetError,
}: {
  error: Error;
  resetError: () => void;
}) {
  // Check if it's an Apollo error
  const isApolloError = error instanceof ApolloError;
  const apolloError = isApolloError ? (error as ApolloError) : null;

  // Extract error message
  const errorMessage = getErrorMessage(error);
  const errorType = getErrorType(error);

  return (
    <div className="flex min-h-[400px] flex-col items-center justify-center rounded-lg border border-error-300 bg-error-25 p-8">
      <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-error-100">
        <svg
          className="h-6 w-6 text-error-600"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
          />
        </svg>
      </div>

      <h3 className="mb-2 text-lg font-semibold text-gray-900">{errorType}</h3>

      <p className="mb-6 max-w-md text-center text-sm text-gray-600">
        {errorMessage}
      </p>

      {/* GraphQL errors details (development only) */}
      {process.env.NODE_ENV === "development" && apolloError?.graphQLErrors && (
        <div className="mb-4 w-full max-w-2xl rounded-lg bg-gray-50 p-4">
          <p className="mb-2 text-xs font-semibold text-gray-700">
            GraphQL Errors (dev only):
          </p>
          <ul className="space-y-2">
            {apolloError.graphQLErrors.map((err, index) => (
              <li key={index} className="text-xs text-gray-600">
                <span className="font-medium">
                  {(err.extensions?.code as string) || "GRAPHQL_ERROR"}:
                </span>{" "}
                {err.message}
                {err.path && (
                  <span className="ml-2 text-gray-500">
                    at {err.path.join(".")}
                  </span>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Network error details (development only) */}
      {process.env.NODE_ENV === "development" && apolloError?.networkError && (
        <div className="mb-4 w-full max-w-2xl rounded-lg bg-gray-50 p-4">
          <p className="mb-2 text-xs font-semibold text-gray-700">
            Network Error (dev only):
          </p>
          <p className="text-xs text-gray-600">
            {apolloError.networkError.message}
          </p>
        </div>
      )}

      <div className="flex gap-3">
        <button
          onClick={resetError}
          className="rounded-lg bg-error-600 px-4 py-2 text-sm font-semibold text-white hover:bg-error-700 focus:outline-none focus:ring-2 focus:ring-error-500 focus:ring-offset-2"
        >
          Try again
        </button>

        <button
          onClick={() => window.location.reload()}
          className="rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-semibold text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2"
        >
          Reload page
        </button>
      </div>
    </div>
  );
}

/**
 * Extract user-friendly error message
 */
function getErrorMessage(error: Error): string {
  if (error instanceof ApolloError) {
    // GraphQL errors
    if (error.graphQLErrors.length > 0) {
      return error.graphQLErrors[0].message;
    }

    // Network errors
    if (error.networkError) {
      return "Unable to connect to the server. Please check your internet connection.";
    }
  }

  // Generic error
  return error.message || "An unexpected error occurred. Please try again.";
}

/**
 * Get error type for display
 */
function getErrorType(error: Error): string {
  if (error instanceof ApolloError) {
    if (error.graphQLErrors.length > 0) {
      const code = error.graphQLErrors[0].extensions?.code as string;

      switch (code) {
        case "UNAUTHORIZED":
        case "UNAUTHENTICATED":
          return "Authentication Error";
        case "FORBIDDEN":
          return "Access Denied";
        case "NOT_FOUND":
          return "Not Found";
        case "BAD_USER_INPUT":
        case "VALIDATION_ERROR":
          return "Invalid Input";
        default:
          return "GraphQL Error";
      }
    }

    if (error.networkError) {
      return "Network Error";
    }
  }

  return "Error";
}
