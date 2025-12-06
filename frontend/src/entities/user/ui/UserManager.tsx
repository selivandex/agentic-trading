/** @format */

"use client";

import { useMemo } from "react";
import { toast } from "sonner";
import { CheckCircle, XCircle } from "@untitledui/icons";
import { Crud } from "@/shared/ui/crud";
import { userCrudConfig } from "../lib/crud-config";
import { useMutation } from "@apollo/client";
import { SET_USER_ACTIVE_MUTATION, GET_USERS_QUERY } from "../api/user.graphql";
import type { User } from "../model/types";

/**
 * UserManager Component
 *
 * Enhanced Crud wrapper for Users with real mutation implementations
 * for activate/deactivate actions.
 *
 * Features:
 * - View all users with search and filters
 * - Edit user settings (risk limits, AI preferences, etc.)
 * - Activate/deactivate user accounts
 * - Batch operations for multiple users
 *
 * Note: Users cannot be created or deleted through this interface.
 * - Users are created via Telegram/OAuth registration
 * - Users can only be deactivated (soft delete) for data retention
 */
export function UserManager() {
  const [setUserActive, { loading: setActiveLoading }] = useMutation(
    SET_USER_ACTIVE_MUTATION,
    {
      refetchQueries: [GET_USERS_QUERY],
    }
  );

  // Enhanced config with real mutation implementations
  const enhancedConfig = useMemo(() => {
    const activateAction = {
      key: "activate",
      label: "Activate",
      icon: CheckCircle,
      onClick: async (user: User) => {
        try {
          await setUserActive({
            variables: {
              userID: user.id,
              isActive: true,
            },
          });
          toast.success("User activated successfully");
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to activate user";
          toast.error(message);
        }
      },
      disabled: () => setActiveLoading,
      hidden: (user: User) => user.isActive,
    };

    const deactivateAction = {
      key: "deactivate",
      label: "Deactivate",
      icon: XCircle,
      onClick: async (user: User) => {
        if (
          !confirm(
            `Are you sure you want to deactivate ${user.firstName} ${user.lastName}? They will lose access to the platform.`
          )
        ) {
          return;
        }

        try {
          await setUserActive({
            variables: {
              userID: user.id,
              isActive: false,
            },
          });
          toast.success("User deactivated successfully");
        } catch (error) {
          const message =
            error instanceof Error
              ? error.message
              : "Failed to deactivate user";
          toast.error(message);
        }
      },
      disabled: () => setActiveLoading,
      hidden: (user: User) => !user.isActive,
      destructive: true,
    };

    // Batch operations
    const batchActivateAction = {
      key: "activate-all",
      label: "Activate Selected",
      icon: CheckCircle,
      onClick: async (user: User) => {
        try {
          await setUserActive({
            variables: {
              userID: user.id,
              isActive: true,
            },
          });
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to activate";
          throw new Error(message);
        }
      },
      onBatchClick: async (users: User[]) => {
        const inactiveUsers = users.filter((u) => !u.isActive);

        if (inactiveUsers.length === 0) {
          toast.info("All selected users are already active");
          return;
        }

        if (inactiveUsers.length < users.length) {
          toast.info(
            `${inactiveUsers.length} of ${users.length} selected users will be activated`
          );
        }

        try {
          // Execute all mutations in parallel
          await Promise.all(
            inactiveUsers.map((user) =>
              setUserActive({
                variables: {
                  userID: user.id,
                  isActive: true,
                },
              })
            )
          );
          toast.success(`Activated ${inactiveUsers.length} users`);
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to activate users";
          toast.error(message);
          throw error;
        }
      },
      disabled: () => setActiveLoading,
      hiddenForBatch: (users: User[]) => !users.some((u) => !u.isActive),
    };

    const batchDeactivateAction = {
      key: "deactivate-all",
      label: "Deactivate Selected",
      icon: XCircle,
      onClick: async (user: User) => {
        try {
          await setUserActive({
            variables: {
              userID: user.id,
              isActive: false,
            },
          });
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "Failed to deactivate";
          throw new Error(message);
        }
      },
      onBatchClick: async (users: User[]) => {
        const activeUsers = users.filter((u) => u.isActive);

        if (activeUsers.length === 0) {
          toast.info("All selected users are already inactive");
          return;
        }

        if (activeUsers.length < users.length) {
          toast.info(
            `${activeUsers.length} of ${users.length} selected users will be deactivated`
          );
        }

        if (
          !confirm(
            `Are you sure you want to deactivate ${activeUsers.length} users? They will lose access to the platform.`
          )
        ) {
          return;
        }

        try {
          // Execute all mutations in parallel
          await Promise.all(
            activeUsers.map((user) =>
              setUserActive({
                variables: {
                  userID: user.id,
                  isActive: false,
                },
              })
            )
          );
          toast.success(`Deactivated ${activeUsers.length} users`);
        } catch (error) {
          const message =
            error instanceof Error
              ? error.message
              : "Failed to deactivate users";
          toast.error(message);
          throw error;
        }
      },
      disabled: () => setActiveLoading,
      destructive: true,
      hiddenForBatch: (users: User[]) => !users.some((u) => u.isActive),
    };

    return {
      ...userCrudConfig,
      // Actions for table rows (dropdown menu)
      actions: [activateAction, deactivateAction],
      // Actions for show page (header buttons)
      showActions: [activateAction, deactivateAction],
      // Batch actions for selected rows
      bulkActions: [batchActivateAction, batchDeactivateAction],
      // Enable selection for batch operations
      enableSelection: true,
    };
  }, [setUserActive, setActiveLoading]);

  return <Crud config={enhancedConfig} />;
}

