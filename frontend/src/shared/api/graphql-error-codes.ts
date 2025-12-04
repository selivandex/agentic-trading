/**
 * GraphQL Error Codes
 *
 * Centralized enum for all error codes from backend GraphQL API.
 * These codes come from extensions.code field in GraphQL errors.
 *
 * @format
 */

/**
 * Standard GraphQL error codes from backend
 */
export enum GraphQLErrorCode {
  // Authentication & Authorization
  UNAUTHENTICATED = "UNAUTHENTICATED", // Rails GraphQL standard code for 401
  UNAUTHORIZED = "UNAUTHORIZED", // Alternative code for 401
  FORBIDDEN = "FORBIDDEN",

  // Context Errors (Organization/Project)
  ORGANIZATION_NOT_FOUND = "ORGANIZATION_NOT_FOUND",
  PROJECT_NOT_FOUND = "PROJECT_NOT_FOUND",
  ORGANIZATION_REQUIRED = "ORGANIZATION_REQUIRED",
  PROJECT_REQUIRED = "PROJECT_REQUIRED",

  // Resource Not Found
  NOT_FOUND = "NOT_FOUND",
  SCENARIO_NOT_FOUND = "SCENARIO_NOT_FOUND",
  SCREEN_NOT_FOUND = "SCREEN_NOT_FOUND",
  LANGUAGE_NOT_FOUND = "LANGUAGE_NOT_FOUND",
  MACRO_NOT_FOUND = "MACRO_NOT_FOUND",
  AUDIENCE_NOT_FOUND = "AUDIENCE_NOT_FOUND",

  // Validation Errors
  VALIDATION_ERROR = "VALIDATION_ERROR",
  INVALID_INPUT = "INVALID_INPUT",

  // Business Logic Errors
  DUPLICATE_RECORD = "DUPLICATE_RECORD",
  OPTIMISTIC_LOCK_ERROR = "OPTIMISTIC_LOCK_ERROR",

  // Server Errors
  INTERNAL_SERVER_ERROR = "INTERNAL_SERVER_ERROR",
  GRAPHQL_PARSE_FAILED = "GRAPHQL_PARSE_FAILED",
  GRAPHQL_VALIDATION_FAILED = "GRAPHQL_VALIDATION_FAILED",
}

/**
 * Error codes that should trigger redirect to error page
 */
export const CONTEXT_ERROR_CODES = [
  GraphQLErrorCode.ORGANIZATION_NOT_FOUND,
  GraphQLErrorCode.PROJECT_NOT_FOUND,
  GraphQLErrorCode.ORGANIZATION_REQUIRED,
  GraphQLErrorCode.PROJECT_REQUIRED,
] as const;

/**
 * Error codes that should trigger redirect to login
 */
export const AUTH_ERROR_CODES = [
  GraphQLErrorCode.UNAUTHENTICATED,
  GraphQLErrorCode.UNAUTHORIZED,
] as const;

/**
 * Error codes that should trigger redirect to forbidden page
 */
export const FORBIDDEN_ERROR_CODES = [GraphQLErrorCode.FORBIDDEN] as const;

/**
 * Check if error code requires context error page
 */
export const isContextError = (code: string): boolean => {
  return (CONTEXT_ERROR_CODES as readonly string[]).includes(code);
};

/**
 * Check if error code requires authentication
 */
export const isAuthError = (code: string): boolean => {
  return (AUTH_ERROR_CODES as readonly string[]).includes(code);
};

/**
 * Check if error code requires forbidden page
 */
export const isForbiddenError = (code: string): boolean => {
  return (FORBIDDEN_ERROR_CODES as readonly string[]).includes(code);
};
