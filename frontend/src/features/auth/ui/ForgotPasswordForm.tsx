/** @format */

"use client";

import { useState } from "react";
import Link from "next/link";
import { useForgotPasswordMutation } from "@/shared/api/generated/graphql";

export const ForgotPasswordForm = () => {
  const [forgotPassword, { loading }] = useForgotPasswordMutation();

  const [email, setEmail] = useState("");
  const [success, setSuccess] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);
    setSuccess(false);

    try {
      const { data } = await forgotPassword({
        variables: {
          email,
        },
      });

      if (data?.forgotPassword?.emailSent) {
        setSuccess(true);
      } else {
        throw new Error("Failed to send reset email");
      }
    } catch (err) {
      console.error("Forgot password error:", err);
      setFormError(
        err instanceof Error
          ? err.message
          : "Failed to send reset email. Please try again."
      );
    }
  };

  if (success) {
    return (
      <div className="space-y-4">
        <div className="p-4 bg-green-50 border border-green-200 rounded-lg text-green-700">
          <h3 className="font-semibold mb-2">Check your email</h3>
          <p className="text-sm">
            We&apos;ve sent a password reset link to <strong>{email}</strong>.
            Please check your inbox and follow the instructions.
          </p>
        </div>

        <Link
          href="/login"
          className="block text-center text-sm text-blue-600 hover:underline"
        >
          Back to sign in
        </Link>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <p className="text-sm text-gray-600">
        Enter your email address and we&apos;ll send you a link to reset your
        password.
      </p>

      {formError && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
          {formError}
        </div>
      )}

      <div>
        <label htmlFor="email" className="block text-sm font-medium mb-1">
          Email
        </label>
        <input
          id="email"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          disabled={loading}
          className="w-full px-3 py-2 border rounded-lg disabled:opacity-50 disabled:cursor-not-allowed"
        />
      </div>

      <button
        type="submit"
        disabled={loading}
        className="w-full bg-blue-600 text-white py-2 rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {loading ? "Sending..." : "Send reset link"}
      </button>

      <p className="text-center text-sm">
        Remember your password?{" "}
        <Link href="/login" className="text-blue-600 hover:underline">
          Sign in
        </Link>
      </p>
    </form>
  );
};
