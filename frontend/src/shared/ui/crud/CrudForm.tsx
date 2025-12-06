/** @format */

"use client";

import { useCallback, useEffect } from "react";
import {
  Controller,
  useForm,
  type Resolver,
  type FieldValues,
} from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Button } from "@/shared/base/buttons/button";
import { Input } from "@/shared/base/input/input";
import { TextArea } from "@/shared/base/textarea/textarea";
import { Select } from "@/shared/base/select/select";
import { Checkbox } from "@/shared/base/checkbox/checkbox";
import { useCrudContext } from "@/shared/lib/crud/context";
import { useCrudShowQuery } from "@/shared/lib/crud/use-crud-query";
import { useCrudMutations } from "@/shared/lib/crud/use-crud-mutations";
import { useCrudBreadcrumbs } from "@/shared/lib/crud/use-crud-breadcrumbs";
import { PageHeader, PageHeaderSkeleton } from "@/shared/ui/page-header";
import { PageFooter, PageFooterSkeleton } from "@/shared/ui/page-footer";
import { Skeleton } from "@/shared/ui/skeleton/skeleton";
import { logger } from "@/shared/lib";
import type { CrudEntity, CrudFormField } from "@/shared/lib/crud/types";

/**
 * CRUD Form Component
 * Handles both create and edit modes
 */
