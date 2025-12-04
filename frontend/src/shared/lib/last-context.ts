/**
 * Last Context Utilities (Server-Side Only)
 *
 * Manages last visited organization and project IDs in cookies.
 * Used for automatic redirect on homepage.
 *
 * WARNING: This module uses next/headers and can only be used in Server Components.
 * Do NOT import from client components or re-export through @/shared/lib index.
 * Import directly from this file in server components only.
 */

import { cookies } from "next/headers";

const LAST_ORG_COOKIE = "flowly_last_org_id";
const LAST_PROJECT_COOKIE = "flowly_last_project_id";
const COOKIE_MAX_AGE = 365 * 24 * 60 * 60; // 1 year

/**
 * Get last visited organization and project IDs from cookies
 */
export async function getLastContext(): Promise<{
  organizationId: string | null;
  projectId: string | null;
}> {
  const cookieStore = await cookies();

  return {
    organizationId: cookieStore.get(LAST_ORG_COOKIE)?.value || null,
    projectId: cookieStore.get(LAST_PROJECT_COOKIE)?.value || null,
  };
}

/**
 * Save last visited organization and project IDs to cookies
 */
export async function setLastContext(
  organizationId: string,
  projectId: string
): Promise<void> {
  const cookieStore = await cookies();

  cookieStore.set(LAST_ORG_COOKIE, organizationId, {
    maxAge: COOKIE_MAX_AGE,
    path: "/",
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
  });

  cookieStore.set(LAST_PROJECT_COOKIE, projectId, {
    maxAge: COOKIE_MAX_AGE,
    path: "/",
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
  });
}

/**
 * Clear last context cookies (useful for logout)
 */
export async function clearLastContext(): Promise<void> {
  const cookieStore = await cookies();

  cookieStore.delete(LAST_ORG_COOKIE);
  cookieStore.delete(LAST_PROJECT_COOKIE);
}
