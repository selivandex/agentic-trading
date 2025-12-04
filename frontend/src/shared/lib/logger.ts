/** @format */

import { createConsola } from "consola";

/**
 * Application Logger
 *
 * Centralized logging utility based on consola library.
 * Provides enhanced logging with colors, tags, and environment-based filtering.
 *
 * Features:
 * - Multiple log levels (debug, info, success, warn, error)
 * - Colored console output with emojis
 * - Auto-disabled in production (can be configured)
 * - Tagged messages for easy filtering
 * - Support for grouped logs
 *
 * @example
 * ```typescript
 * import { logger } from "@/shared/lib";
 *
 * logger.debug("Node created", { nodeId: "123" });
 * logger.info("Flow loaded");
 * logger.success("Changes saved!");
 * logger.warn("Validation warning", warnings);
 * logger.error("Sync failed", error);
 *
 * logger.group(() => {
 *   logger.info("Validation Results:");
 *   logger.debug("Errors:", errors);
 *   logger.debug("Warnings:", warnings);
 * });
 * ```
 */

// Create consola instance with app-specific configuration
const consolaInstance = createConsola({
  defaults: {
    tag: "Prometheus",
  },
  // Levels: 0 = silent, 1 = fatal, 2 = error, 3 = warn, 4 = log/info, 5 = debug/verbose
  level: process.env.NODE_ENV === "development" ? 5 : 3,
});

/**
 * Grouped log execution
 *
 * Executes function and wraps output in a collapsible group.
 * Note: consola doesn't have native group support, so we use visual separators.
 *
 * @param title - Group title
 * @param collapsed - Whether to start collapsed (not supported in consola)
 * @param fn - Function to execute within the group
 *
 * @example
 * ```typescript
 * logger.group("Validation", false, () => {
 *   logger.info("Errors:", errors);
 *   logger.info("Warnings:", warnings);
 * });
 * ```
 */
const group = (title: string, _collapsed: boolean = false, fn?: () => void): void => {
  consolaInstance.log(`\n━━━ ${title} ━━━`);
  if (fn) {
    fn();
    consolaInstance.log(`━━━━━━━━━━━━━━━━\n`);
  }
};

/**
 * End current log group (for compatibility)
 */
const groupEnd = (): void => {
  consolaInstance.log(`━━━━━━━━━━━━━━━━\n`);
};

/**
 * Time measurement
 *
 * @example
 * ```typescript
 * logger.time("sync-operation");
 * // ... do work ...
 * logger.timeEnd("sync-operation");
 * ```
 */
const timeTrackers = new Map<string, number>();

const time = (label: string): void => {
  timeTrackers.set(label, Date.now());
  consolaInstance.debug(`⏱️  Started: ${label}`);
};

const timeEnd = (label: string): void => {
  const start = timeTrackers.get(label);
  if (start) {
    const duration = Date.now() - start;
    consolaInstance.success(`⏱️  ${label}: ${duration}ms`);
    timeTrackers.delete(label);
  } else {
    consolaInstance.warn(`⏱️  Timer "${label}" was not started`);
  }
};

/**
 * Application logger with enhanced methods
 */
export const logger = {
  // Consola methods (native)
  debug: consolaInstance.debug.bind(consolaInstance),
  info: consolaInstance.info.bind(consolaInstance),
  log: consolaInstance.log.bind(consolaInstance),
  warn: consolaInstance.warn.bind(consolaInstance),
  error: consolaInstance.error.bind(consolaInstance),
  success: consolaInstance.success.bind(consolaInstance),
  fatal: consolaInstance.fatal.bind(consolaInstance),
  trace: consolaInstance.trace.bind(consolaInstance),

  // Custom methods
  group,
  groupEnd,
  time,
  timeEnd,

  // Direct access to consola instance for advanced usage
  consola: consolaInstance,
};

/**
 * Export default for compatibility
 */
export default logger;
