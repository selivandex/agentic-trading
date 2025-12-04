/**
 * Next-Auth Middleware
 *
 * Protects routes and handles authentication redirects.
 */

import NextAuth from "next-auth";
import { authConfig } from "./auth";

const { auth: nextAuthMiddleware } = NextAuth(authConfig);

// No context cookies needed for trading platform
export const auth = nextAuthMiddleware((req) => {
  const { pathname } = req.nextUrl;
  const isLoggedIn = !!req.auth?.user;

  // Root page - redirect to login if not authenticated
  if (pathname === "/" && !isLoggedIn) {
    return Response.redirect(new URL("/login", req.url));
  }

  // Auth page - redirect to dashboard if already logged in
  if (pathname === "/login" && isLoggedIn) {
    return Response.redirect(new URL("/dashboard", req.url));
  }

  // Protected routes - redirect to login if not authenticated
  if (!isLoggedIn && pathname !== "/login") {
    const url = new URL("/login", req.url);
    url.searchParams.set("redirect", pathname);
    return Response.redirect(url);
  }

  return undefined; // Continue to next middleware/page
});
