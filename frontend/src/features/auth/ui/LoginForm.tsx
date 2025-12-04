"use client";

import { useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { signIn } from "next-auth/react";
import { Input } from "@/shared/base";
import { Button } from "@/shared/base";
import { logger } from "@/shared/lib";

export const LoginForm = () => {
  const router = useRouter();
  const searchParams = useSearchParams();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);
    setLoading(true);

    try {
      // Use Next-Auth signIn
      const result = await signIn("credentials", {
        email,
        password,
        redirect: false,
      });

      if (!result?.ok) {
        throw new Error(result?.error || "Sign in failed");
      }

      // Get redirect URL from query params or default to dashboard
      const redirectUrl = searchParams.get("redirect") || "/dashboard";

      // Redirect after successful login
      router.push(redirectUrl);
    } catch (err) {
      logger.error("Sign in error:", err);
      setFormError(
        err instanceof Error
          ? err.message
          : "Failed to sign in. Please check your credentials."
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {formError && (
        <div className="p-3 bg-utility-error-50 border border-utility-error-200 rounded-lg text-utility-error-700 text-sm">
          {formError}
        </div>
      )}

      <Input
        label="Email"
        type="email"
        value={email}
        onChange={(value) => setEmail(value as string)}
        isRequired
        isDisabled={loading}
        placeholder="Enter your email"
      />

      <Input
        label="Password"
        type="password"
        value={password}
        onChange={(value) => setPassword(value as string)}
        isRequired
        isDisabled={loading}
        placeholder="Enter your password"
      />

      <Button
        type="submit"
        disabled={loading}
        size="xl"
        className="w-full"
      >
        {loading ? "Signing in..." : "Sign in"}
      </Button>
    </form>
  );
};
