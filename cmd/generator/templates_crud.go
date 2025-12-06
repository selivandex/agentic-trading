package main

const frontendCrudConfigTemplate = `/** @format */

import { z } from "zod";
import type { CrudConfig } from "@/shared/lib/crud";
import { Badge } from "@/components/base/badges/badges";
import type { {{.ResourceName}} } from "@/entities/{{toKebab .ResourceName}}";
import {
  GET_ALL_{{toUpper (pluralize .ResourceName)}}_QUERY,
  GET_{{toUpper .ResourceName}}_QUERY,
  CREATE_{{toUpper .ResourceName}}_MUTATION,
  UPDATE_{{toUpper .ResourceName}}_MUTATION,
} from "@/entities/{{toKebab .ResourceName}}";
{{range .FormFields}}{{if eq .Type "custom"}}// TODO: Import select field component for {{.Label}}
// import { {{trimSuffix (replace .Render "(props) => <" "") "SelectField {...props} />"}}SelectField } from "@/entities/...";
{{end}}{{end}}
/**
 * CRUD Configuration for {{.ResourceName}} entity
 * Auto-generated from table: {{.TableName}}
 */
export const {{toCamel .ResourceName}}CrudConfig: CrudConfig<{{.ResourceName}}> = {
  // Resource names
  resourceName: "{{.ResourceName}}",
  resourceNamePlural: "{{pluralize .ResourceName}}",

  // Base path for navigation
  basePath: "/{{toKebab (pluralize .ResourceName)}}",

  // Display name for breadcrumbs and titles
  getDisplayName: (entity) => entity.name,

  // GraphQL operations
  graphql: {
    list: {
      query: GET_ALL_{{toUpper (pluralize .ResourceName)}}_QUERY,
      dataPath: "{{toCamel (pluralize .ResourceName)}}",
      variables: {},
      useConnection: true,
    },
    show: {
      query: GET_{{toUpper .ResourceName}}_QUERY,
      variables: { id: {{if .IsUUID}}""{{else}}0{{end}} },
      dataPath: "{{toCamel .ResourceName}}",
    },
    create: {
      mutation: CREATE_{{toUpper .ResourceName}}_MUTATION,
      variables: {},
      dataPath: "create{{.ResourceName}}",
    },
    update: {
      mutation: UPDATE_{{toUpper .ResourceName}}_MUTATION,
      variables: {},
      dataPath: "update{{.ResourceName}}",
    },
    destroy: undefined,
  },

  // Table columns
  columns: [
{{range .DisplayColumns}}    {
      key: "{{.Key}}",
      label: "{{.Label}}",
      sortable: {{if .Sortable}}true{{else}}false{{end}},{{if .Width}}
      width: "{{.Width}}",{{end}}{{if .Render}}
      render: {{.Render}},{{end}}{{if .HideOnMobile}}
      hideOnMobile: true,{{end}}
    },
{{end}}  ],

  // Form fields
  formFields: [
{{range .FormFields}}    {
      name: "{{.Name}}",
      label: "{{.Label}}",
      type: "{{.Type}}",{{if .Placeholder}}
      placeholder: "{{.Placeholder}}",{{end}}{{if .HelperText}}
      helperText: "{{.HelperText}}",{{end}}
      validation: {{.Validation}},{{if .Options}}
      options: {{.Options}},{{end}}{{if .Render}}
      render: {{.Render}},{{end}}{{if .DefaultValue}}
      defaultValue: {{.DefaultValue}},{{end}}{{if .Disabled}}
      disabled: {{.Disabled}},{{end}}
      colSpan: {{.ColSpan}},
    },
{{end}}  ],

  // Custom actions (will be overridden in Manager)
  actions: [],
  bulkActions: [],

  // Tabs configuration
  tabs: {
    enabled: true,
    type: "underline",
    size: "md",
    filterVariable: "scope",
  },

  // Dynamic filters configuration
  dynamicFilters: {
    enabled: true,
  },

  // Feature flags
  enableSelection: true,
  enableSearch: true,
  defaultPageSize: 20,

  // Custom messages
  emptyStateMessage: "No {{toLower (pluralize .ResourceName)}} yet. Create your first {{toLower .ResourceName}} to get started.",
  errorMessage: "Failed to load {{toLower (pluralize .ResourceName)}}. Please try again.",
};
`

const frontendManagerTemplate = `/** @format */

"use client";

import { useMemo } from "react";
import { Crud } from "@/shared/ui/crud";
import { {{toCamel .ResourceName}}CrudConfig } from "@/entities/{{toKebab .ResourceName}}";

/**
 * {{.ResourceName}}Manager Component
 *
 * Enhanced Crud wrapper for {{pluralize .ResourceName}}
 * Auto-generated from table: {{.TableName}}
 */
export function {{.ResourceName}}Manager() {
  const enhancedConfig = useMemo(() => {
    return {
      ...{{toCamel .ResourceName}}CrudConfig,
      enableSelection: true,
    };
  }, []);

  return <Crud config={enhancedConfig} />;
}
`

const frontendIndexTemplate = `/** @format */

{{range .Exports}}export * from "./{{.}}";
{{end}}`
