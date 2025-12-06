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
 * Scope from backend (used for tabs)
 * Matches GraphQL Scope type from backend
 */
export interface Scope {
  /** Unique identifier for the scope */
  id: string;
  /** Human-readable name for the scope */
  name: string;
  /** Number of items that match this scope */
  count: number;
}

/**
 * Filter type from backend
 * Matches GraphQL FilterType enum
 */
export type FilterType =
  | "TEXT"
  | "NUMBER"
  | "DATE"
  | "SELECT"
  | "MULTISELECT"
  | "BOOLEAN"
  | "DATE_RANGE"
  | "NUMBER_RANGE";

/**
 * Filter option from backend
 * Matches GraphQL FilterOption type
 */
export interface FilterOption {
  /** Value of the option */
  value: string;
  /** Display label for the option */
  label: string;
}

/**
 * Filter metadata from backend
 * Matches GraphQL Filter type
 */
export interface FilterMetadata {
  /** Unique identifier for the filter */
  id: string;
  /** Human-readable name for the filter */
  name: string;
  /** Type of filter input */
  type: FilterType;
  /** Options for select/multiselect filters */
  options?: FilterOption[];
  /** Default value (optional) */
  defaultValue?: string;
  /** Placeholder text (optional) */
  placeholder?: string;
}

/**
 * Relay Connection
 */
export interface Connection<TEntity> {
  edges: Edge<TEntity>[];
  pageInfo: PageInfo;
  totalCount?: number;
  /** Available filter scopes with counts (optional) */
  scopes?: Scope[];
  /** Available dynamic filters (optional) */
  filters?: FilterMetadata[];
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
    /**
     * Scope variables that are always included in list queries
     * These are merged with pagination/filter/search variables
     * Example: { userID: "123" } to filter by user
     */
    scope?: Record<string, unknown> | (() => Record<string, unknown>);
  };
  /** Query for fetching single entity */
  show: {
    query: DocumentNode;
    variables: TShowVariables;
    dataPath: string;
    /**
     * Scope variables that are always included in show queries
     * Example: { userID: "123" } to ensure user owns the entity
     */
    scope?: Record<string, unknown> | (() => Record<string, unknown>);
  };
  /** Mutation for creating entity */
  create: {
    mutation: DocumentNode;
    variables: TCreateVariables;
    dataPath: string;
    /**
     * Scope variables that are always included in create mutations
     * Example: { userID: "123" } to auto-assign ownership
     */
    scope?: Record<string, unknown> | (() => Record<string, unknown>);
  };
  /** Mutation for updating entity */
  update: {
    mutation: DocumentNode;
    variables: TUpdateVariables;
    dataPath: string;
    /**
     * Scope variables that are always included in update mutations
     * Example: { userID: "123" } to ensure user owns the entity
     */
    scope?: Record<string, unknown> | (() => Record<string, unknown>);
  };
  /** Mutation for deleting entity (optional) */
  destroy?: {
    mutation: DocumentNode;
    variables: TDeleteVariables;
    dataPath: string;
    /**
     * Scope variables that are always included in destroy mutations
     * Example: { userID: "123" } to ensure user owns the entity
     */
    scope?: Record<string, unknown> | (() => Record<string, unknown>);
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
  /** Action handler for single entity (used in row actions and fallback for bulk) */
  onClick: (entity: TEntity) => void | Promise<void>;
  /** Batch action handler for multiple entities (optional, for bulk operations) */
  onBatchClick?: (entities: TEntity[]) => void | Promise<void>;
  /** Whether action is destructive */
  destructive?: boolean;
  /** Whether action is disabled */
  disabled?: (entity: TEntity) => boolean;
  /** Whether action is hidden for single entity */
  hidden?: (entity: TEntity) => boolean;
  /** Whether action is hidden for batch (receives all selected entities) */
  hiddenForBatch?: (entities: TEntity[]) => boolean;
}

/**
 * Breadcrumb item configuration
 */
export interface CrudBreadcrumbItem<TEntity extends CrudEntity = CrudEntity> {
  /** Breadcrumb label */
  label: string | ((entity?: TEntity) => string);
  /** Breadcrumb href (optional) */
  href?: string | ((entity?: TEntity) => string);
  /** Icon component (must be a React component, not JSX) */
  icon?: React.FC<{ className?: string }>;
  /** Custom onClick handler (overrides href navigation) */
  onClick?: (entity?: TEntity) => void;
}

/**
 * Resource group configuration for hierarchical organization
 */
export interface CrudResourceGroup {
  /** Group name (e.g., "Content", "Settings") */
  name: string;
  /** Group path (e.g., "/content") */
  path?: string;
  /** Icon component for the group */
  icon?: React.FC<{ className?: string }>;
  /** Parent group (for nested hierarchies) */
  parent?: CrudResourceGroup;
}

/**
 * Breadcrumbs configuration for different CRUD views
 */
