/** @format */

"use client";

import {
  LayoutDashboard,
  TrendingUp,
  Eye,
  User as UserIcon,
  Settings,
} from "lucide-react";
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
      icon: LayoutDashboard,
    },
    {
      label: "Strategies",
      href: "/strategies",
      icon: TrendingUp,
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
      icon: Settings,
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
