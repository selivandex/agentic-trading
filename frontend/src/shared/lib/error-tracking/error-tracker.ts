/** @format */

/**
 * Error severity levels
 */
export enum ErrorLevel {
  Fatal = "fatal",
  Error = "error",
  Warning = "warning",
  Info = "info",
  Debug = "debug",
}

/**
 * User context for error tracking
 */
export interface UserContext {
  id: string;
  email?: string;
  username?: string;
  [key: string]: string | number | boolean | undefined;
}

/**
 * Additional context for errors
 */
export interface ErrorContext {
  /** Tags for categorization */
  tags?: Record<string, string>;
  /** Additional custom data */
  extra?: Record<string, unknown>;
  /** Context name (e.g., "GraphQL", "Network", "Validation") */
  contexts?: Record<string, Record<string, unknown>>;
}

/**
 * Breadcrumb for tracking user actions
 */
export interface Breadcrumb {
  /** Breadcrumb type (e.g., "navigation", "user", "http") */
  type?: string;
  /** Breadcrumb category (e.g., "ui.click", "console") */
  category?: string;
  /** Breadcrumb message */
  message: string;
  /** Breadcrumb data */
  data?: Record<string, unknown>;
  /** Breadcrumb level */
  level?: ErrorLevel;
  /** Timestamp (auto-generated if not provided) */
  timestamp?: number;
}

/**
 * Abstract Error Tracker Interface
 *
 * Provides a unified interface for error tracking across different implementations.
 * Implementations can be Sentry, console logging, or custom solutions.
 */
export interface ErrorTracker {
  /**
   * Capture an exception/error
   *
   * @param error - Error object to capture
   * @param context - Additional context for the error
   */
  captureException(error: Error, context?: ErrorContext): void;

  /**
   * Capture a message
   *
   * @param message - Message to capture
   * @param level - Message severity level
   * @param context - Additional context for the message
   */
  captureMessage(
    message: string,
    level: ErrorLevel,
    context?: ErrorContext
  ): void;

  /**
   * Set user context
   *
   * Associates user information with subsequent errors.
   * Pass null to clear user context.
   *
   * @param user - User context or null to clear
   */
  setUser(user: UserContext | null): void;

  /**
   * Add a breadcrumb
   *
   * Breadcrumbs are used to track user actions leading up to an error.
   *
   * @param breadcrumb - Breadcrumb to add
   */
  addBreadcrumb(breadcrumb: Breadcrumb): void;

  /**
   * Set custom context
   *
   * Adds custom context that will be attached to errors.
   *
   * @param name - Context name
   * @param context - Context data
   */
  setContext(name: string, context: Record<string, unknown> | null): void;

  /**
   * Set a tag
   *
   * Tags are key-value pairs used for filtering and searching errors.
   *
   * @param key - Tag key
   * @param value - Tag value
   */
  setTag(key: string, value: string): void;
}

/**
 * No-op Error Tracker
 *
 * Disabled error tracker that does nothing.
 * Used when error tracking is disabled.
 */
export class NoopErrorTracker implements ErrorTracker {
  captureException(_error: Error, _context?: ErrorContext): void {
    // No-op
  }

  captureMessage(
    _message: string,
    _level: ErrorLevel,
    _context?: ErrorContext
  ): void {
    // No-op
  }

  setUser(_user: UserContext | null): void {
    // No-op
  }

  addBreadcrumb(_breadcrumb: Breadcrumb): void {
    // No-op
  }

  setContext(_name: string, _context: Record<string, unknown> | null): void {
    // No-op
  }

  setTag(_key: string, _value: string): void {
    // No-op
  }
}
