/** @format */

/**
 * CRUD UI Components
 * Generic, reusable CRUD components for Next.js + GraphQL
 *
 * @example
 * ```tsx
 * import { Crud } from '@/shared/ui/crud';
 *
 * const strategiesConfig = {
 *   resourceName: 'Strategy',
 *   resourceNamePlural: 'Strategies',
 *   graphql: { ... },
 *   columns: [ ... ],
 *   formFields: [ ... ],
 * };
 *
 * function StrategiesPage() {
 *   return <Crud config={strategiesConfig} />;
 * }
 * ```
 */

export { Crud } from "./Crud";
export { CrudTable } from "./CrudTable";
export { CrudForm } from "./CrudForm";
export { CrudShow } from "./CrudShow";
