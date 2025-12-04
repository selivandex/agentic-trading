/**
 * Apollo Performance Monitoring Link
 *
 * Tracks GraphQL query/mutation performance and logs slow operations.
 * Can be integrated with external monitoring services.
 *
 * @format
 */

import { ApolloLink } from "@apollo/client";
import type { Operation, FetchResult, NextLink } from "@apollo/client";
import { Observable } from "@apollo/client/utilities";

// Threshold for slow operations (milliseconds)
const SLOW_OPERATION_THRESHOLD = 1000;

// Development mode - more verbose logging
const IS_DEVELOPMENT = process.env.NODE_ENV === "development";

// Enable debug logging with NEXT_PUBLIC_APOLLO_DEBUG=true
const APOLLO_DEBUG_ENABLED = process.env.NEXT_PUBLIC_APOLLO_DEBUG === "true";

/**
 * Performance metrics for an operation
 */
interface OperationMetrics {
  operationName: string | undefined;
  operationType: string;
  startTime: number;
  endTime: number;
  duration: number;
  variables: Record<string, unknown>;
  slowOperation: boolean;
}

/**
 * Log performance metrics
 * In production, this could send to an external monitoring service
 */
function logMetrics(metrics: OperationMetrics) {
  // Always log slow operations
  if (metrics.slowOperation) {
    console.warn(
      `🐌 Slow GraphQL ${metrics.operationType}: ${metrics.operationName}`,
      {
        duration: `${metrics.duration}ms`,
        variables: metrics.variables,
        threshold: `${SLOW_OPERATION_THRESHOLD}ms`,
      }
    );
  }

  // In development with debug flag, log all operations
  if (IS_DEVELOPMENT && APOLLO_DEBUG_ENABLED) {
    const emoji = metrics.operationType === "query" ? "🔍" : "✏️";
    const color = metrics.slowOperation ? "color: red" : "color: green";

    console.log(
      `%c${emoji} GraphQL ${metrics.operationType}: ${metrics.operationName || "unnamed"} (${metrics.duration}ms)`,
      color,
      {
        variables: metrics.variables,
        startTime: new Date(metrics.startTime).toISOString(),
      }
    );
  }

  // In production, send to monitoring service
  if (!IS_DEVELOPMENT && typeof window !== "undefined") {
    // Example: Send to analytics service
    // analytics.track('graphql_operation', {
    //   operation_name: metrics.operationName,
    //   operation_type: metrics.operationType,
    //   duration_ms: metrics.duration,
    //   is_slow: metrics.slowOperation,
    // });
  }
}

/**
 * Track cache hits/misses for queries
 */
function trackCacheMetrics(operation: Operation) {
  if (operation.getContext().fetchPolicy === "cache-only") {
    if (IS_DEVELOPMENT) {
      console.log(
        `💾 Cache-only query: ${operation.operationName}`,
        { variables: operation.variables }
      );
    }
  }
}

/**
 * Performance monitoring link for Apollo Client
 * Tracks execution time of all GraphQL operations
 */
export const performanceLink = new ApolloLink((operation: Operation, forward: NextLink): Observable<FetchResult> => {
  const startTime = performance.now();

  // Determine operation type
  const operationType = operation.query.definitions.some(
    (def: { kind: string; operation?: string }) => def.operation === "mutation"
  )
    ? "mutation"
    : operation.query.definitions.some(
        (def: { kind: string; operation?: string }) => def.operation === "subscription"
      )
    ? "subscription"
    : "query";

  // Track cache metrics for queries
  if (operationType === "query") {
    trackCacheMetrics(operation);
  }

  return new Observable<FetchResult>((observer) => {
    const subscription = forward(operation).subscribe({
      next: (result) => {
        const endTime = performance.now();
        const duration = Math.round(endTime - startTime);

        // Create metrics object
        const metrics: OperationMetrics = {
          operationName: operation.operationName,
          operationType,
          startTime,
          endTime,
          duration,
          variables: operation.variables || {},
          slowOperation: duration > SLOW_OPERATION_THRESHOLD,
        };

        // Log metrics
        logMetrics(metrics);

        // Check for GraphQL errors in result
        if (result.errors && result.errors.length > 0) {
          console.error(
            `❌ GraphQL errors in ${operation.operationName}:`,
            result.errors
          );
        }

        observer.next(result);
      },
      error: (error) => {
        const endTime = performance.now();
        const duration = Math.round(endTime - startTime);

        console.error(
          `❌ GraphQL ${operationType} failed: ${operation.operationName}`,
          {
            duration: `${duration}ms`,
            error,
            variables: operation.variables,
          }
        );

        observer.error(error);
      },
      complete: () => {
        observer.complete();
      },
    });

    return () => {
      subscription.unsubscribe();
    };
  });
});

/**
 * Helper to batch log operations (useful for reducing console spam)
 */
class BatchedOperationLogger {
  private operations: OperationMetrics[] = [];
  private timer: NodeJS.Timeout | null = null;

  add(metrics: OperationMetrics) {
    this.operations.push(metrics);

    if (!this.timer) {
      this.timer = setTimeout(() => {
        this.flush();
      }, 1000); // Batch every second
    }
  }

  flush() {
    if (this.operations.length === 0) return;

    const totalDuration = this.operations.reduce((sum, op) => sum + op.duration, 0);
    const avgDuration = Math.round(totalDuration / this.operations.length);
    const slowOps = this.operations.filter(op => op.slowOperation);

    console.group(`📊 GraphQL Performance Summary (${this.operations.length} operations)`);
    console.log(`Average duration: ${avgDuration}ms`);
    console.log(`Total duration: ${totalDuration}ms`);

    if (slowOps.length > 0) {
      console.warn(`Slow operations (>${SLOW_OPERATION_THRESHOLD}ms):`, slowOps);
    }

    console.table(
      this.operations.map(op => ({
        Operation: op.operationName || "unnamed",
        Type: op.operationType,
        Duration: `${op.duration}ms`,
        Slow: op.slowOperation ? "⚠️" : "✅",
      }))
    );
    console.groupEnd();

    this.operations = [];
    this.timer = null;
  }
}

// Export batched logger for use in development
export const batchedLogger = new BatchedOperationLogger();
