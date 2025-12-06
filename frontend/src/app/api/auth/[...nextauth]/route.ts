/**
 * Next-Auth API Route Handler with Sign Out Middleware
 *
 * Handles all authentication routes:
 * - /api/auth/signin
 * - /api/auth/signout (+ calls Rails to invalidate token)
 * - /api/auth/session
 * - /api/auth/providers
 * - /api/auth/csrf
 */

import { NextRequest } from "next/server";
import { handlers } from "@/shared/lib/auth";
import { getToken } from "next-auth/jwt";
import { createServerApolloClient } from "@/shared/api/apollo-server-client";
import { gql } from "@apollo/client";

const LOGOUT_MUTATION = gql`
  mutation Logout {
    logout
  }
`;

export const { GET } = handlers;

/**
 * POST handler with sign out middleware
 * Intercepts /api/auth/signout to invalidate token on Rails backend
 */
export async function POST(request: NextRequest) {
  const url = new URL(request.url);

  // Check if this is a signout request
  if (url.pathname.includes("/signout")) {
    try {
      // Extract Go JWT from NextAuth session (server-side)
      // This is the same way /api/graphql proxy extracts the token
      const token = await getToken({
        req: request,
        secret: process.env.NEXTAUTH_SECRET,
        cookieName: "prometheus.session-token", // Must match authConfig.cookies.sessionToken.name
      });
      const railsAccessToken = token?.accessToken as string | undefined;

      // Try to invalidate token on backend
      if (railsAccessToken) {
        try {
          // Create server-side Apollo Client for signOut mutation
          const client = createServerApolloClient();

          await client.mutate({
            mutation: LOGOUT_MUTATION,
            context: {
              accessToken: railsAccessToken, // Pass JWT token
            },
          });
        } catch (error) {
          // Backend signOut failed - continue with Next-Auth signout anyway
          console.error("Backend sign out error (ignored):", error);
        }
      } else {
        console.warn("[Sign Out] No accessToken in NextAuth JWT - user already signed out or token expired");
      }
    } catch (error) {
      console.error("Sign out middleware error (ignored):", error);
    }
  }

  // Continue with Next-Auth handler (clears session, redirects, etc.)
  return handlers.POST(request);
}
