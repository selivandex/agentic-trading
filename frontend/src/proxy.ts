/**
 * Next.js Proxy with Next-Auth
 *
 * Protects routes using Next-Auth session.
 * This is a simplified version - Next-Auth handles most of the auth logic.
 */

import { auth } from "@/shared/lib/auth-middleware";

// Export auth as default export (Next.js 16 proxy requirement)
export default auth;

/**
 * Matcher configuration
 * Apply proxy to protected routes only
 */
export const config = {
  matcher: [
    /*
     * Match all request paths except:
     * - api routes
     * - _next/static (static files)
     * - _next/image (image optimization)
     * - favicon.ico
     * - login page
     */
    "/((?!api|_next/static|_next/image|favicon.ico|login|.*\\.(?:svg|png|jpg|jpeg|gif|webp)$).*)",
  ],
};
