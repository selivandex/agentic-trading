/**
 * Toast Notification Service
 *
 * Wrapper around Sonner with custom IconNotification component
 * Uses project's design system for consistent UI
 *
 * @format
 */

"use client";

import { toast as sonnerToast } from "sonner";
import { IconNotification } from "@/components/application/notifications/notifications";

interface NotificationOptions {
  title: string;
  description: string;
  confirmLabel?: string;
  dismissLabel?: string;
  hideDismissLabel?: boolean;
  progress?: number;
  onConfirm?: () => void;
  duration?: number;
}

interface ToastReturn {
  success: (options: NotificationOptions | string) => string | number;
  error: (options: NotificationOptions | string) => string | number;
  info: (options: NotificationOptions | string) => string | number;
  warning: (options: NotificationOptions | string) => string | number;
  loading: (options: NotificationOptions | string) => string | number;
  dismiss: (toastId?: string | number) => void;
  promise: <T>(
    promise: Promise<T>,
    messages: {
      loading: NotificationOptions | string;
      success: NotificationOptions | string;
      error: NotificationOptions | string;
    }
  ) => string | number;
}

export const toast: ToastReturn = {
  /**
   * Show success notification with IconNotification component
   */
  success: (options: NotificationOptions | string) => {
    const opts =
      typeof options === "string"
        ? { title: "Success", description: options }
        : options;

    return sonnerToast.custom(
      (id) => (
        <IconNotification
          title={opts.title}
          description={opts.description}
          color="success"
          confirmLabel={opts.confirmLabel}
          dismissLabel={opts.dismissLabel}
          hideDismissLabel={opts.hideDismissLabel}
          progress={opts.progress}
          onClose={() => sonnerToast.dismiss(id)}
          onConfirm={() => {
            opts.onConfirm?.();
            sonnerToast.dismiss(id);
          }}
        />
      ),
      {
        duration: opts.duration ?? 3000,
      }
    );
  },

  /**
   * Show error notification with IconNotification component
   */
  error: (options: NotificationOptions | string) => {
    const opts =
      typeof options === "string"
        ? { title: "Error", description: options }
        : options;

    return sonnerToast.custom(
      (id) => (
        <IconNotification
          title={opts.title}
          description={opts.description}
          color="error"
          confirmLabel={opts.confirmLabel}
          dismissLabel={opts.dismissLabel}
          hideDismissLabel={opts.hideDismissLabel}
          progress={opts.progress}
          onClose={() => sonnerToast.dismiss(id)}
          onConfirm={() => {
            opts.onConfirm?.();
            sonnerToast.dismiss(id);
          }}
        />
      ),
      {
        duration: opts.duration ?? 5000, // Errors stay longer
      }
    );
  },

  /**
   * Show info notification with IconNotification component
   */
  info: (options: NotificationOptions | string) => {
    const opts =
      typeof options === "string"
        ? { title: "Info", description: options }
        : options;

    return sonnerToast.custom(
      (id) => (
        <IconNotification
          title={opts.title}
          description={opts.description}
          color="default"
          confirmLabel={opts.confirmLabel}
          dismissLabel={opts.dismissLabel}
          hideDismissLabel={opts.hideDismissLabel}
          progress={opts.progress}
          onClose={() => sonnerToast.dismiss(id)}
          onConfirm={() => {
            opts.onConfirm?.();
            sonnerToast.dismiss(id);
          }}
        />
      ),
      {
        duration: opts.duration ?? 3000,
      }
    );
  },

  /**
   * Show warning notification with IconNotification component
   */
  warning: (options: NotificationOptions | string) => {
    const opts =
      typeof options === "string"
        ? { title: "Warning", description: options }
        : options;

    return sonnerToast.custom(
      (id) => (
        <IconNotification
          title={opts.title}
          description={opts.description}
          color="warning"
          confirmLabel={opts.confirmLabel}
          dismissLabel={opts.dismissLabel}
          hideDismissLabel={opts.hideDismissLabel}
          progress={opts.progress}
          onClose={() => sonnerToast.dismiss(id)}
          onConfirm={() => {
            opts.onConfirm?.();
            sonnerToast.dismiss(id);
          }}
        />
      ),
      {
        duration: opts.duration ?? 3000,
      }
    );
  },

  /**
   * Show loading notification with progress
   */
  loading: (options: NotificationOptions | string) => {
    const opts =
      typeof options === "string"
        ? { title: "Loading", description: options }
        : options;

    return sonnerToast.custom(
      (id) => (
        <IconNotification
          title={opts.title}
          description={opts.description}
          color="default"
          progress={opts.progress}
          hideDismissLabel
          onClose={() => sonnerToast.dismiss(id)}
        />
      ),
      {
        duration: Infinity, // Loading stays until manually dismissed
      }
    );
  },

  /**
   * Dismiss toast
   */
  dismiss: (toastId?: string | number) => {
    return sonnerToast.dismiss(toastId);
  },

  /**
   * Show promise-based toast (auto success/error)
   */
  promise: <T,>(
    promise: Promise<T>,
    messages: {
      loading: NotificationOptions | string;
      success: NotificationOptions | string;
      error: NotificationOptions | string;
    }
  ) => {
    return sonnerToast.promise(promise, {
      loading:
        typeof messages.loading === "string"
          ? messages.loading
          : messages.loading.description,
      success:
        typeof messages.success === "string"
          ? messages.success
          : messages.success.description,
      error: (err) => {
        const errorMessage =
          err instanceof Error ? err.message : "An error occurred";
        return typeof messages.error === "string"
          ? messages.error
          : messages.error.description || errorMessage;
      },
    }) as string | number;
  },
};

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
