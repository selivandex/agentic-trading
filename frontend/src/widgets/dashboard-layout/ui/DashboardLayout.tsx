/** @format */

"use client";

import {
  HomeLine,
  TrendUp01,
  Eye,
  User01 as UserIcon,
  Settings01,
  Users01,
} from "@untitledui/icons";
import { signOut } from "next-auth/react";
import type { User } from "@/entities/user";
import type { NavItemType } from "@/components/application/app-navigation/config";
import { SidebarNavigationSimple } from "@/components/application/app-navigation/sidebar-navigation";

interface DashboardLayoutProps {
  children: React.ReactNode;
  user: User;
}

/**
 * DashboardLayout Component
 *
 * Main layout with sidebar navigation for authenticated users
 */
export function DashboardLayout({ children, user }: DashboardLayoutProps) {
  const handleSignOut = async () => {
    await signOut({ callbackUrl: "/login" });
  };

  const navItems: NavItemType[] = [
    {
      label: "Dashboard",
      href: "/dashboard",
      icon: HomeLine,
    },
    {
      label: "Users",
      href: "/users",
      icon: Users01,
    },
    {
      label: "Strategies",
      href: "/strategies",
      icon: TrendUp01,
    },
    {
      label: "Watchlist",
      href: "/watchlist",
      icon: Eye,
    },
    {
      label: "Profile",
      href: "/profile",
      icon: UserIcon,
    },
    {
      label: "Settings",
      href: "/settings",
      icon: Settings01,
    },
  ];

  return (
    <div className="flex h-screen bg-secondary">
      <SidebarNavigationSimple
        items={navItems}
        user={user}
        currentOrganizationId="default"
        currentProjectId="default"
        onSignOut={handleSignOut}
      />
      <main className="flex-1 overflow-y-auto">{children}</main>
    </div>
  );
}
