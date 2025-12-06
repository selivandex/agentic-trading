/** @format */

"use client";

import { useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { signIn } from "next-auth/react";
import { Input } from "@/shared/base";
import { Button } from "@/shared/base";
import { logger, useTranslation } from "@/shared/lib";

export const LoginForm = () => {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { t } = useTranslation();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);
    setLoading(true);

    try {
      // First, validate credentials and get detailed error message if any
      const validateResponse = await fetch("/api/auth/validate-credentials", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });

      if (!validateResponse.ok) {
        const errorData = await validateResponse.json();
        const errorMessage = getErrorMessage(errorData.error);
        throw new Error(errorMessage);
      }

      // If validation passed, proceed with NextAuth signIn
      const result = await signIn("credentials", {
        email,
        password,
        redirect: false,
      });

      if (!result?.ok) {
        // This should rarely happen since we pre-validated
        const errorMessage = getErrorMessage(result?.error);
        throw new Error(errorMessage);
      }

      // Get redirect URL from query params or default to dashboard
      const redirectUrl = searchParams.get("redirect") || "/dashboard";

      // Redirect after successful login
      router.push(redirectUrl);
    } catch (err) {
      logger.error("Sign in error:", err);
      setFormError(
        err instanceof Error ? err.message : t("auth.errors.genericError")
      );
    } finally {
      setLoading(false);
    }
  };

  /**
   * Map backend/NextAuth error messages to user-friendly localized messages
   */
  const getErrorMessage = (error?: string | null): string => {
    if (!error) {
      return t("auth.errors.genericError");
    }

    // Check for specific error messages from backend
    if (error.includes("invalid email or password")) {
      return t("auth.errors.invalidCredentials");
    }

    if (error.includes("CredentialsSignin")) {
      return t("auth.errors.invalidCredentials");
    }

    if (error.includes("Invalid response from server")) {
      return t("auth.errors.serverError");
    }

    if (error.includes("Authentication failed")) {
      return t("auth.errors.authenticationFailed");
    }

    // Return original error if no mapping found
    return error;
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {formError && (
        <div className="p-3 bg-utility-error-50 border border-utility-error-200 rounded-lg text-utility-error-700 text-sm">
          {formError}
        </div>
      )}

      <Input
        label={t("auth.login.emailLabel")}
        type="email"
        value={email}
        onChange={(value) => setEmail(value as string)}
        isRequired
        isDisabled={loading}
        placeholder={t("auth.login.emailPlaceholder")}
      />

      <Input
        label={t("auth.login.passwordLabel")}
        type="password"
        value={password}
        onChange={(value) => setPassword(value as string)}
        isRequired
        isDisabled={loading}
        placeholder={t("auth.login.passwordPlaceholder")}
      />

      <Button type="submit" disabled={loading} size="xl" className="w-full">
        {loading
          ? t("auth.login.submittingButton")
          : t("auth.login.submitButton")}
      </Button>
    </form>
  );
};
