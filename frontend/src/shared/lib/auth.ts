/**
 * Next-Auth Configuration
 *
 * Integrates with Rails GraphQL API for authentication.
 * Uses Credentials provider to call signIn/signUp mutations.
 * Uses shared Apollo Client with direct backend URL override.
 */

import NextAuth from "next-auth";
import type { NextAuthConfig, Session } from "next-auth";
import Credentials from "next-auth/providers/credentials";
import { createServerApolloClient } from "@/shared/api/apollo-server-client";
import { GetCurrentUserDocument } from "@/shared/api/generated/graphql";
import { ApolloError, gql } from "@apollo/client";
import { logger } from "./logger";

export const authConfig: NextAuthConfig = {
  providers: [
    Credentials({
      name: "credentials",
      credentials: {
        email: { label: "Email", type: "email" },
        password: { label: "Password", type: "password" },
      },
      async authorize(credentials) {
        if (!credentials?.email || !credentials?.password) {
          throw new Error("Email and password are required");
        }

        try {
          // Create server-side Apollo Client for auth mutation
          const client = createServerApolloClient();

          // Use raw GraphQL string instead of generated document for server-side compatibility
          const LOGIN_MUTATION = gql`
            mutation Login($email: String!, $password: String!) {
              login(input: { email: $email, password: $password }) {
                token
                user {
                  id
                  email
                  firstName
                  lastName
                  telegramID
                  telegramUsername
                  languageCode
                  isActive
                  isPremium
                  createdAt
                  updatedAt
                }
              }
            }
          `;

          const variables = {
            email: credentials.email as string,
            password: credentials.password as string,
          };

          const mutateResult = await client.mutate({
            mutation: LOGIN_MUTATION,
            variables,
          });

          const { data, errors } = mutateResult;

          if (errors && errors.length > 0) {
            // Throw error with message from GraphQL to pass it to the client
            const errorMessage = errors[0]?.message || "Authentication failed";
            throw new Error(errorMessage);
          }

          if (!data?.login?.user) {
            throw new Error("Invalid response from server");
          }

          const user = data.login.user;

          console.log('[Next-Auth Authorize] Login successful:', {
            userId: user.id,
            email: user.email,
            hasToken: !!data.login.token,
            tokenLength: data.login.token?.length,
          });

          // Return user object for Next-Auth session
          const authUser = {
            id: user.id,
            email: user.email || "",
            name: user.firstName
              ? `${user.firstName} ${user.lastName}`.trim()
              : user.email || "User",
            image: null,
            // Store access token in session (will be used for cookie)
            accessToken: data.login.token,
          };

          console.log('[Next-Auth Authorize] Returning user with accessToken:', {
            hasAccessToken: !!authUser.accessToken,
          });

          return authUser;
        } catch (error) {
          logger.error("[Auth] Login failed:", error);
          // Re-throw to pass error message to the client
          if (error instanceof Error) {
            throw error;
          }
          throw new Error("Authentication failed. Please try again.");
        }
      },
    }),
  ],
  pages: {
    signIn: "/login",
    signOut: "/login",
    error: "/login",
  },
  callbacks: {
    async jwt({ token, user, account, trigger }) {
      console.log('[Next-Auth JWT] Callback triggered:', {
        hasAccount: !!account,
        hasUser: !!user,
        trigger,
        hasAccessTokenInToken: !!token.accessToken,
        userAccessToken: user?.accessToken?.substring(0, 20),
      });

      // Initial sign in - store accessToken in JWT (server-side only)
      if (account && user) {
        console.log('[Next-Auth JWT] Initial sign in - storing accessToken');
        token.accessToken = user.accessToken;
        token.userId = user.id;
        token.lastFetch = Date.now();
      }

      // If no accessToken, session is invalid - return empty token
      // This happens after 401 error invalidates the session
      if (!token.accessToken) {
        logger.warn('[Next-Auth] No accessToken in JWT - session invalid');
        console.log('[Next-Auth JWT] Returning empty token - will invalidate session');
        return {};
      }

      console.log('[Next-Auth JWT] Token has accessToken, continuing...');

      // Fetch full user data from GraphQL (with memberships)
      // Refresh every 5 minutes or on update trigger
      const shouldFetch =
        !token.fullUser ||
        !token.lastFetch ||
        Date.now() - token.lastFetch > 5 * 60 * 1000 ||
        trigger === "update";

      if (shouldFetch) {
        try {
          // Create server-side Apollo Client for fetching user data
          const client = createServerApolloClient();

          const { data, errors } = await client.query({
            query: GetCurrentUserDocument,
            context: {
              accessToken: token.accessToken as string, // Pass JWT token
            },
            fetchPolicy: 'network-only',
          });

          // Check for authentication errors (401 Unauthorized)
          if (errors && errors.length > 0) {
            const hasAuthError = errors.some(error =>
              error.extensions?.code === 'UNAUTHENTICATED' ||
              error.extensions?.code === 'UNAUTHORIZED'
            );

            if (hasAuthError) {
              logger.error('[Next-Auth] 401 error on GetCurrentUser - invalidating session');
              // Return empty token to invalidate NextAuth session
              // This will cause middleware to redirect to /login
              return {};
            }
          }

          if (data?.me) {
            token.fullUser = data.me;
            token.lastFetch = Date.now();
          }
        } catch (error) {
          if (error instanceof ApolloError) {
            const statusCode =
              (error.networkError as { statusCode?: number } | null)?.statusCode;

            if (statusCode === 401) {
              logger.error(
                "[Next-Auth] 401 network error on GetCurrentUser - invalidating session"
              );
              return {};
            }
          }

          logger.error("[Next-Auth] Failed to fetch full user data:", error);
          // Don't invalidate session on other network errors, just keep old data
        }
      }

      return token;
    },
    async session({ session, token }): Promise<Session> {
      console.log('[Next-Auth Session] Callback triggered:', {
        hasToken: !!token,
        hasAccessToken: !!token.accessToken,
        hasFullUser: !!token.fullUser,
      });

      // Populate session with full user data from GraphQL
      if (token.fullUser) {
        return {
          ...session,
          user: token.fullUser,
        };
      }

      // Fallback: if no fullUser yet, return minimal user data
      // This should only happen briefly during initial sign in
      console.warn('[Next-Auth Session] No fullUser in token - returning default session');
      return session;
    },
  },
  session: {
    strategy: "jwt",
    maxAge: 365 * 24 * 60 * 60, // 1 year
  },
  cookies: {
    sessionToken: {
      name: "prometheus.session-token",
      options: {
        httpOnly: true,
        sameSite: "lax",
        path: "/",
        secure: process.env.NODE_ENV === "production",
      },
    },
    callbackUrl: {
      name: "prometheus.callback-url",
      options: {
        httpOnly: true,
        sameSite: "lax",
        path: "/",
        secure: process.env.NODE_ENV === "production",
      },
    },
    csrfToken: {
      name: "prometheus.csrf-token",
      options: {
        httpOnly: true,
        sameSite: "lax",
        path: "/",
        secure: process.env.NODE_ENV === "production",
      },
    },
  },
};

// Export Next-Auth instance
export const { handlers, auth, signIn: nextAuthSignIn, signOut: nextAuthSignOut } = NextAuth(authConfig);
