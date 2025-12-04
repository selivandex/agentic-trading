/** @format */

import * as Sentry from "@sentry/nextjs";
import type {
  ErrorTracker,
  ErrorContext,
  ErrorLevel,
  UserContext,
  Breadcrumb,
} from "./error-tracker";

/**
 * Sentry Error Tracker
 *
 * Uses the globally initialized Sentry SDK (via instrumentation.ts)
 * to provide a type-safe, abstracted error tracking interface.
 *
 * Features:
 * - Error and exception tracking
 * - User context tracking
 * - Breadcrumbs (user actions trail)
 * - Custom tags and contexts
 * - GDPR compliant data sanitization
 *
 * Note: Sentry must be initialized via instrumentation.ts before using this tracker.
 *
 * @example
 * ```ts
 * const tracker = new SentryErrorTracker();
 * tracker.captureException(new Error('Something went wrong'), {
 *   tags: { feature: 'auth' },
 *   extra: { userId: '123' }
 * });
 * ```
 */
export class SentryErrorTracker implements ErrorTracker {
  private isAvailable = false;

  constructor() {
    this.checkAvailability();
  }

  /**
   * Check if Sentry is available and initialized
   */
  private checkAvailability(): void {
    try {
      // Check if Sentry is available (initialized via instrumentation.ts)
      this.isAvailable = typeof Sentry.getCurrentScope === "function";
    } catch {
      this.isAvailable = false;
    }
  }

  /**
   * Capture an exception
   */
  captureException(error: Error, context?: ErrorContext): void {
    if (!this.isAvailable) {
      return;
    }

    try {
      Sentry.captureException(error, {
        ...this.sanitizeContext(context),
      });
    } catch (err) {
      console.error("Failed to capture exception in Sentry:", err);
    }
  }

  /**
   * Capture a message
   */
  captureMessage(
    message: string,
    level: ErrorLevel,
    context?: ErrorContext
  ): void {
    if (!this.isAvailable) {
      return;
    }

    try {
      Sentry.captureMessage(message, {
        level: this.mapErrorLevel(level),
        ...this.sanitizeContext(context),
      });
    } catch (err) {
      console.error("Failed to capture message in Sentry:", err);
    }
  }

  /**
   * Set user context
   */
  setUser(user: UserContext | null): void {
    if (!this.isAvailable) {
      return;
    }

    try {
      Sentry.setUser(user ? this.sanitizeUser(user) : null);
    } catch (err) {
      console.error("Failed to set user in Sentry:", err);
    }
  }

  /**
   * Add a breadcrumb
   */
  addBreadcrumb(breadcrumb: Breadcrumb): void {
    if (!this.isAvailable) {
      return;
    }

    try {
      Sentry.addBreadcrumb({
        ...breadcrumb,
        timestamp: breadcrumb.timestamp || Date.now() / 1000, // Sentry uses seconds
      });
    } catch (err) {
      console.error("Failed to add breadcrumb in Sentry:", err);
    }
  }

  /**
   * Set custom context
   */
  setContext(name: string, context: Record<string, unknown> | null): void {
    if (!this.isAvailable) {
      return;
    }

    try {
      Sentry.setContext(name, context);
    } catch (err) {
      console.error("Failed to set context in Sentry:", err);
    }
  }

  /**
   * Set a tag
   */
  setTag(key: string, value: string): void {
    if (!this.isAvailable) {
      return;
    }

    try {
      Sentry.setTag(key, value);
    } catch (err) {
      console.error("Failed to set tag in Sentry:", err);
    }
  }

  /**
   * Map our ErrorLevel to Sentry severity level
   */
  private mapErrorLevel(
    level: ErrorLevel
  ): "fatal" | "error" | "warning" | "info" | "debug" | "log" {
    switch (level) {
      case "fatal":
        return "fatal";
      case "error":
        return "error";
      case "warning":
        return "warning";
      case "info":
        return "info";
      case "debug":
        return "debug";
      default:
        return "error";
    }
  }

  /**
   * Sanitize context for GDPR compliance
   *
   * Removes sensitive data before sending to Sentry.
   */
  private sanitizeContext(
    context?: ErrorContext
  ): Partial<ErrorContext> | undefined {
    if (!context) {
      return undefined;
    }

    const sanitized: Partial<ErrorContext> = {};

    if (context.tags) {
      sanitized.tags = this.sanitizeTags(context.tags);
    }

    if (context.extra) {
      sanitized.extra = this.sanitizeExtra(context.extra);
    }

    if (context.contexts) {
      sanitized.contexts = context.contexts;
    }

    return sanitized;
  }

  /**
   * Sanitize user data for GDPR compliance
   */
  private sanitizeUser(user: UserContext): UserContext {
    const sanitized: UserContext = {
      id: user.id,
    };

    // Only include email/username if explicitly provided
    if (user.email) {
      sanitized.email = user.email;
    }

    if (user.username) {
      sanitized.username = user.username;
    }

    // Include other non-sensitive fields
    Object.keys(user).forEach((key) => {
      if (
        key !== "id" &&
        key !== "email" &&
        key !== "username" &&
        !this.isSensitiveField(key)
      ) {
        sanitized[key] = user[key];
      }
    });

    return sanitized;
  }

  /**
   * Sanitize tags
   */
  private sanitizeTags(tags: Record<string, string>): Record<string, string> {
    const sanitized: Record<string, string> = {};

    Object.entries(tags).forEach(([key, value]) => {
      if (!this.isSensitiveField(key)) {
        sanitized[key] = value;
      }
    });

    return sanitized;
  }

  /**
   * Sanitize extra data
   */
  private sanitizeExtra(extra: Record<string, unknown>): Record<string, unknown> {
    const sanitized: Record<string, unknown> = {};

    Object.entries(extra).forEach(([key, value]) => {
      if (!this.isSensitiveField(key)) {
        sanitized[key] = this.sanitizeValue(value);
      }
    });

    return sanitized;
  }

  /**
   * Recursively sanitize values
   */
  private sanitizeValue(value: unknown): unknown {
    if (value === null || value === undefined) {
      return value;
    }

    if (typeof value === "object") {
      if (Array.isArray(value)) {
        return value.map((item) => this.sanitizeValue(item));
      }

      const sanitized: Record<string, unknown> = {};
      Object.entries(value as Record<string, unknown>).forEach(([key, val]) => {
        if (!this.isSensitiveField(key)) {
          sanitized[key] = this.sanitizeValue(val);
        }
      });
      return sanitized;
    }

    return value;
  }

  /**
   * Check if field name is sensitive
   */
  private isSensitiveField(fieldName: string): boolean {
    const sensitivePatterns = [
      /password/i,
      /secret/i,
      /token/i,
      /api[_-]?key/i,
      /auth/i,
      /credit[_-]?card/i,
      /ssn/i,
      /social[_-]?security/i,
    ];

    return sensitivePatterns.some((pattern) => pattern.test(fieldName));
  }
}