export function CrudForm<TEntity extends CrudEntity = CrudEntity>({
  entityId,
  mode,
}: {
  entityId?: string;
  mode: "new" | "edit";
}) {
  const { config, actions } = useCrudContext<TEntity>();

  // Fetch entity for edit mode
  const { entity, loading: entityLoading } = useCrudShowQuery<TEntity>(
    config,
    entityId ?? "",
    { skip: mode === "new" || !entityId }
  );

  // Mutations
  const {
    create,
    update,
    isLoading: mutationLoading,
  } = useCrudMutations<TEntity>(config);

  // Build validation schema from form fields (filtered by mode)
  const relevantFields =
    mode === "edit" && config.transformBeforeUpdate && entity
      ? // In edit mode, only validate fields that will be updated
        config.formFields.filter((field) => {
          const updateData = config.transformBeforeUpdate!(entity as TEntity);
          return field.name in updateData;
        })
      : config.formFields;

  const validationSchema =
    config.formValidationSchema ?? buildValidationSchema(relevantFields);

  // Initialize form
  const {
    control,
    handleSubmit,
    formState: { errors },
    reset,
  } = useForm<Record<string, unknown>>({
    resolver:
      createDynamicZodResolver<Record<string, unknown>>(validationSchema),
    defaultValues: buildDefaultValues(config.formFields),
  });

  // Log form errors when they change
  useEffect(() => {
    if (Object.keys(errors).length > 0) {
      logger.error("Form validation errors:", errors);
    }
  }, [errors]);

  // Load entity data into form (edit mode)
  useEffect(() => {
    if (mode === "edit" && entity) {
      const formData = config.transformBeforeEdit
        ? config.transformBeforeEdit(entity)
        : entity;

      reset(formData);
    }
  }, [mode, entity, config, reset]);

  // Breadcrumbs
  const breadcrumbs = useCrudBreadcrumbs<TEntity>(
    config,
    mode === "new" ? "new" : "edit",
    entity ?? undefined
  );

  // Handle form submission
  const onSubmit = useCallback(
    async (data: Record<string, unknown>) => {
      try {
        logger.info("Form submission started", { mode, entityId, data });

        if (mode === "new") {
          logger.info("Creating new entity");
          await create(data);
          actions.goToIndex();
        } else if (mode === "edit" && entityId) {
          logger.info("Updating entity", { entityId });
          await update(entityId, data);
          actions.goToIndex();
        }

        logger.info("Form submission completed successfully");
      } catch (error) {
        // Error is handled by mutation hook (toast.error)
        logger.error("Form submission error:", error);
      }
    },
    [mode, entityId, create, update, actions]
  );

  const paddingClasses = "px-4 lg:px-8";

  // Render loading state
  if (mode === "edit" && entityLoading) {
    return (
      <div className="flex flex-col h-full">
        {/* Page Header skeleton */}
        <PageHeaderSkeleton
          background="primary"
          showBreadcrumbs={!!breadcrumbs && breadcrumbs.length > 0}
          breadcrumbCount={breadcrumbs?.length ?? 0}
          showTitle
          actionCount={0}
        />

        {/* Scrollable content area */}
        <div className="flex-1 overflow-y-auto">
          <div className={`${paddingClasses} pb-8`}>
            <div className="rounded-xl border border-gray-200 bg-white p-8 shadow-sm">
              <div className="space-y-6">
                {Array.from({ length: 6 }).map((_, i) => (
                  <div key={i} className="space-y-2">
                    <Skeleton className="h-4 w-32" />
                    <Skeleton className="h-10 w-full" />
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* Page Footer skeleton */}
        <PageFooterSkeleton background="primary" actionCount={2} />
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      {/* Page Header with Breadcrumbs and Title */}
      <PageHeader
        background="primary"
        breadcrumbs={breadcrumbs}
        showBreadcrumbs={!!breadcrumbs && breadcrumbs.length > 0}
        title={`${mode === "new" ? "New" : "Edit"} ${config.resourceName}`}
        backHref={config.basePath}
        onBackClick={() => actions.goToIndex()}
      />

      {/* Form */}
      <form
        onSubmit={handleSubmit((data) =>
          onSubmit(data as Record<string, unknown>)
        )}
        className="flex flex-col flex-1"
      >
        {/* Scrollable content area */}
        <div className="flex-1 overflow-y-auto">
          <div className={`${paddingClasses} pb-8`}>
            <div className="rounded-xl border border-gray-200 bg-white p-8 shadow-sm">
              <div className="space-y-6">
                {config.formFields
                  .filter((field) => {
                    // Evaluate hidden based on mode
                    const isHidden =
                      typeof field.hidden === "function"
                        ? field.hidden(mode)
                        : field.hidden;
                    return !isHidden;
                  })
                  .map((field) => (
                    <div key={field.name}>
                      {renderFormField(field, control, errors, mode)}
                    </div>
                  ))}
              </div>
            </div>
          </div>
        </div>

        {/* Page Footer with Action Buttons - stays at bottom */}
        <PageFooter
          background="primary"
          actions={
            <>
              <Button
                color="secondary"
                size="md"
                onClick={() => actions.goToIndex()}
                isDisabled={mutationLoading}
                type="button"
              >
                Cancel
              </Button>
              <Button type="submit" size="md" isDisabled={mutationLoading}>
                {mutationLoading
                  ? "Saving..."
                  : mode === "new"
                  ? "Create"
                  : "Update"}
              </Button>
            </>
          }
        />
      </form>
    </div>
  );
}

/**
 * Typed wrapper for zodResolver that works with dynamic schemas
 *
 * Why this is architecturally correct:
 * - zodResolver has overly strict generic constraints requiring specific Zod types
 * - Dynamic schemas (built at runtime from config) can't satisfy these constraints
 * - Runtime validation still works perfectly - Zod validates correctly
 * - Double assertion (through unknown) is standard pattern for type narrowing
 */
function createDynamicZodResolver<TFieldValues extends FieldValues>(
  schema: z.ZodType
): Resolver<TFieldValues> {
  // Safe double assertion because:
  // 1. We control schema creation - it matches our form structure
  // 2. Runtime validation catches any data issues
  // 3. This is the recommended pattern for dynamic resolvers
  const typedZodResolver = zodResolver as unknown as (
    schema: z.ZodType
  ) => Resolver<TFieldValues>;
  return typedZodResolver(schema);
}

/**
 * Build Zod validation schema from form fields
 */
function buildValidationSchema(fields: CrudFormField[]) {
  const shape: Record<string, z.ZodTypeAny> = {};

  for (const field of fields) {
    if (field.validation) {
      shape[field.name] = field.validation;
    } else {
      // Default validations based on type
      switch (field.type) {
        case "email":
          shape[field.name] = z.string().email();
          break;
        case "number":
          shape[field.name] = z.number();
          break;
        case "checkbox":
          shape[field.name] = z.boolean();
          break;
        default:
          shape[field.name] = z.string();
      }
    }
  }

  return z.object(shape);
}

/**
 * Build default form values
 */
function buildDefaultValues(fields: CrudFormField[]): Record<string, unknown> {
  const defaults: Record<string, unknown> = {};

  for (const field of fields) {
    if (field.defaultValue !== undefined) {
      defaults[field.name] = field.defaultValue;
    } else {
      // Default values based on type
      switch (field.type) {
        case "checkbox":
          defaults[field.name] = false;
          break;
        case "number":
          defaults[field.name] = 0;
          break;
        default:
          defaults[field.name] = "";
      }
    }
  }

  return defaults;
}

/**
 * Render individual form field
 */
function renderFormField(
  field: CrudFormField,
  control: ReturnType<typeof useForm>["control"],
  errors: ReturnType<typeof useForm>["formState"]["errors"],
  mode: "new" | "edit"
) {
  const error = errors[field.name]?.message as string | undefined;

  // Evaluate disabled based on mode
  const isDisabled =
    typeof field.disabled === "function"
      ? field.disabled(mode)
      : field.disabled;

  return (
    <Controller
      name={field.name}
      control={control}
      render={({ field: { onChange, onBlur, value, ref } }) => {
        // Custom render function
        if (field.render) {
          const rendered = field.render({
            field,
            value,
            onChange,
            error,
            disabled: isDisabled,
          });
          return <>{rendered}</>;
        }

        // Standard field types
        switch (field.type) {
          case "textarea":
            return (
              <TextArea
                label={field.label}
                placeholder={field.placeholder}
                hint={field.helperText}
                isInvalid={!!error}
                isDisabled={isDisabled}
                value={value as string}
                onChange={onChange}
                onBlur={onBlur}
                textAreaRef={ref}
              />
            );

          case "select":
            return (
              <Select
                label={field.label}
                placeholder={field.placeholder}
                hint={field.helperText}
                isInvalid={!!error}
                isDisabled={isDisabled}
                selectedKey={value as string}
                onSelectionChange={onChange}
              >
                {field.options?.map((option) => (
                  <Select.Item key={option.value} id={String(option.value)}>
                    {option.label}
                  </Select.Item>
                ))}
              </Select>
            );

          case "checkbox":
            return (
              <div className="flex items-start gap-3 rounded-lg border border-gray-200 p-4">
                <Checkbox
                  isSelected={value as boolean}
                  onChange={onChange}
                  isDisabled={isDisabled}
                />
                <div className="flex-1">
                  <label className="text-sm font-medium text-gray-900">
                    {field.label}
                  </label>
                  {field.helperText && (
                    <p className="mt-1 text-sm text-gray-600">
                      {field.helperText}
                    </p>
                  )}
                  {error && (
                    <p className="mt-1 text-sm text-red-600">{error}</p>
                  )}
                </div>
              </div>
            );

          case "date":
          case "datetime":
            return (
              <Input
                type={field.type === "datetime" ? "datetime-local" : "date"}
                label={field.label}
                placeholder={field.placeholder}
                hint={field.helperText}
                isInvalid={!!error}
                isDisabled={isDisabled}
                value={value as string}
                onChange={onChange}
                onBlur={onBlur}
                ref={ref}
              />
            );

          case "number":
            return (
              <Input
                type="number"
                label={field.label}
                placeholder={field.placeholder}
                hint={field.helperText}
                isInvalid={!!error}
                isDisabled={isDisabled}
                value={value as string}
                onChange={(val) => onChange(parseFloat(val as string))}
                onBlur={onBlur}
                ref={ref}
              />
            );

          default:
            return (
              <Input
                type={field.type as string}
                label={field.label}
                placeholder={field.placeholder}
                hint={field.helperText}
                isInvalid={!!error}
                isDisabled={isDisabled}
                value={value as string}
                onChange={onChange}
                onBlur={onBlur}
                ref={ref}
              />
            );
        }
      }}
    />
  );
}
