/**
 * GraphQL API Proxy Route
 * 
 * Proxies GraphQL requests to Rails backend API.
 * Extracts Rails JWT from NextAuth session and adds as flowly_access_token cookie.
 * 
 * Architecture:
 * - Browser stores: authjs.session-token (encrypted NextAuth JWT)
 * - NextAuth JWT contains: Rails accessToken (encrypted)
 * - Proxy extracts accessToken from NextAuth JWT (server-side)
 * - Proxy creates: Cookie: flowly_access_token=xxx for Rails
 * - Browser NEVER sees flowly_access_token directly!
 * 
 * Benefits:
 * - No CORS issues (same origin)
 * - Token encrypted in NextAuth JWT (secure)
 * - flowly_access_token created on-the-fly for each request
 * - Single source of truth (NextAuth JWT)
 */

import { NextRequest, NextResponse } from "next/server";
import { getToken } from "next-auth/jwt";

// Backend GraphQL endpoint
const BACKEND_GRAPHQL_URL =
  process.env.BACKEND_GRAPHQL_URL || "http://api.lvh.me:3000/graphql";

/**
 * POST handler for GraphQL requests
 * Forwards requests to Rails backend with cookies
 */
export async function POST(request: NextRequest) {
  try {
    // Get request body
    const body = await request.json();

    // Extract Rails JWT from NextAuth session (server-side)
    const token = await getToken({ 
      req: request,
      secret: process.env.NEXTAUTH_SECRET 
    });
    const railsAccessToken = token?.accessToken as string | undefined;

    // Build Cookie header with flowly_access_token for Rails
    let cookieHeader = request.headers.get("cookie") || "";
    
    if (railsAccessToken) {
      // Add flowly_access_token cookie (created on-the-fly from NextAuth JWT)
      const railsCookie = `flowly_access_token=${railsAccessToken}`;
      cookieHeader = cookieHeader ? `${cookieHeader}; ${railsCookie}` : railsCookie;
    } else {
      console.warn('[GraphQL Proxy] ⚠️  No accessToken in NextAuth JWT - user not authenticated');
    }

    // Get organization and project context from client headers
    const organizationId = request.headers.get("X-Flowly-Organization");
    const projectId = request.headers.get("X-Flowly-Project");

    // Forward request to backend
    const response = await fetch(BACKEND_GRAPHQL_URL, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        // Forward all cookies from browser (includes flowly_access_token from Rails)
        ...(cookieHeader && { Cookie: cookieHeader }),
        // Forward organization and project context headers
        ...(organizationId && { "X-Flowly-Organization": organizationId }),
        ...(projectId && { "X-Flowly-Project": projectId }),
        "Accept": "application/json",
      },
      body: JSON.stringify(body),
      credentials: "include",
    });

    // Get response data
    const data = await response.json();

    // Create Next.js response
    const nextResponse = NextResponse.json(data, {
      status: response.status,
    });

    // Forward Set-Cookie headers from Rails to browser
    // Note: Rails should NOT set flowly_access_token cookie
    // (it's managed by NextAuth JWT and created on-the-fly by proxy)
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

    // Extract Rails JWT from NextAuth session (server-side)
    const token = await getToken({ 
      req: request,
      secret: process.env.NEXTAUTH_SECRET 
    });
    const railsAccessToken = token?.accessToken as string | undefined;

    // Build Cookie header with flowly_access_token for Rails
    let cookieHeader = request.headers.get("cookie") || "";
    
    if (railsAccessToken) {
      // Add flowly_access_token cookie (created on-the-fly from NextAuth JWT)
      const railsCookie = `flowly_access_token=${railsAccessToken}`;
      cookieHeader = cookieHeader ? `${cookieHeader}; ${railsCookie}` : railsCookie;
    }
    
    // Get organization and project context from client headers
    const organizationId = request.headers.get("X-Flowly-Organization");
    const projectId = request.headers.get("X-Flowly-Project");

    // Forward request to backend
    const response = await fetch(`${BACKEND_GRAPHQL_URL}?${params.toString()}`, {
      method: "GET",
      headers: {
        "Accept": "application/json",
        // Forward all cookies from browser
        ...(cookieHeader && { Cookie: cookieHeader }),
        // Forward organization and project context headers
        ...(organizationId && { "X-Flowly-Organization": organizationId }),
        ...(projectId && { "X-Flowly-Project": projectId }),
      },
      credentials: "include",
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

