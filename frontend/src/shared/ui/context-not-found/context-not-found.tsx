/** @format */

"use client";

import { ArrowLeft, SearchLg } from "@untitledui/icons";

import { Button } from "@/components/base/buttons/button";
import { FeaturedIcon } from "@/components/foundations/featured-icon/featured-icon";
import { BackgroundPattern } from "@/components/shared-assets/background-patterns";

interface ContextNotFoundProps {
  /**
   * Error message from backend (e.g., "Organization not found", "Project not found")
   */
  message?: string;
  /**
   * Optional callback when "Go back" button is clicked
   * If not provided, uses browser's back navigation
   */
  onGoBack?: () => void;
  /**
   * Optional callback when "Go home" button is clicked
   * If not provided, uses window.location to navigate to root
   */
  onGoHome?: () => void;
}

/**
 * Context Not Found Component
 *
 * Displays when organization or project fails to load or is not found.
 * Automatically extracts the title from the error message.
 * Used in layout error states for better UX than simple error messages.
 *
 * Examples:
 * - "Organization not found" → title: "Organization not found"
 * - "Project not found" → title: "Project not found"
 */
export const ContextNotFound = ({
  message = "The resource you are looking for doesn't exist or you don't have access to it.",
  onGoBack,
  onGoHome,
}: ContextNotFoundProps) => {
  const handleGoBack = () => {
    if (onGoBack) {
      onGoBack();
    } else {
      window.history.back();
    }
  };

  const handleGoHome = () => {
    if (onGoHome) {
      onGoHome();
    } else {
      window.location.href = "/";
    }
  };

  // Extract title from error message or use generic title
  // "Organization not found" → title: "Organization not found", description: default
  // "Project not found" → title: "Project not found", description: default
  const getDisplayTexts = () => {
    const lowerMessage = message.toLowerCase();

    if (lowerMessage.includes("organization")) {
      return {
        title: "Organization not found",
        description:
          "The organization you are looking for doesn't exist or you don't have access to it.",
      };
    }

    if (lowerMessage.includes("project")) {
      return {
        title: "Project not found",
        description:
          "The project you are looking for doesn't exist or you don't have access to it.",
      };
    }

    // Generic fallback
    return {
      title: "Resource not found",
      description: message,
    };
  };

  const { title, description } = getDisplayTexts();

  return (
    <section className="flex min-h-screen items-center justify-center overflow-hidden bg-primary py-16 md:py-24">
      <div className="mx-auto w-full max-w-container grow px-4 md:px-8">
        <div className="mx-auto flex w-full max-w-5xl flex-col items-center gap-16 text-center">
          <div className="flex flex-col items-center justify-center gap-8 md:gap-12">
            <div className="z-10 flex flex-col items-center justify-center gap-4 md:gap-6">
              <div className="relative">
                <FeaturedIcon
                  color="gray"
                  theme="modern"
                  size="xl"
                  className="z-10 hidden md:flex"
                  icon={SearchLg}
                />
                <FeaturedIcon
                  color="gray"
                  theme="modern"
                  size="lg"
                  className="z-10 md:hidden"
                  icon={SearchLg}
                />

                <BackgroundPattern
                  pattern="grid"
                  className="absolute top-1/2 left-1/2 z-0 hidden -translate-x-1/2 -translate-y-1/2 md:block"
                />
                <BackgroundPattern
                  pattern="grid"
                  size="md"
                  className="absolute top-1/2 left-1/2 z-0 -translate-x-1/2 -translate-y-1/2 md:hidden"
                />
              </div>

              <h1 className="z-10 text-display-md font-semibold text-primary md:text-display-lg lg:text-display-xl">
                {title}
              </h1>
              <p className="z-10 text-lg text-tertiary md:text-xl">
                {description}
              </p>
            </div>

            <div className="z-10 flex flex-col-reverse gap-3 self-stretch md:flex-row md:self-auto">
              <Button
                iconLeading={ArrowLeft}
                color="secondary"
                size="xl"
                onClick={handleGoBack}
              >
                Go back
              </Button>
              <Button size="xl" onClick={handleGoHome}>
                Take me home
              </Button>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};
