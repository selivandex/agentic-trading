/** @format */

"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  TrendingUp,
  Eye,
  User,
  Settings,
  LogOut
} from "lucide-react";
import { UserAvatar } from "@/entities/user";
import { signOut } from "next-auth/react";
import type { User } from "@/entities/user";

interface DashboardLayoutProps {
  children: React.ReactNode;
  user: User;
}

interface NavItem {
  href: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
}

const navigation: NavItem[] = [
  {
    href: "/dashboard",
    label: "Dashboard",
    icon: LayoutDashboard,
  },
  {
    href: "/strategies",
    label: "Strategies",
    icon: TrendingUp,
  },
  {
    href: "/watchlist",
    label: "Watchlist",
    icon: Eye,
  },
  {
    href: "/profile",
    label: "Profile",
    icon: User,
  },
  {
    href: "/settings",
    label: "Settings",
    icon: Settings,
  },
];

/**
 * DashboardLayout Component
 *
 * Main layout with sidebar navigation for authenticated users
 */
export function DashboardLayout({ children, user }: DashboardLayoutProps) {
  const pathname = usePathname();

  const handleSignOut = async () => {
    await signOut({ callbackUrl: "/login" });
  };

  return (
    <div className="flex h-screen bg-primary">
      {/* Sidebar */}
      <aside className="w-64 border-r border-primary bg-primary flex flex-col">
        {/* Logo */}
        <div className="p-6 border-b border-primary">
          <h1 className="text-xl font-bold text-primary">
            Prometheus Trading
          </h1>
        </div>

        {/* Navigation */}
        <nav className="flex-1 p-4 space-y-1">
          {navigation.map((item) => {
            const Icon = item.icon;
            const isActive = pathname === item.href;

            return (
              <Link
                key={item.href}
                href={item.href}
                className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
                  isActive
                    ? "bg-utility-brand-50 text-utility-brand-700"
                    : "text-secondary hover:bg-utility-gray-50"
                }`}
              >
                <Icon className="w-5 h-5" />
                <span className="font-medium">{item.label}</span>
              </Link>
            );
          })}
        </nav>

        {/* User Section */}
        <div className="p-4 border-t border-primary">
          <div className="flex items-center gap-3 mb-3">
            <UserAvatar user={user} size="sm" />
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-primary truncate">
                {user.firstName} {user.lastName}
              </p>
              <p className="text-xs text-tertiary truncate">
                {user.email}
              </p>
            </div>
          </div>

          <button
            onClick={handleSignOut}
            className="flex items-center gap-2 w-full px-4 py-2 text-sm text-secondary hover:bg-utility-gray-50 rounded-lg transition-colors"
          >
            <LogOut className="w-4 h-4" />
            <span>Sign out</span>
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-auto">
        {children}
      </main>
    </div>
  );
}

