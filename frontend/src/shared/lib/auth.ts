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
          console.error("[Auth] Missing credentials");
          return null;
        }

        try {
          // Create server-side Apollo Client for auth mutation
          const client = createServerApolloClient();

          console.log("[Auth] Attempting login for:", credentials.email);
          console.log("[Auth] Apollo client created:", !!client);

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

          console.log("[Auth] LOGIN_MUTATION:", LOGIN_MUTATION);
          console.log("[Auth] LOGIN_MUTATION.loc:", LOGIN_MUTATION.loc);
          console.log("[Auth] LOGIN_MUTATION.definitions:", LOGIN_MUTATION.definitions);

          const variables = {
            email: credentials.email as string,
            password: credentials.password as string,
          };
          console.log("[Auth] Variables:", variables);

          let mutateResult;
          try {
            mutateResult = await client.mutate({
              mutation: LOGIN_MUTATION,
              variables,
            });
            console.log("[Auth] Mutate raw result:", mutateResult);
          } catch (mutateError) {
            console.error("[Auth] Mutate threw error:", mutateError);
            console.error("[Auth] Error details:", {
              message: mutateError instanceof Error ? mutateError.message : String(mutateError),
              name: mutateError instanceof Error ? mutateError.name : undefined,
              stack: mutateError instanceof Error ? mutateError.stack : undefined,
            });
            throw mutateError;
          }

          const { data, errors } = mutateResult;

          console.log("[Auth] Mutate completed - data:", data);
          console.log("[Auth] Mutate completed - errors:", errors);

          if (errors && errors.length > 0) {
            console.error("[Auth] GraphQL errors:", errors);
            return null;
          }

          console.log("[Auth] Login response data:", JSON.stringify(data, null, 2));
          console.log("[Auth] data?.login:", data?.login);
          console.log("[Auth] data?.login?.user:", data?.login?.user);
          console.log("[Auth] data?.login?.token:", data?.login?.token);

          if (!data?.login?.user) {
            console.error("[Auth] No user in response", {
              hasData: !!data,
              hasLogin: !!data?.login,
              hasUser: !!data?.login?.user,
              dataKeys: data ? Object.keys(data) : [],
              loginKeys: data?.login ? Object.keys(data.login) : []
            });
            return null;
          }

          const user = data.login.user;

          // Return user object for Next-Auth session
          return {
            id: user.id,
            email: user.email || "",
            name: user.firstName
              ? `${user.firstName} ${user.lastName}`.trim()
              : user.email || "User",
            image: null,
            // Store access token in session (will be used for cookie)
            accessToken: data.login.token,
          };
        } catch (error) {
          console.error("[Auth] Exception during login:", error);
          return null;
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
      // Initial sign in - store accessToken in JWT (server-side only)
      if (account && user) {
        token.accessToken = user.accessToken;
        token.userId = user.id;
        token.lastFetch = Date.now();
      }

      // If no accessToken, session is invalid - return empty token
      // This happens after 401 error invalidates the session
      if (!token.accessToken) {
        console.warn('[Next-Auth] No accessToken in JWT - session invalid');
        return {};
      }

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
              console.error('[Next-Auth] 401 error on GetCurrentUser - invalidating session');
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
              console.error(
                "[Next-Auth] 401 network error on GetCurrentUser - invalidating session"
              );
              return {};
            }
          }

          console.error("[Next-Auth] Failed to fetch full user data:", error);
          // Don't invalidate session on other network errors, just keep old data
        }
      }

      return token;
    },
    async session({ session, token }): Promise<Session> {
      // Populate session with full user data from GraphQL
      if (token.fullUser) {
        return {
          ...session,
          user: token.fullUser,
        };
      }

      // Fallback: if no fullUser yet, return minimal user data
      // This should only happen briefly during initial sign in
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
