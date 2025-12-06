/**
 * GraphQL API Proxy Route
 *
 * Proxies GraphQL requests to Go backend API.
 * Extracts JWT from NextAuth session and sets as HTTP-only cookie.
 *
 * Architecture:
 * - Browser stores: authjs.session-token (encrypted NextAuth JWT)
 * - NextAuth JWT contains: Go JWT accessToken (encrypted)
 * - Proxy extracts accessToken from NextAuth JWT (server-side)
 * - Proxy sets: Cookie: auth_token=xxx for Go backend
 * - Browser NEVER sees accessToken directly!
 *
 * Benefits:
 * - No CORS issues (same origin)
 * - Token encrypted in NextAuth JWT (secure)
 * - Single source of truth (NextAuth JWT)
 * - HTTP-only cookies for better security
 */

import { NextRequest, NextResponse } from "next/server";
import { getToken } from "next-auth/jwt";

// Backend GraphQL endpoint
const BACKEND_GRAPHQL_URL =
  process.env.BACKEND_GRAPHQL_URL || "http://localhost:8080/graphql";

/**
 * POST handler for GraphQL requests
 * Forwards requests to Rails backend with cookies
 */
export async function POST(request: NextRequest) {
  try {
    // Get request body
    const body = await request.json();

    // Debug: log request cookies
    console.log('[GraphQL Proxy] 📥 Request cookies:', {
      cookies: request.cookies.getAll(),
      cookieHeader: request.headers.get('cookie'),
    });

    // Extract JWT from NextAuth session (server-side)
    const token = await getToken({
      req: request,
      secret: process.env.NEXTAUTH_SECRET,
      cookieName: "prometheus.session-token", // Must match authConfig.cookies.sessionToken.name
    });
    const accessToken = token?.accessToken as string | undefined;

    console.log('[GraphQL Proxy] 🔍 Session token:', {
      hasToken: !!token,
      hasAccessToken: !!accessToken,
      tokenKeys: token ? Object.keys(token) : [],
      accessTokenLength: accessToken?.length,
      token: token, // Full token for debugging
    });

    if (!accessToken) {
      console.warn('[GraphQL Proxy] ⚠️  No accessToken in NextAuth JWT - user not authenticated');
    }

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      "Accept": "application/json",
    };

    // Set auth token as Authorization Bearer (Go backend accepts both Bearer and Cookie)
    if (accessToken) {
      headers["Authorization"] = `Bearer ${accessToken}`;
      console.log('[GraphQL Proxy] ✅ Sending Bearer token to backend:', `${accessToken.substring(0, 20)}...`);
    }

    // Forward request to backend
    const response = await fetch(BACKEND_GRAPHQL_URL, {
      method: "POST",
      headers,
      body: JSON.stringify(body),
    });

    // Get response data
    const data = await response.json();

    // Create Next.js response
    const nextResponse = NextResponse.json(data, {
      status: response.status,
    });

    // Forward Set-Cookie headers from backend to browser if any
    const setCookieHeaders = response.headers.getSetCookie?.() || [];
    setCookieHeaders.forEach((cookie) => {
      nextResponse.headers.append("Set-Cookie", cookie);
    });

    return nextResponse;
  } catch (error) {
    console.error("GraphQL proxy error:", error);

    return NextResponse.json(
      {
        errors: [
          {
            message: "Failed to connect to GraphQL API",
            extensions: {
              code: "PROXY_ERROR",
            },
          },
        ],
      },
      { status: 500 }
    );
  }
}

/**
 * GET handler for GraphQL requests (for GET queries)
 * Some GraphQL clients send GET requests for queries
 */
export async function GET(request: NextRequest) {
  try {
    const { searchParams } = request.nextUrl;
    const query = searchParams.get("query");
    const variables = searchParams.get("variables");
    const operationName = searchParams.get("operationName");

    if (!query) {
      return NextResponse.json(
        {
          errors: [
            {
              message: "Query parameter is required",
            },
          ],
        },
        { status: 400 }
      );
    }

    // Build query params
    const params = new URLSearchParams();
    params.set("query", query);
    if (variables) params.set("variables", variables);
    if (operationName) params.set("operationName", operationName);

    // Extract JWT from NextAuth session (server-side)
    const token = await getToken({
      req: request,
      secret: process.env.NEXTAUTH_SECRET,
      cookieName: "prometheus.session-token", // Must match authConfig.cookies.sessionToken.name
    });
    const accessToken = token?.accessToken as string | undefined;

    // Forward request to backend with Authorization Bearer
    const response = await fetch(`${BACKEND_GRAPHQL_URL}?${params.toString()}`, {
      method: "GET",
      headers: {
        "Accept": "application/json",
        // Set auth token as Bearer (Go backend accepts both Bearer and Cookie)
        ...(accessToken && { "Authorization": `Bearer ${accessToken}` }),
      },
    });

    const data = await response.json();

    // Create Next.js response
    const nextResponse = NextResponse.json(data, {
      status: response.status,
    });

    // Forward Set-Cookie headers from Rails to browser
    const setCookieHeaders = response.headers.getSetCookie?.() || [];
    setCookieHeaders.forEach((cookie) => {
      nextResponse.headers.append("Set-Cookie", cookie);
    });

    return nextResponse;
  } catch (error) {
    console.error("GraphQL proxy error:", error);

    return NextResponse.json(
      {
        errors: [
          {
            message: "Failed to connect to GraphQL API",
            extensions: {
              code: "PROXY_ERROR",
            },
          },
        ],
      },
      { status: 500 }
    );
  }
}
