/** @format */

"use client";

import { UserManager } from "@/entities/user";

/**
 * Users Management Page
 *
 * Admin interface for managing system users:
 * - List all users with search and filters
 * - View user details and settings
 * - Edit user settings (risk parameters, limits, etc.)
 * - Activate/deactivate users
 * - Batch operations (activate/deactivate multiple users)
 *
 * Note: Users cannot be created or deleted through this interface.
 * - Users are created via Telegram/OAuth registration
 * - Users can only be deactivated, not deleted (data retention)
 */
export default function UsersPage() {
  return <UserManager />;
}
