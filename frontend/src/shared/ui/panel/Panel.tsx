/** @format */

"use client";

import React from "react";
import { cx } from "@/utils/cx";

/**
 * Panel Component
 * Flexible container for displaying structured content with optional header, footer, and actions
 * Perfect for show pages, dashboards, and any grouped content
 */

interface PanelProps {
  /** Panel content */
  children: React.ReactNode;
  /** Optional CSS classes */
  className?: string;
}

interface PanelHeaderProps {
  /** Header title */
  title?: string;
  /** Header subtitle/description */
  subtitle?: string;
  /** Custom title render (overrides title string) */
  children?: React.ReactNode;
  /** Optional CSS classes */
  className?: string;
}

interface PanelContentProps {
  /** Content */
  children: React.ReactNode;
  /** Optional CSS classes */
  className?: string;
  /** Remove default padding */
  noPadding?: boolean;
}

interface PanelActionsProps {
  /** Action buttons/elements */
  children: React.ReactNode;
  /** Optional CSS classes */
  className?: string;
}

interface PanelFooterProps {
  /** Footer content */
  children: React.ReactNode;
  /** Optional CSS classes */
  className?: string;
}

interface PanelDividerProps {
  /** Optional CSS classes */
  className?: string;
}

/**
 * Main Panel Container
 */
export function Panel({ children, className }: PanelProps) {
  return (
    <div
      className={cx(
        "rounded-xl bg-primary shadow-sm ring-1 ring-secondary overflow-hidden",
        className
      )}
    >
      {children}
    </div>
  );
}

/**
 * Panel Header
 * Contains title, subtitle, and optional actions
 */
Panel.Header = function PanelHeader({
  title,
  subtitle,
  children,
  className,
}: PanelHeaderProps) {
  if (children) {
    return <div className={cx("px-6 py-4", className)}>{children}</div>;
  }

  return (
    <div className={cx("px-6 py-4", className)}>
      {title && <h3 className="text-lg font-semibold text-primary">{title}</h3>}
      {subtitle && <p className="mt-1 text-sm text-secondary">{subtitle}</p>}
    </div>
  );
};

/**
 * Panel Content
 * Main content area
 */
Panel.Content = function PanelContent({
  children,
  className,
  noPadding = false,
}: PanelContentProps) {
  return (
    <div className={cx(!noPadding && "px-6 py-4", className)}>{children}</div>
  );
};

/**
 * Panel Actions
 * Action buttons area (typically in header)
 */
Panel.Actions = function PanelActions({
  children,
  className,
}: PanelActionsProps) {
  return (
    <div className={cx("flex items-center gap-2", className)}>{children}</div>
  );
};

/**
 * Panel Footer
 * Footer area for additional info or actions
 */
Panel.Footer = function PanelFooter({ children, className }: PanelFooterProps) {
  return (
    <div
      className={cx(
        "px-6 py-4 bg-secondary/5 border-t border-secondary",
        className
      )}
    >
      {children}
    </div>
  );
};

/**
 * Panel Divider
 * Visual separator between sections
 */
Panel.Divider = function PanelDivider({ className }: PanelDividerProps) {
  return <div className={cx("border-t border-secondary", className)} />;
};

/**
 * Compound export for convenience
 */
Panel.displayName = "Panel";
