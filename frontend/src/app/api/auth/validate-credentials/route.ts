/**
 * API Route for validating credentials before NextAuth signIn
 *
 * This endpoint validates user credentials and returns detailed error messages
 * that can be displayed on the login form. NextAuth's authorize callback
 * doesn't expose detailed error messages to the client for security reasons,
 * so we use this pre-validation endpoint.
 */

import { NextRequest, NextResponse } from "next/server";
import { createServerApolloClient } from "@/shared/api/apollo-server-client";
import { gql } from "@apollo/client";
import { logger } from "@/shared/lib/logger";

const LOGIN_MUTATION = gql`
  mutation ValidateLogin($email: String!, $password: String!) {
    login(input: { email: $email, password: $password }) {
      token
      user {
        id
        email
      }
    }
  }
`;

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const { email, password } = body;

    if (!email || !password) {
      return NextResponse.json(
        { error: "Email and password are required" },
        { status: 400 }
      );
    }

    // Create server-side Apollo Client for auth mutation
    const client = createServerApolloClient();

    const { data, errors } = await client.mutate({
      mutation: LOGIN_MUTATION,
      variables: { email, password },
    });

    if (errors && errors.length > 0) {
      const errorMessage = errors[0]?.message || "Authentication failed";
      return NextResponse.json(
        { error: errorMessage },
        { status: 401 }
      );
    }

    if (!data?.login?.user) {
      return NextResponse.json(
        { error: "Invalid response from server" },
        { status: 500 }
      );
    }

    // Validation successful
    return NextResponse.json({ success: true });
  } catch (error) {
    logger.error("[Validate Credentials] Error:", error);

    return NextResponse.json(
      {
        error: error instanceof Error
          ? error.message
          : "Authentication failed. Please try again."
      },
      { status: 500 }
    );
  }
}