export interface CrudBreadcrumbsConfig<TEntity extends CrudEntity = CrudEntity> {
  /** Show breadcrumbs */
  enabled?: boolean;
  /** Breadcrumb type style */
  type?: "text" | "text-line" | "button";
  /** Divider style */
  divider?: "chevron" | "slash";
  /** Maximum visible items before collapsing */
  maxVisibleItems?: number;
  /** Auto-generate breadcrumbs from resource group and base path (default: true) */
  autoGenerate?: boolean;
  /** Root breadcrumb items (always visible, e.g., Home) */
  rootItems?: CrudBreadcrumbItem<TEntity>[];
  /** Breadcrumbs for list view (overrides auto-generated) */
  list?: CrudBreadcrumbItem<TEntity>[];
  /** Breadcrumbs for show view (overrides auto-generated, receives entity) */
  show?: CrudBreadcrumbItem<TEntity>[];
  /** Breadcrumbs for new form view (overrides auto-generated) */
  new?: CrudBreadcrumbItem<TEntity>[];
  /** Breadcrumbs for edit form view (overrides auto-generated, receives entity) */
  edit?: CrudBreadcrumbItem<TEntity>[];
}

/**
 * Filter configuration (DEPRECATED - use dynamic filters from backend)
 * @deprecated Use dynamic filters from backend via Connection.filters
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
 * Dynamic filters configuration
 * Filters come from backend, this is just display config
 */
export interface CrudDynamicFiltersConfig {
  /** Whether dynamic filters are enabled */
  enabled: boolean;
  /** Custom render function for specific filter types (optional) */
  customRenderer?: (
    filter: FilterMetadata,
    value: unknown,
    onChange: (value: unknown) => void
  ) => ReactNode;
}

/**
 * Tabs configuration for list view
 * Tabs data comes from backend as scopes, this is just display config
 */
export interface CrudTabsConfig {
  /** Whether tabs are enabled */
  enabled: boolean;
  /** Tab style type */
  type?: "button-brand" | "button-gray" | "button-border" | "button-minimal" | "underline";
  /** Tab size */
  size?: "sm" | "md";
  /** Full width tabs */
  fullWidth?: boolean;
  /**
   * GraphQL variable name for active tab filter
   * This variable will be passed to list query with active scope id
   * Example: "status", "scope", "filter"
   * @default "scope"
   */
  filterVariable?: string;
  /**
   * Default scope ID to select (optional)
   * If not provided, first scope will be selected
   * Example: "all", "active"
   */
  defaultScope?: string;
  /**
   * Icon mapping function (optional)
   * Maps scope id to React icon component
   * Example: (scopeId) => scopeId === 'active' ? ActiveIcon : DefaultIcon
   */
  iconMapper?: (scopeId: string) => React.FC<{ className?: string }> | undefined;
}

/**
 * Complete CRUD configuration
 */
export interface CrudConfig<TEntity extends CrudEntity = CrudEntity> {
  /** Resource name (singular) */
  resourceName: string;
  /** Resource name (plural) */
  resourceNamePlural: string;
  /** Base path for navigation (e.g., "/strategies") */
  basePath?: string;
  /** Resource group for hierarchical organization (e.g., "Content", "Settings") */
  resourceGroup?: CrudResourceGroup;
  /** GraphQL operations */
  graphql: CrudGraphQLConfig<TEntity>;
  /** Column definitions for table */
  columns: CrudColumn<TEntity>[];
  /** Form fields for create/edit */
  formFields: CrudFormField<TEntity>[];
  /** Row actions in table (dropdown menu) */
  actions?: CrudAction<TEntity>[];
  /** Actions for show page header (buttons next to Edit) */
  showActions?: CrudAction<TEntity>[];
  /** Bulk actions for selected rows */
  bulkActions?: CrudAction<TEntity>[];
  /** Filters for list view (DEPRECATED - use dynamicFilters) */
  filters?: CrudFilter[];
  /**
   * Tabs configuration for list view (optional)
   * Tabs data (labels, counts) come from backend in list query response
   */
  tabs?: CrudTabsConfig;
  /**
   * Dynamic filters configuration (optional)
   * Filter definitions come from backend in list query response
   */
  dynamicFilters?: CrudDynamicFiltersConfig;
  /** Breadcrumbs configuration */
  breadcrumbs?: CrudBreadcrumbsConfig<TEntity>;
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
  /** Active tab key (if tabs are enabled) */
  activeTab?: string;
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
  /** Set active tab */
  setActiveTab: (tabKey: string) => void;
  /** Refresh data */
  refresh: () => Promise<void>;
  /** Update pagination state from query results */
  updatePaginationState: (pageInfo?: PageInfo, totalCount?: number) => void;
  /** Navigate to next page (Relay cursor pagination) */
  goToNextPage?: () => void;
  /** Navigate to previous page (Relay cursor pagination) */
  goToPrevPage?: () => void;
  /** Navigate to first page (Relay cursor pagination) */
  goToFirstPage?: () => void;
}
