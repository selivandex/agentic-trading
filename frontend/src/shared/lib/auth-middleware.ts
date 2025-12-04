/**
 * Next-Auth Middleware
 * 
 * Wraps Next-Auth's auth() function with custom redirect logic.
 * Saves last visited organization and project IDs to cookies.
 */

import NextAuth from "next-auth";
import { NextResponse } from "next/server";
import { authConfig } from "./auth";

const { auth: nextAuthMiddleware } = NextAuth(authConfig);

const LAST_ORG_COOKIE = "flowly_last_org_id";
const LAST_PROJECT_COOKIE = "flowly_last_project_id";
const COOKIE_MAX_AGE = 365 * 24 * 60 * 60; // 1 year

export const auth = nextAuthMiddleware((req) => {
  const { pathname } = req.nextUrl;
  // Check if user has valid session with user data
  // req.auth will be null if JWT callback cleared the session (401 error)
  const isLoggedIn = !!req.auth?.user;

  // Extract org_id and project_id from pathname if present
  // Pattern: /[org_id]/[project_id]/...
  const pathMatch = pathname.match(/^\/([^\/]+)\/([^\/]+)/);
  
  // Save last context if user is navigating to org/project page
  if (isLoggedIn && pathMatch) {
    const [, orgId, projectId] = pathMatch;
    
    // Only save if both IDs look valid (not special Next.js routes)
    if (orgId && projectId && !orgId.startsWith("_") && !projectId.startsWith("_")) {
      // Create response to set cookies
      const response = NextResponse.next();
      
      response.cookies.set(LAST_ORG_COOKIE, orgId, {
        maxAge: COOKIE_MAX_AGE,
        path: "/",
        sameSite: "lax",
        secure: process.env.NODE_ENV === "production",
      });
      
      response.cookies.set(LAST_PROJECT_COOKIE, projectId, {
        maxAge: COOKIE_MAX_AGE,
        path: "/",
        sameSite: "lax",
        secure: process.env.NODE_ENV === "production",
      });
      
      return response;
    }
  }

  // Root page - let the page component handle redirect logic
  // (it will check for last context or redirect to first org/project)
  if (pathname === "/" && !isLoggedIn) {
    return Response.redirect(new URL("/login", req.url));
  }

  // Auth pages - redirect to root if already logged in
  // Root page will handle redirecting to last context or first org/project
  const authPages = ["/login", "/signup", "/forgot-password"];
  if (authPages.includes(pathname) && isLoggedIn) {
    return Response.redirect(new URL("/", req.url));
  }

  // Protected routes - redirect to login if not authenticated
  if (!isLoggedIn && !authPages.includes(pathname)) {
    const url = new URL("/login", req.url);
    url.searchParams.set("redirect", pathname);
    return Response.redirect(url);
  }

  return undefined; // Continue to next middleware/page
});

