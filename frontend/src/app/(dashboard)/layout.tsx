/** @format */

import { redirect } from "next/navigation";
import { auth } from "@/shared/lib/auth";
import { DashboardLayout } from "@/widgets/dashboard-layout";

/**
 * Dashboard Layout
 *
 * Layout for authenticated users with sidebar navigation
 */
export default async function Layout({
  children,
}: {
  children: React.ReactNode;
}) {
  const session = await auth();

  if (!session?.user) {
    redirect("/login");
  }

  return <DashboardLayout user={session.user}>{children}</DashboardLayout>;
}

