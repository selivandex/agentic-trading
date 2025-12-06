/** @format */

"use client";

import { Crud } from "@/shared/ui/crud";
import { strategyCrudConfig } from "@/entities/strategy";

/**
 * Strategies Management Page
 *
 * Full CRUD interface for managing trading strategies:
 * - List view with search, sort, pagination
 * - Create new strategy
 * - Edit existing strategy
 * - View strategy details
 * - Delete strategy
 */
export default function StrategiesPage() {
  return (
    <div className="container mx-auto py-8">
      <Crud config={strategyCrudConfig} />
    </div>
  );
}
