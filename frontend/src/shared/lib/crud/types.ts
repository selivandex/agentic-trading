/** @format */

import type { DocumentNode } from "graphql";
import type { ReactNode } from "react";
import type { ZodSchema } from "zod";

/**
 * Base CRUD entity must have an ID
 */
export interface CrudEntity {
  id: string;
  [key: string]: unknown;
}

/**
 * Column definition for table view
 */
export interface CrudColumn<TEntity extends CrudEntity> {
  /** Unique key for the column */
  key: string;
  /** Column header label */
  label: string;
  /** Optional tooltip for the column header */
  tooltip?: string;
  /** Render function for cell content */
  render?: (entity: TEntity) => ReactNode;
  /** Whether this column is sortable */
  sortable?: boolean;
  /** Column width (responsive or fixed) */
  width?: string | number;
  /** Whether to hide column on mobile */
  hideOnMobile?: boolean;
}

/**
 * Form field definition
 */
export interface CrudFormField<TEntity extends CrudEntity = CrudEntity> {
  /** Field name (must match entity property) */
  name: string;
  /** Field label */
  label: string;
  /** Field type */
  type:
    | "text"
    | "email"
    | "password"
    | "number"
    | "textarea"
    | "select"
    | "checkbox"
    | "radio"
    | "date"
    | "datetime"
    | "custom";
  /** Placeholder text */
  placeholder?: string;
  /** Help text below field */
  helperText?: string;
  /** Field validation schema */
  validation?: ZodSchema;
  /** Options for select/radio fields */
  options?: Array<{ label: string; value: string | number }>;
  /** Custom render function for complex fields */
  render?: (props: CrudFormFieldRenderProps<TEntity>) => ReactNode;
  /** Whether field is disabled */
  disabled?: boolean;
  /** Whether field is hidden */
  hidden?: boolean;
  /** Grid column span (1-12) */
  colSpan?: number;
  /** Default value for create form */
  defaultValue?: unknown;
}

/**
 * Props passed to custom field render function
 */
export interface CrudFormFieldRenderProps<_TEntity extends CrudEntity> {
  field: CrudFormField<_TEntity>;
  value: unknown;
  onChange: (value: unknown) => void;
  error?: string;
  disabled?: boolean;
}

/**
 * Relay PageInfo for cursor-based pagination
 */
export interface PageInfo {
  hasNextPage: boolean;
  hasPreviousPage: boolean;
  startCursor?: string | null;
  endCursor?: string | null;
}

/**
 * Relay Connection Edge
 */
export interface Edge<TEntity> {
  node: TEntity;
  cursor: string;
}

/**
 * Relay Connection
 */
export interface Connection<TEntity> {
  edges: Edge<TEntity>[];
  pageInfo: PageInfo;
  totalCount?: number;
}

/**
 * GraphQL operations configuration
 */
export interface CrudGraphQLConfig<
  _TEntity extends CrudEntity = CrudEntity,
  TListVariables = Record<string, unknown>,
  TShowVariables = { id: string },
  TCreateVariables = Record<string, unknown>,
  TUpdateVariables = Record<string, unknown>,
  TDeleteVariables = { id: string },
> {
  /** Query for fetching list of entities (Relay Connection) */
  list: {
    query: DocumentNode;
    variables?: TListVariables;
    /** Path to extract connection from response (e.g., "strategiesConnection") */
    dataPath: string;
    /** Whether this query uses Relay connections (default: true) */
    useConnection?: boolean;
  };
  /** Query for fetching single entity */
  show: {
    query: DocumentNode;
    variables: TShowVariables;
    dataPath: string;
  };
  /** Mutation for creating entity */
  create: {
    mutation: DocumentNode;
    variables: TCreateVariables;
    dataPath: string;
  };
  /** Mutation for updating entity */
  update: {
    mutation: DocumentNode;
    variables: TUpdateVariables;
    dataPath: string;
  };
  /** Mutation for deleting entity */
  destroy: {
    mutation: DocumentNode;
    variables: TDeleteVariables;
    dataPath: string;
  };
}

/**
 * Action button configuration
 */
