package main

const frontendTypesTemplate = `/** @format */

/**
 * {{.ResourceName}} Entity Types
 *
 * Auto-generated from table: {{.TableName}}
 */

import type { CrudEntity } from "@/shared/lib/crud";

export interface {{.ResourceName}} extends CrudEntity {
{{- range .Columns}}
  {{toCamel .Name}}{{if .Nullable}}?{{end}}: {{.TSType}}{{if .Nullable}} | null{{end}};
{{- end}}
}

export interface Create{{.ResourceName}}Input {
{{- range .Columns}}
{{- if not .IsPrimaryKey}}
{{- if not (or (eq .Name "created_at") (eq .Name "updated_at"))}}
  {{toCamel .Name}}?: {{.TSType}};
{{- end}}
{{- end}}
{{- end}}
}

export interface Update{{.ResourceName}}Input {
{{- range .Columns}}
{{- if not .IsPrimaryKey}}
{{- if not (or (eq .Name "created_at") (eq .Name "updated_at"))}}
  {{toCamel .Name}}?: {{.TSType}};
{{- end}}
{{- end}}
{{- end}}
}
`

const frontendGraphQLTemplate = `/** @format */

import { gql } from "@apollo/client";

/**
 * {{.ResourceName}} GraphQL Queries and Mutations
 * Auto-generated from table: {{.TableName}}
 */

{{$resourceUpper := toUpper .ResourceName -}}
{{$resourcePluralCamel := toCamel (pluralize .ResourceName) -}}
{{$resourcePluralUpper := toUpper (pluralize .ResourceName) -}}
{{$dollar := "$" -}}

// Fragment for {{.ResourceName}} fields
export const {{$resourceUpper}}_FRAGMENT = gql` + "`" + `
  fragment {{.ResourceName}}Fields on {{.ResourceName}} {
{{range .Columns}}    {{toCamel .Name}}
{{end}}  }
` + "`" + `;

// Get {{.ResourceName}} by ID
export const GET_{{$resourceUpper}}_QUERY = gql` + "`" + `
  {{$dollar}}{{"{"}}{{$resourceUpper}}_FRAGMENT}
  query Get{{.ResourceName}}($id: UUID!) {
    {{toCamel .ResourceName}}(id: $id) {
      ...{{.ResourceName}}Fields
    }
  }
` + "`" + `;

// Get all {{pluralize .ResourceName}}
export const GET_ALL_{{$resourcePluralUpper}}_QUERY = gql` + "`" + `
  {{$dollar}}{{"{"}}{{$resourceUpper}}_FRAGMENT}
  query GetAll{{pluralize .ResourceName}}(
    $scope: String
    $search: String
    $filters: JSONObject
    $first: Int
    $after: String
    $last: Int
    $before: String
  ) {
    {{$resourcePluralCamel}}(
      scope: $scope
      search: $search
      filters: $filters
      first: $first
      after: $after
      last: $last
      before: $before
    ) {
      edges {
        node {
          ...{{.ResourceName}}Fields
        }
        cursor
      }
      pageInfo {
        hasNextPage
        hasPreviousPage
        startCursor
        endCursor
      }
      totalCount
      scopes {
        id
        name
        count
      }
      filters {
        id
        name
        type
        options {
          value
          label
        }
        defaultValue
        placeholder
        min
        max
      }
    }
  }
` + "`" + `;

// Create {{.ResourceName}}
export const CREATE_{{$resourceUpper}}_MUTATION = gql` + "`" + `
  {{$dollar}}{{"{"}}{{$resourceUpper}}_FRAGMENT}
  mutation Create{{.ResourceName}}($input: Create{{.ResourceName}}Input!) {
    create{{.ResourceName}}(input: $input) {
      ...{{.ResourceName}}Fields
    }
  }
` + "`" + `;

// Update {{.ResourceName}}
export const UPDATE_{{$resourceUpper}}_MUTATION = gql` + "`" + `
  {{$dollar}}{{"{"}}{{$resourceUpper}}_FRAGMENT}
  mutation Update{{.ResourceName}}($id: {{if .IsUUID}}UUID{{else}}Int{{end}}!, $input: Update{{.ResourceName}}Input!) {
    update{{.ResourceName}}(id: $id, input: $input) {
      ...{{.ResourceName}}Fields
    }
  }
` + "`" + `;

// Delete {{.ResourceName}}
export const DELETE_{{$resourceUpper}}_MUTATION = gql` + "`" + `
  mutation Delete{{.ResourceName}}($id: {{if .IsUUID}}UUID{{else}}Int{{end}}!) {
    delete{{.ResourceName}}(id: $id)
  }
` + "`" + `;
`

const frontendListPageTemplate = `/** @format */

"use client";

import { {{.ResourceName}}Manager } from "@/entities/{{toKebab .ResourceName}}";

/**
 * {{pluralize .ResourceName}} Management Page
 *
 * Auto-generated CRUD interface for {{pluralize .ResourceName}}
 */
export default function {{pluralize .ResourceName}}Page() {
  return <{{.ResourceName}}Manager />;
}
`

const frontendShowPageTemplate = `/** @format */

"use client";

import { Crud } from "@/shared/ui/crud";
import { {{toCamel .ResourceName}}CrudConfig } from "@/entities/{{toKebab .ResourceName}}";
import { use } from "react";

/**
 * {{.ResourceName}} Detail Page
 */
export default function {{.ResourceName}}DetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);

  return (
    <div className="h-full">
      <Crud config={{"{"}}{{toCamel .ResourceName}}CrudConfig} mode="show" entityId={id} />
    </div>
  );
}
`

const frontendEditPageTemplate = `/** @format */

"use client";

import { Crud } from "@/shared/ui/crud";
import { {{toCamel .ResourceName}}CrudConfig } from "@/entities/{{toKebab .ResourceName}}";
import { use } from "react";

/**
 * Edit {{.ResourceName}} Page
 */
export default function Edit{{.ResourceName}}Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);

  return (
    <div className="h-full">
      <Crud config={{"{"}}{{toCamel .ResourceName}}CrudConfig} mode="edit" entityId={id} />
    </div>
  );
}
`

const frontendNewPageTemplate = `/** @format */

"use client";

import { Crud } from "@/shared/ui/crud";
import { {{toCamel .ResourceName}}CrudConfig } from "@/entities/{{toKebab .ResourceName}}";

/**
 * Create New {{.ResourceName}} Page
 */
export default function New{{.ResourceName}}Page() {
  return (
    <div className="h-full">
      <Crud config={{"{"}}{{toCamel .ResourceName}}CrudConfig} mode="new" />
    </div>
  );
}
`
