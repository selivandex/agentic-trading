/**
 * Root page - redirects authenticated users to dashboard
 *
 * @format
 */

import { redirect } from "next/navigation";
import { auth } from "@/shared/lib/auth";

export default async function Home() {
  // Check if user is authenticated
  const session = await auth();

  // If not authenticated, redirect to login
  if (!session?.user) {
    redirect("/login");
  }

  // Redirect to dashboard
  redirect("/dashboard");
}
