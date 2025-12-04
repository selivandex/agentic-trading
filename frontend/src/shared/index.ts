/**
 * Shared Layer - Public API
 * 
 * Reusable infrastructure code that doesn't depend on business logic.
 * See README.md for more information.
 */

// Base UI Components (UntitledUI)
export * from "./base";

// Content UI Components
export * from "./ui";

// Application Components
// Note: Import from specific subfolders to avoid naming conflicts
// Example: import { SidebarNavigationSimple } from "@/shared/application/app-navigation";
// export * from "./application";

// Foundation Components
export * from "./foundations";

// API (GraphQL, REST, Optimistic Updates)
export * from "./api";

// Library Wrappers (Contexts, Providers)
export * from "./lib";

// Utilities
export * from "./utils";

// Shared Assets (Illustrations, Background Patterns)
export * from "./shared-assets";

