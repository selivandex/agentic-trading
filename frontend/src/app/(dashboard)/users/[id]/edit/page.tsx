/** @format */

"use client";

import { Crud } from "@/shared/ui/crud";
import { userCrudConfig } from "@/entities/user";
import { use } from "react";

/**
 * Edit User Settings Page
 *
 * Allows editing of user settings including:
 * - Risk tolerance and limits
 * - Position size constraints
 * - Circuit breaker settings
 * - AI provider preferences
 * - Notification settings
 * - Timezone and reporting preferences
 *
 * Note: User profile information (name, email, telegram) cannot be edited.
 * These are managed through the authentication provider.
 */
export default function EditUserPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);

  return (
    <div className="h-full">
      <Crud config={userCrudConfig} mode="edit" entityId={id} />
    </div>
  );
}

