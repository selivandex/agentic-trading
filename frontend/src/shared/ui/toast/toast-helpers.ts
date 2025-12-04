/**
 * Toast Notification Helpers
 * 
 * Utility functions for common toast patterns
 * 
 * @format
 */

import { toast } from "./toast";

/**
 * Helper to handle mutation errors
 */
export const handleMutationError = (
  error: unknown,
  fallbackMessage = "An error occurred"
) => {
  const message = error instanceof Error ? error.message : fallbackMessage;
  toast.error({
    title: "Error",
    description: message,
  });
  console.error("Mutation error:", error);
};

/**
 * Helper to show success message
 */
export const showSuccess = (message: string, title = "Success") => {
  toast.success({
    title,
    description: message,
  });
};
