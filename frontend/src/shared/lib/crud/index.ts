/** @format */

/**
 * CRUD Library
 * Type-safe, generic CRUD operations for Next.js + GraphQL + FSD
 *
 * @module shared/lib/crud
 *
 * Features:
 * - Type-safe CRUD operations
 * - GraphQL integration via Apollo Client
 * - React Hook Form + Zod validation
 * - Customizable columns, forms, and actions
 * - Built-in pagination, sorting, filtering
 * - Optimistic updates
 *
 * @example
 * ```tsx
 * import { Crud, type CrudConfig } from '@/shared/lib/crud';
 *
 * const config: CrudConfig<Strategy> = {
 *   resourceName: 'Strategy',
 *   resourceNamePlural: 'Strategies',
 *   graphql: {
 *     list: { query: GET_STRATEGIES, dataPath: 'strategies' },
 *     show: { query: GET_STRATEGY, dataPath: 'strategy' },
 *     create: { mutation: CREATE_STRATEGY, dataPath: 'createStrategy' },
 *     update: { mutation: UPDATE_STRATEGY, dataPath: 'updateStrategy' },
 *     destroy: { mutation: DELETE_STRATEGY, dataPath: 'deleteStrategy' },
 *   },
 *   columns: [
 *     { key: 'name', label: 'Name', sortable: true },
 *     { key: 'status', label: 'Status', render: (s) => <Badge>{s.status}</Badge> },
 *   ],
 *   formFields: [
 *     { name: 'name', label: 'Name', type: 'text', validation: z.string().min(1) },
 *     { name: 'description', label: 'Description', type: 'textarea' },
 *   ],
 * };
 *
 * function StrategiesPage() {
 *   return <Crud config={config} />;
 * }
 * ```
 */

// Types
export type {
  CrudEntity,
  CrudConfig,
  CrudColumn,
  CrudFormField,
  CrudFormFieldRenderProps,
  CrudGraphQLConfig,
  CrudAction,
  CrudFilter,
  CrudState,
  CrudActions,
} from "./types";

// Context and hooks
export { CrudProvider, useCrudContext } from "./context";
export { useCrudListQuery, useCrudShowQuery } from "./use-crud-query";
export { useCrudMutations } from "./use-crud-mutations";

// Utilities
export { get } from "./utils";
