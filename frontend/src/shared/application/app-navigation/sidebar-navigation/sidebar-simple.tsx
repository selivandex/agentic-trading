/** @format */

"use client";

import type { ReactNode } from "react";
import { usePathname } from "next/navigation";
import { cx } from "@/utils/cx";
import type { NavItemType } from "@/components/application/app-navigation/config";
import { NavList } from "@/components/application/app-navigation/base-components/nav-list";
import { NavAccountCard } from "@/components/application/app-navigation/base-components/nav-account-card";
import type { Session } from "next-auth";

interface SidebarNavigationSimpleProps {
  /** List of navigation items to display. */
  items: NavItemType[];
  /** List of footer navigation items to display. */
  footerItems?: NavItemType[];
  /** Feature card component to display in the sidebar. */
  featureCard?: ReactNode;
  /** Header component to display at the top of the sidebar (e.g., project selector). */
  headerSlot?: ReactNode;
  /** Additional CSS classes to apply to the sidebar. */
  className?: string;
  /** Current user data for account card (from Next-Auth session) */
  user?: Session["user"] | null;
  /** Current organization ID (slug from URL) */
  currentOrganizationId: string;
  /** Current project ID (slug from URL) */
  currentProjectId: string;
  /** Sign out handler */
  onSignOut?: () => Promise<void> | void;
}

export const SidebarNavigationSimple = ({
  items,
  footerItems = [],
  featureCard,
  headerSlot,
  className,
  user,
  currentOrganizationId,
  currentProjectId,
  onSignOut,
}: SidebarNavigationSimpleProps) => {
  const pathname = usePathname();

  return (
    <aside
      className={cx(
        "flex h-full w-64 flex-col border-r border-secondary bg-primary",
        className
      )}
    >
      {/* Header slot (e.g., project selector) */}
      {headerSlot && <div>{headerSlot}</div>}

      {/* Navigation items */}
      <nav className="flex-1 overflow-y-auto">
        <NavList activeUrl={pathname} items={items} />
      </nav>

      {/* Feature card */}
      {featureCard && <div className="px-4 pb-4">{featureCard}</div>}

      {/* Footer items */}
      {footerItems.length > 0 && (
        <div className="border-t border-secondary pt-4">
          <NavList activeUrl={pathname} items={footerItems} />
        </div>
      )}

      {/* Account card */}
      <div className="border-t border-secondary p-4">
        <NavAccountCard
          user={user}
          currentOrganizationId={currentOrganizationId}
          currentProjectId={currentProjectId}
          onSignOut={onSignOut}
        />
      </div>
    </aside>
  );
};
