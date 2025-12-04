/**
 * Toast Provider Component
 *
 * Wraps app with Sonner Toaster component
 * Uses custom Toaster with gradient overlay from notifications
 *
 * @format
 */

"use client";

import { Toaster } from "@/shared/application/notifications/toaster";

export const ToastProvider = () => {
  return <Toaster />;
};
