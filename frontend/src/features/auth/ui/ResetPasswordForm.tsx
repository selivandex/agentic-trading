"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useResetPasswordMutation } from "@/shared/api/generated/graphql";

export const ResetPasswordForm = () => {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [resetPassword, { loading }] = useResetPasswordMutation();
  
  const token = searchParams.get("token");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [formError, setFormError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  // Show error if no token in URL
  if (!token) {
    return (
      <div className="space-y-4">
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          <h3 className="font-semibold mb-2">Invalid reset link</h3>
          <p className="text-sm">
            This password reset link is invalid or has expired. 
            Please request a new one.
          </p>
        </div>
        
        <Link
          href="/forgot-password"
          className="block text-center text-sm text-blue-600 hover:underline"
        >
          Request new reset link
        </Link>
      </div>
    );
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);

    // Validate password confirmation
    if (password !== confirmPassword) {
      setFormError("Passwords do not match");
      return;
    }

    // Validate password length
    if (password.length < 6) {
      setFormError("Password must be at least 6 characters long");
      return;
    }

    try {
      const { data } = await resetPassword({
        variables: {
          resetPasswordToken: token,
          password,
          passwordConfirmation: confirmPassword,
        },
      });

      if (data?.resetPassword?.id) {
        setSuccess(true);
        // Redirect to login after 2 seconds
        setTimeout(() => {
          router.push("/login");
        }, 2000);
      } else {
        throw new Error("Failed to reset password");
      }
    } catch (err) {
      console.error("Reset password error:", err);
      setFormError(
        err instanceof Error
          ? err.message
          : "Failed to reset password. The link may have expired."
      );
    }
  };

  if (success) {
    return (
      <div className="space-y-4">
        <div className="p-4 bg-green-50 border border-green-200 rounded-lg text-green-700">
          <h3 className="font-semibold mb-2">Password reset successful</h3>
          <p className="text-sm">
            Your password has been reset successfully. 
            Redirecting to sign in...
          </p>
        </div>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <p className="text-sm text-gray-600">
        Enter your new password below.
      </p>

      {formError && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
          {formError}
        </div>
      )}

      <div>
        <label htmlFor="password" className="block text-sm font-medium mb-1">
          New Password
        </label>
        <input
          id="password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          minLength={6}
          disabled={loading}
          className="w-full px-3 py-2 border rounded-lg disabled:opacity-50 disabled:cursor-not-allowed"
        />
        <p className="mt-1 text-xs text-gray-500">
          Minimum 6 characters
        </p>
      </div>
      
      <div>
        <label
          htmlFor="confirmPassword"
          className="block text-sm font-medium mb-1"
        >
          Confirm New Password
        </label>
        <input
          id="confirmPassword"
          type="password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
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
        {loading ? "Resetting..." : "Reset password"}
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
