/** @format */

"use client";

import { StrategyManager } from "@/entities/strategy";

/**
 * Strategies Management Page
 *
 * Full CRUD interface for managing trading strategies:
 * - List view with search, sort, pagination
 * - Create new strategy
 * - Edit existing strategy
 * - View strategy details
 * - Delete strategy
 * - Pause/Resume/Close actions
 */
export default function StrategiesPage() {
  return <StrategyManager />;
}
