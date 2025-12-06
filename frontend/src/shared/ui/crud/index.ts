/** @format */

export { Crud } from "./Crud";
export { CrudList } from "./CrudList";
export { CrudForm } from "./CrudForm";
export { CrudShow } from "./CrudShow";

// Re-export presentation components for custom views
export * from "./views";

// Re-export breadcrumb types for configuration
export type {
  CrudBreadcrumbItem,
  CrudBreadcrumbsConfig,
  CrudResourceGroup,
} from "@/shared/lib/crud/types";
