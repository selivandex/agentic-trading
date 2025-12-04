/**
 * Loading Skeleton for Dynamically Loaded Components
 *
 * Provides consistent loading UI for code-split components.
 *
 * @format
 */

import { Skeleton } from "@/shared/ui/skeleton";

interface ComponentLoadingProps {
  height?: string;
  message?: string;
}

/**
 * Generic component loading skeleton
 */
export const ComponentLoading = ({
  height = "h-screen",
  message,
}: ComponentLoadingProps) => {
  return (
    <div className={`flex items-center justify-center ${height}`}>
      <div className="space-y-4 text-center">
        <Skeleton className="mx-auto h-12 w-12 rounded-full" />
        {message && (
          <p className="text-sm text-gray-500">{message}</p>
        )}
      </div>
    </div>
  );
};

/**
 * Editor loading skeleton (for ScenarioEditor)
 */
export const EditorLoading = () => {
  return (
    <div className="flex h-screen flex-col">
      <Skeleton className="h-16 w-full" /> {/* Header */}
      <div className="flex flex-1">
        <Skeleton className="h-full w-64" /> {/* Sidebar */}
        <Skeleton className="h-full flex-1" /> {/* Canvas */}
      </div>
    </div>
  );
};

/**
 * Layout loading skeleton (for ProjectLayout)
 */
export const LayoutLoading = () => {
  return (
    <div className="flex min-h-screen">
      <Skeleton className="h-screen w-64" /> {/* Sidebar */}
      <div className="flex-1 p-8">
        <Skeleton className="mb-6 h-12 w-64" /> {/* Page title */}
        <Skeleton className="h-96 w-full" /> {/* Content */}
      </div>
    </div>
  );
};

/**
 * Wizard loading skeleton (for OnboardingWizard)
 */
export const WizardLoading = () => {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="w-full max-w-2xl space-y-8 p-8">
        <Skeleton className="mx-auto h-20 w-20 rounded-full" />
        <Skeleton className="mx-auto h-8 w-64" />
        <Skeleton className="h-64 w-full rounded-lg" />
      </div>
    </div>
  );
};

