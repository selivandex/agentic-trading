/** @format */

"use client";

import { Crud } from "@/shared/ui/crud";
import { userCrudConfig } from "@/entities/user";
import { use } from "react";

/**
 * User Detail Page
 *
 * Displays detailed information about a specific user including:
 * - Profile information (name, email, telegram)
 * - Account status and premium tier
 * - Risk settings and limits
 * - AI preferences
 * - Trading preferences
 */
export default function UserDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);

  return (
    <div className="h-full">
      <Crud config={userCrudConfig} mode="show" entityId={id} />
    </div>
  );
}

