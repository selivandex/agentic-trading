/**
 * Next-Auth TypeScript Module Augmentation
 * 
 * Extends Next-Auth types to include custom fields:
 * - JWT stores accessToken server-side (httpOnly cookie)
 * - Session only exposes user info (no token)
 */

import "next-auth";
import "next-auth/jwt";
import type { GetCurrentUserQuery } from "@/shared/api/generated/graphql";

// Extract User type from GraphQL query
type FullUser = NonNullable<GetCurrentUserQuery["currentUser"]>;
type MembershipEdge = NonNullable<NonNullable<FullUser["memberships"]>["edges"]>[number];
type Membership = NonNullable<MembershipEdge>["node"];

declare module "next-auth" {
  /**
   * Extends the built-in session type
   * 
   * SECURITY: accessToken is NOT exposed to client
   * Token is sent via httpOnly cookie in /api/graphql proxy
   * 
   * Session contains full user data from GraphQL including memberships
   */
  interface Session {
    user: FullUser;
  }

  /**
   * Extends the built-in user type
   * Used during sign in to store token in JWT
   */
  interface User {
    id: string;
    email: string;
    name?: string | null;
    image?: string | null;
    accessToken?: string; // Only used during initial sign in
    expiresIn?: number;
  }
}

declare module "next-auth/jwt" {
  /**
   * Extends the built-in JWT type
   * 
   * JWT is stored in encrypted httpOnly cookie (authjs.session-token)
   * accessToken is NEVER exposed to browser
   * Proxy extracts it server-side to create flowly_access_token for Rails
   */
  interface JWT {
    accessToken?: string; // Rails JWT token (encrypted in NextAuth JWT, never sent to browser)
    userId?: string;
    fullUser?: FullUser; // Full user data from GraphQL
    lastFetch?: number; // Timestamp of last data fetch
  }
}

