/** @format */

import { logger } from "@/shared/lib/logger";
import type {
  ErrorTracker,
  ErrorContext,
  ErrorLevel,
  UserContext,
  Breadcrumb,
} from "./error-tracker";

/**
 * Console Error Tracker
 *
 * Simple error tracker that logs to console.
 * Used as fallback when Sentry is not configured or in development.
 */
export class ConsoleErrorTracker implements ErrorTracker {
  private userContext: UserContext | null = null;
  private breadcrumbs: Breadcrumb[] = [];
  private contexts: Map<string, Record<string, unknown>> = new Map();
  private tags: Map<string, string> = new Map();

  /**
   * Capture an exception
   */
  captureException(error: Error, context?: ErrorContext): void {
    const logData = this.buildLogData(context);

    logger.error("Error captured:", {
      error: {
        name: error.name,
        message: error.message,
        stack: error.stack,
      },
      ...logData,
    });

    // Log breadcrumbs if any
    if (this.breadcrumbs.length > 0) {
      logger.debug("Breadcrumbs:", this.breadcrumbs);
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
    const logData = this.buildLogData(context);

    switch (level) {
      case "fatal":
      case "error":
        logger.error(message, logData);
        break;
      case "warning":
        logger.warn(message, logData);
        break;
      case "info":
        logger.info(message, logData);
        break;
      case "debug":
        logger.debug(message, logData);
        break;
    }
  }

  /**
   * Set user context
   */
  setUser(user: UserContext | null): void {
    this.userContext = user;
    if (user) {
      logger.debug("User context set:", user);
    } else {
      logger.debug("User context cleared");
    }
  }

  /**
   * Add a breadcrumb
   */
  addBreadcrumb(breadcrumb: Breadcrumb): void {
    const crumb = {
      ...breadcrumb,
      timestamp: breadcrumb.timestamp || Date.now(),
    };

    // Keep last 100 breadcrumbs
    this.breadcrumbs.push(crumb);
    if (this.breadcrumbs.length > 100) {
      this.breadcrumbs.shift();
    }

    logger.debug("Breadcrumb added:", crumb);
  }

  /**
   * Set custom context
   */
  setContext(name: string, context: Record<string, unknown> | null): void {
    if (context === null) {
      this.contexts.delete(name);
      logger.debug(`Context "${name}" cleared`);
    } else {
      this.contexts.set(name, context);
      logger.debug(`Context "${name}" set:`, context);
    }
  }

  /**
   * Set a tag
   */
  setTag(key: string, value: string): void {
    this.tags.set(key, value);
    logger.debug(`Tag "${key}" set:`, value);
  }

  /**
   * Build log data from context and stored state
   */
  private buildLogData(context?: ErrorContext): Record<string, unknown> {
    const data: Record<string, unknown> = {};

    // Add user context
    if (this.userContext) {
      data.user = this.userContext;
    }

    // Add tags
    if (this.tags.size > 0 || context?.tags) {
      data.tags = {
        ...Object.fromEntries(this.tags),
        ...context?.tags,
      };
    }

    // Add contexts
    if (this.contexts.size > 0 || context?.contexts) {
      data.contexts = {
        ...Object.fromEntries(this.contexts),
        ...context?.contexts,
      };
    }

    // Add extra data
    if (context?.extra) {
      data.extra = context.extra;
    }

    return data;
  }
}
