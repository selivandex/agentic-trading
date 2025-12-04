/** @format */

import { v7 as uuidv7 } from "uuid";

/**
 * Generate UUID v7 for nodes and edges
 *
 * UUID v7 features:
 * - Time-ordered (sortable by creation time)
 * - Monotonically increasing
 * - Better for database indexing than v4
 * - 128-bit unique identifier
 */
export const generateNodeId = (): string => {
  return uuidv7();
};

/**
 * Generate UUID v7 for edges
 *
 * Alias for generateNodeId() for semantic clarity
 */
export const generateEdgeId = (): string => {
  return uuidv7();
};

/**
 * Generate temporary ID for optimistic UI updates
 *
 * Temporary IDs are prefixed with "temp-" to distinguish them
 * from backend-generated UUIDs. They will be replaced with
 * real UUIDs after backend sync.
 */
export const generateTempId = (type: "node" | "edge"): string => {
  return `temp-${type}-${uuidv7()}`;
};