export interface CrudAction<TEntity extends CrudEntity = CrudEntity> {
  /** Action key */
  key: string;
  /** Action label */
  label: string;
  /** Icon component (must be a React component, not JSX) */
  icon?: React.FC<{ className?: string }>;
  /** Action handler */
  onClick: (entity: TEntity) => void | Promise<void>;
  /** Whether action is destructive */
  destructive?: boolean;
  /** Whether action is disabled */
  disabled?: (entity: TEntity) => boolean;
  /** Whether action is hidden */
  hidden?: (entity: TEntity) => boolean;
}

/**
 * Filter configuration
 */
export interface CrudFilter {
  /** Filter key */
  key: string;
  /** Filter label */
  label: string;
  /** Filter type */
  type: "search" | "select" | "date" | "daterange" | "custom";
  /** Options for select filter */
  options?: Array<{ label: string; value: string | number }>;
  /** Placeholder text */
  placeholder?: string;
  /** Custom render function */
  render?: (props: {
    value: unknown;
    onChange: (value: unknown) => void;
  }) => ReactNode;
}

/**
 * Complete CRUD configuration
 */
export interface CrudConfig<TEntity extends CrudEntity = CrudEntity> {
  /** Resource name (singular) */
  resourceName: string;
  /** Resource name (plural) */
  resourceNamePlural: string;
  /** GraphQL operations */
  graphql: CrudGraphQLConfig<TEntity>;
  /** Column definitions for table */
  columns: CrudColumn<TEntity>[];
  /** Form fields for create/edit */
  formFields: CrudFormField<TEntity>[];
  /** Row actions in table */
  actions?: CrudAction<TEntity>[];
  /** Bulk actions for selected rows */
  bulkActions?: CrudAction<TEntity>[];
  /** Filters for list view */
  filters?: CrudFilter[];
  /** Custom empty state message */
  emptyStateMessage?: string;
  /** Custom error message */
  errorMessage?: string;
  /** Enable selection in table */
  enableSelection?: boolean;
  /** Enable search */
  enableSearch?: boolean;
  /** Default page size */
  defaultPageSize?: number;
  /** Transform entity before edit form */
  transformBeforeEdit?: (entity: TEntity) => Record<string, unknown>;
  /** Transform form data before create */
  transformBeforeCreate?: (data: Record<string, unknown>) => Record<string, unknown>;
  /** Transform form data before update */
  transformBeforeUpdate?: (data: Record<string, unknown>) => Record<string, unknown>;
  /** Custom validation schema for entire form */
  formValidationSchema?: ZodSchema;
}

/**
 * CRUD state
 */
export interface CrudState<TEntity extends CrudEntity = CrudEntity> {
  /** Current view mode */
  mode: "index" | "show" | "new" | "edit";
  /** Current entity (for show/edit) */
  currentEntity: TEntity | null;
  /** Selected entities (for bulk actions) */
  selectedEntities: TEntity[];
  /** Current filters */
  filters: Record<string, unknown>;
  /** Current page (1-indexed) */
  page: number;
  /** Page size */
  pageSize: number;
  /** Sort configuration */
  sort?: {
    column: string;
    direction: "asc" | "desc";
  };
  /** Search query */
  searchQuery?: string;
  /** Relay pagination cursors */
  cursors?: {
    after?: string | null;
    before?: string | null;
  };
  /** Total count of items */
  totalCount?: number;
  /** Page info from Relay */
  pageInfo?: PageInfo;
}

/**
 * CRUD actions
 */
export interface CrudActions<TEntity extends CrudEntity = CrudEntity> {
  /** Navigate to index view */
  goToIndex: () => void;
  /** Navigate to show view */
  goToShow: (id: string) => void;
  /** Navigate to new form */
  goToNew: () => void;
  /** Navigate to edit form */
  goToEdit: (id: string) => void;
  /** Create entity */
  create: (data: Record<string, unknown>) => Promise<void>;
  /** Update entity */
  update: (id: string, data: Record<string, unknown>) => Promise<void>;
  /** Delete entity */
  destroy: (id: string) => Promise<void>;
  /** Select entities */
  setSelectedEntities: (entities: TEntity[]) => void;
  /** Update filters */
  setFilters: (filters: Record<string, unknown>) => void;
  /** Update page */
  setPage: (page: number) => void;
  /** Update page size */
  setPageSize: (pageSize: number) => void;
  /** Update sort */
  setSort: (column: string, direction: "asc" | "desc") => void;
  /** Update search query */
  setSearchQuery: (query: string) => void;
  /** Refresh data */
  refresh: () => Promise<void>;
}
