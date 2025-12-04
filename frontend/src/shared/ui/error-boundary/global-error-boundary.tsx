/**
 * Global Error Boundary
 *
 * Catches and handles React errors gracefully
 * Provides fallback UI and error reporting
 *
 * @format
 */

"use client";

import { Component, type ReactNode } from "react";
import Link from "next/link";
import { AlertCircle, RefreshCw01 } from "@untitledui/icons";
import { Button } from "@/components/base/buttons/button";
import { toast } from "@/shared/ui/toast";

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: React.ErrorInfo) => void;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: React.ErrorInfo | null;
}

export class GlobalErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
    };
  }

  static getDerivedStateFromError(error: Error): State {
    return {
      hasError: true,
      error,
      errorInfo: null,
    };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    // Log error to console
    console.error("Error Boundary caught:", error, errorInfo);

    // Call custom error handler if provided
    this.props.onError?.(error, errorInfo);

    // Show toast notification
    toast.error(
      "An unexpected error occurred. Please try refreshing the page."
    );

    // Update state with error info
    this.setState({
      error,
      errorInfo,
    });

    // TODO: Send error to error tracking service (Sentry, etc.)
    // sendErrorToService(error, errorInfo);
  }

  handleReset = () => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
    });
  };

  handleReload = () => {
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      // Custom fallback UI if provided
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // Default error UI
      return (
        <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4">
          <div className="w-full max-w-md space-y-6 rounded-xl border border-error-300 bg-white p-8 shadow-lg">
            {/* Error Icon */}
            <div className="flex justify-center">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-error-50">
                <AlertCircle className="h-6 w-6 text-error-600" />
              </div>
            </div>

            {/* Error Message */}
            <div className="space-y-2 text-center">
              <h1 className="text-xl font-semibold text-gray-900">
                Something went wrong
              </h1>
              <p className="text-sm text-gray-600">
                An unexpected error occurred. Please try refreshing the page.
              </p>
            </div>

            {/* Error Details (Development Only) */}
            {process.env.NODE_ENV === "development" && this.state.error && (
              <details className="rounded-lg bg-gray-50 p-4">
                <summary className="cursor-pointer text-sm font-medium text-gray-700">
                  Error Details
                </summary>
                <div className="mt-2 space-y-2">
                  <div>
                    <p className="text-xs font-semibold text-gray-600">
                      Error:
                    </p>
                    <pre className="mt-1 overflow-auto text-xs text-error-600">
                      {this.state.error.toString()}
                    </pre>
                  </div>
                  {this.state.errorInfo && (
                    <div>
                      <p className="text-xs font-semibold text-gray-600">
                        Component Stack:
                      </p>
                      <pre className="mt-1 max-h-40 overflow-auto text-xs text-gray-600">
                        {this.state.errorInfo.componentStack}
                      </pre>
                    </div>
                  )}
                </div>
              </details>
            )}

            {/* Action Buttons */}
            <div className="flex flex-col gap-3">
              <Button
                onClick={this.handleReload}
                color="primary"
                size="lg"
                className="w-full"
              >
                <RefreshCw01 className="h-5 w-5" />
                Reload Page
              </Button>
              <Button
                onClick={this.handleReset}
                color="secondary"
                size="lg"
                className="w-full"
              >
                Try Again
              </Button>
            </div>

            {/* Support Link */}
            <p className="text-center text-xs text-gray-500">
              If the problem persists, please{" "}
              <Link
                href="/support"
                className="text-brand-600 hover:text-brand-700 font-medium"
              >
                contact support
              </Link>
            </p>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
